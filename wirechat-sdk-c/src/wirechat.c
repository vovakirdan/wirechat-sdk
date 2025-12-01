#include "wirechat.h"

#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <libwebsockets.h>

#define JSMN_PARENT_LINKS
#define JSMN_IMPLEMENTATION
#include "jsmn.h"

#define WIRECHAT_PROTOCOL_VERSION 1
#define WIRECHAT_MAX_TOKENS 128

typedef struct outbound_msg {
    char *payload;
    struct outbound_msg *next;
} outbound_msg_t;

struct wirechat_client {
    struct lws_context *ctx;
    struct lws *wsi;
    wirechat_on_message_cb on_message;
    void *on_message_ud;
    wirechat_config_t cfg;
    pthread_t thread;
    int running;
    int hello_sent;
    pthread_mutex_t lock;
    outbound_msg_t *out_head;
    outbound_msg_t *out_tail;
};

static int lws_callback(struct lws *wsi, enum lws_callback_reasons reason, void *user, void *in, size_t len);
static void *service_thread(void *arg);
static int enqueue_outbound(wirechat_client_t *client, const char *payload);
static int send_text(struct lws *wsi, const char *text);
static char *make_hello_payload(const wirechat_client_t *client);
static char *make_join_payload(const char *room);
static char *make_msg_payload(const char *room, const char *text);
static char *escape_json(const char *src);
static void parse_inbound(wirechat_client_t *client, const char *text, size_t len);
static int parse_message_event(const char *json, size_t len, wirechat_message_event_t *out);
static char *substr_dup(const char *start, int len);
static const jsmntok_t *find_in_object(const char *json, const jsmntok_t *tokens, int count, int obj_index, const char *key);
static int next_token(const jsmntok_t *tokens, int count, int idx);

wirechat_client_t *wirechat_client_new(const wirechat_config_t *cfg) {
    wirechat_client_t *client = calloc(1, sizeof(wirechat_client_t));
    if (!client) {
        return NULL;
    }
    client->cfg.url = cfg && cfg->url ? strdup(cfg->url) : NULL;
    client->cfg.token = cfg && cfg->token ? strdup(cfg->token) : NULL;
    client->cfg.timeout_ms = cfg ? cfg->timeout_ms : 10000;
    pthread_mutex_init(&client->lock, NULL);
    return client;
}

void wirechat_client_free(wirechat_client_t *client) {
    if (!client) {
        return;
    }
    wirechat_client_close(client);
    free((char *)client->cfg.url);
    free((char *)client->cfg.token);
    pthread_mutex_destroy(&client->lock);
    free(client);
}

void wirechat_client_set_on_message(wirechat_client_t *client, wirechat_on_message_cb cb, void *user_data) {
    if (!client) {
        return;
    }
    client->on_message = cb;
    client->on_message_ud = user_data;
}

int wirechat_client_connect(wirechat_client_t *client) {
    if (!client || !client->cfg.url) {
        return -1;
    }
    if (client->running) {
        return -1;
    }

    struct lws_context_creation_info info;
    memset(&info, 0, sizeof(info));
    info.port = CONTEXT_PORT_NO_LISTEN;
    info.options = LWS_SERVER_OPTION_DO_SSL_GLOBAL_INIT;
    info.protocols = (struct lws_protocols[]){
        {
            .name = "wirechat",
            .callback = lws_callback,
            .per_session_data_size = 0,
            .user = client,
        },
        { NULL, NULL, 0, 0 },
    };

    client->ctx = lws_create_context(&info);
    if (!client->ctx) {
        return -1;
    }

    char uri_copy[512];
    strncpy(uri_copy, client->cfg.url, sizeof(uri_copy) - 1);
    uri_copy[sizeof(uri_copy) - 1] = '\0';

    const char *prot = NULL;
    const char *ads = NULL;
    int port = 0;
    const char *path = "/";
    if (lws_parse_uri(uri_copy, &prot, &ads, &port, &path)) {
        lws_context_destroy(client->ctx);
        client->ctx = NULL;
        return -1;
    }

    struct lws_client_connect_info ccinfo;
    memset(&ccinfo, 0, sizeof(ccinfo));
    ccinfo.context = client->ctx;
    ccinfo.address = ads;
    ccinfo.port = port;
    ccinfo.path = path;
    ccinfo.host = ads;
    ccinfo.origin = ads;
    ccinfo.ssl_connection = (strcmp(prot, "wss") == 0) ? LCCSCF_USE_SSL : 0;
    ccinfo.protocol = "wirechat";
    ccinfo.userdata = client;

    client->wsi = lws_client_connect_via_info(&ccinfo);
    if (!client->wsi) {
        lws_context_destroy(client->ctx);
        client->ctx = NULL;
        return -1;
    }
    lws_set_opaque_user_data(client->wsi, client);

    client->running = 1;
    client->hello_sent = 0;
    if (pthread_create(&client->thread, NULL, service_thread, client) != 0) {
        client->running = 0;
        lws_context_destroy(client->ctx);
        client->ctx = NULL;
        return -1;
    }
    return 0;
}

int wirechat_client_join(wirechat_client_t *client, const char *room) {
    if (!client || !client->running || !room) {
        return -1;
    }
    char *payload = make_join_payload(room);
    if (!payload) {
        return -1;
    }
    int rc = enqueue_outbound(client, payload);
    free(payload);
    return rc;
}

int wirechat_client_send(wirechat_client_t *client, const char *room, const char *text) {
    if (!client || !client->running || !room || !text) {
        return -1;
    }
    char *payload = make_msg_payload(room, text);
    if (!payload) {
        return -1;
    }
    int rc = enqueue_outbound(client, payload);
    free(payload);
    return rc;
}

int wirechat_client_close(wirechat_client_t *client) {
    if (!client) {
        return -1;
    }
    if (client->running) {
        client->running = 0;
        lws_cancel_service(client->ctx);
        pthread_join(client->thread, NULL);
    }

    pthread_mutex_lock(&client->lock);
    outbound_msg_t *msg = client->out_head;
    while (msg) {
        outbound_msg_t *next = msg->next;
        free(msg->payload);
        free(msg);
        msg = next;
    }
    client->out_head = client->out_tail = NULL;
    pthread_mutex_unlock(&client->lock);

    if (client->ctx) {
        lws_context_destroy(client->ctx);
        client->ctx = NULL;
    }
    client->wsi = NULL;
    return 0;
}

static void *service_thread(void *arg) {
    wirechat_client_t *client = (wirechat_client_t *)arg;
    while (client->running && client->ctx) {
        lws_service(client->ctx, 50);
    }
    return NULL;
}

static int lws_callback(struct lws *wsi, enum lws_callback_reasons reason, void *user, void *in, size_t len) {
    (void)user;
    wirechat_client_t *client = (wirechat_client_t *)lws_get_opaque_user_data(wsi);
    if (!client) {
        return 0;
    }
    switch (reason) {
        case LWS_CALLBACK_CLIENT_ESTABLISHED:
            lws_callback_on_writable(wsi);
            break;
        case LWS_CALLBACK_CLIENT_WRITEABLE: {
            if (!client->hello_sent) {
                char *hello = make_hello_payload(client);
                if (hello) {
                    send_text(wsi, hello);
                    free(hello);
                    client->hello_sent = 1;
                }
            }

            pthread_mutex_lock(&client->lock);
            outbound_msg_t *msg = client->out_head;
            if (msg) {
                client->out_head = msg->next;
                if (!client->out_head) {
                    client->out_tail = NULL;
                }
            }
            pthread_mutex_unlock(&client->lock);

            if (msg) {
                send_text(wsi, msg->payload);
                free(msg->payload);
                free(msg);
            }

            if (client->out_head) {
                lws_callback_on_writable(wsi);
            }
            break;
        }
        case LWS_CALLBACK_CLIENT_RECEIVE:
            parse_inbound(client, (const char *)in, len);
            break;
        case LWS_CALLBACK_CLIENT_CLOSED:
        case LWS_CALLBACK_CLIENT_CONNECTION_ERROR:
            client->running = 0;
            break;
        default:
            break;
    }
    return 0;
}

static int enqueue_outbound(wirechat_client_t *client, const char *payload) {
    outbound_msg_t *msg = calloc(1, sizeof(outbound_msg_t));
    if (!msg) {
        return -1;
    }
    msg->payload = strdup(payload);
    if (!msg->payload) {
        free(msg);
        return -1;
    }

    pthread_mutex_lock(&client->lock);
    if (client->out_tail) {
        client->out_tail->next = msg;
        client->out_tail = msg;
    } else {
        client->out_head = client->out_tail = msg;
    }
    pthread_mutex_unlock(&client->lock);

    lws_callback_on_writable(client->wsi);
    return 0;
}

static int send_text(struct lws *wsi, const char *text) {
    size_t len = strlen(text);
    size_t buf_len = LWS_PRE + len;
    unsigned char *buf = malloc(buf_len);
    if (!buf) {
        return -1;
    }
    memcpy(buf + LWS_PRE, text, len);
    int rc = lws_write(wsi, buf + LWS_PRE, len, LWS_WRITE_TEXT);
    free(buf);
    return rc < 0 ? -1 : 0;
}

static char *make_hello_payload(const wirechat_client_t *client) {
    const char *token = client->cfg.token ? client->cfg.token : "";
    char *escaped_token = escape_json(token);
    if (!escaped_token) {
        return NULL;
    }
    const char *fmt = "{\"type\":\"hello\",\"data\":{\"protocol\":%d,\"token\":\"%s\"}}";
    size_t needed = strlen(fmt) + strlen(escaped_token) + 16;
    char *buf = malloc(needed);
    if (!buf) {
        free(escaped_token);
        return NULL;
    }
    snprintf(buf, needed, fmt, WIRECHAT_PROTOCOL_VERSION, escaped_token);
    free(escaped_token);
    return buf;
}

static char *make_join_payload(const char *room) {
    char *room_esc = escape_json(room);
    if (!room_esc) {
        return NULL;
    }
    const char *fmt = "{\"type\":\"join\",\"data\":{\"room\":\"%s\"}}";
    size_t needed = strlen(fmt) + strlen(room_esc) + 4;
    char *buf = malloc(needed);
    if (!buf) {
        free(room_esc);
        return NULL;
    }
    snprintf(buf, needed, fmt, room_esc);
    free(room_esc);
    return buf;
}

static char *make_msg_payload(const char *room, const char *text) {
    char *room_esc = escape_json(room);
    char *text_esc = escape_json(text);
    if (!room_esc || !text_esc) {
        free(room_esc);
        free(text_esc);
        return NULL;
    }
    const char *fmt = "{\"type\":\"msg\",\"data\":{\"room\":\"%s\",\"text\":\"%s\"}}";
    size_t needed = strlen(fmt) + strlen(room_esc) + strlen(text_esc) + 4;
    char *buf = malloc(needed);
    if (!buf) {
        free(room_esc);
        free(text_esc);
        return NULL;
    }
    snprintf(buf, needed, fmt, room_esc, text_esc);
    free(room_esc);
    free(text_esc);
    return buf;
}

static char *escape_json(const char *src) {
    // Проверка на NULL
    if (!src) {
        return NULL;
    }
    
    size_t len = strlen(src);
    // Выделяем память с учетом худшего случая: каждый символ может стать \u00XX (6 символов)
    char *out = malloc(len * 6 + 1);
    if (!out) {
        return NULL;
    }
    
    char *p = out;
    for (size_t i = 0; i < len; i++) {
        unsigned char c = (unsigned char)src[i];
        
        // Обработка специальных символов
        if (c == '"') {
            *p++ = '\\';
            *p++ = '"';
        } else if (c == '\\') {
            *p++ = '\\';
            *p++ = '\\';
        } else if (c == '\b') {
            *p++ = '\\';
            *p++ = 'b';
        } else if (c == '\f') {
            *p++ = '\\';
            *p++ = 'f';
        } else if (c == '\n') {
            *p++ = '\\';
            *p++ = 'n';
        } else if (c == '\r') {
            *p++ = '\\';
            *p++ = 'r';
        } else if (c == '\t') {
            *p++ = '\\';
            *p++ = 't';
        } else if (c < 0x20) {
            // Остальные управляющие символы (U+0000..U+001F) как \u00XX
            *p++ = '\\';
            *p++ = 'u';
            *p++ = '0';
            *p++ = '0';
            *p++ = "0123456789ABCDEF"[c >> 4];
            *p++ = "0123456789ABCDEF"[c & 0x0F];
        } else {
            // Обычные символы копируем как есть
            *p++ = (char)c;
        }
    }
    *p = '\0';
    return out;
}

static void parse_inbound(wirechat_client_t *client, const char *text, size_t len) {
    wirechat_message_event_t ev;
    memset(&ev, 0, sizeof(ev));
    if (parse_message_event(text, len, &ev) == 0 && client->on_message) {
        client->on_message(client->on_message_ud, &ev);
    }
    free((char *)ev.room);
    free((char *)ev.user);
    free((char *)ev.text);
}

static int token_streq(const char *json, const jsmntok_t *tok, const char *s) {
    int len = tok->end - tok->start;
    return (int)strlen(s) == len && strncmp(json + tok->start, s, len) == 0;
}

static const jsmntok_t *find_key(const char *json, const jsmntok_t *tokens, int count, const char *key) {
    for (int i = 0; i < count - 1; i++) {
        if (tokens[i].type == JSMN_STRING && token_streq(json, &tokens[i], key)) {
            return &tokens[i + 1];
        }
    }
    return NULL;
}

static int parse_message_event(const char *json, size_t len, wirechat_message_event_t *out) {
    jsmn_parser parser;
    jsmntok_t tokens[WIRECHAT_MAX_TOKENS];
    jsmn_init(&parser);
    int r = jsmn_parse(&parser, json, len, tokens, WIRECHAT_MAX_TOKENS);
    if (r < 0) {
        return -1;
    }
    const jsmntok_t *type = find_key(json, tokens, r, "type");
    const jsmntok_t *event = find_key(json, tokens, r, "event");
    if (!type || !event) {
        return -1;
    }
    if (!token_streq(json, type, "event") || !token_streq(json, event, "message")) {
        return -1;
    }
    const jsmntok_t *data = find_key(json, tokens, r, "data");
    if (!data || data->type != JSMN_OBJECT) {
        return -1;
    }

    int data_index = (int)(data - tokens);
    const jsmntok_t *room = find_in_object(json, tokens, r, data_index, "room");
    const jsmntok_t *user = find_in_object(json, tokens, r, data_index, "user");
    const jsmntok_t *text = find_in_object(json, tokens, r, data_index, "text");
    const jsmntok_t *ts = find_in_object(json, tokens, r, data_index, "ts");

    if (room && room->type == JSMN_STRING) {
        out->room = substr_dup(json + room->start, room->end - room->start);
    }
    if (user && user->type == JSMN_STRING) {
        out->user = substr_dup(json + user->start, user->end - user->start);
    }
    if (text && text->type == JSMN_STRING) {
        out->text = substr_dup(json + text->start, text->end - text->start);
    }
    if (ts && ts->type == JSMN_PRIMITIVE) {
        char *ts_str = substr_dup(json + ts->start, ts->end - ts->start);
        if (ts_str) {
            out->timestamp = strtoll(ts_str, NULL, 10);
            free(ts_str);
        }
    }

    return (out->room && out->user && out->text) ? 0 : -1;
}

static char *substr_dup(const char *start, int len) {
    char *s = malloc((size_t)len + 1);
    if (!s) {
        return NULL;
    }
    memcpy(s, start, (size_t)len);
    s[len] = '\0';
    return s;
}

static const jsmntok_t *find_in_object(const char *json, const jsmntok_t *tokens, int count, int obj_index, const char *key) {
    int obj_end = tokens[obj_index].end;
    for (int i = obj_index + 1; i < count && tokens[i].start < obj_end; ) {
        if (tokens[i].type == JSMN_STRING && token_streq(json, &tokens[i], key)) {
            if (i + 1 < count && tokens[i + 1].start < obj_end) {
                return &tokens[i + 1];
            }
        }
        int value_idx = i + 1;
        int next = value_idx < count ? next_token(tokens, count, value_idx) : i + 1;
        i = next;
    }
    return NULL;
}

static int next_token(const jsmntok_t *tokens, int count, int idx) {
    int end = tokens[idx].end;
    int i = idx + 1;
    while (i < count && tokens[i].start < end) {
        end = tokens[i].end > end ? tokens[i].end : end;
        i++;
    }
    return i;
}
