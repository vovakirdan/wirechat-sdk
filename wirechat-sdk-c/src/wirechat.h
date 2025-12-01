#ifndef WIRECHAT_H
#define WIRECHAT_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct wirechat_client wirechat_client_t;

typedef struct {
    const char *url;
    const char *token;
    int timeout_ms;
} wirechat_config_t;

typedef struct {
    const char *room;
    const char *user;
    const char *text;
    int64_t timestamp;
} wirechat_message_event_t;

typedef void (*wirechat_on_message_cb)(void *user_data, const wirechat_message_event_t *ev);

wirechat_client_t *wirechat_client_new(const wirechat_config_t *cfg);
void wirechat_client_free(wirechat_client_t *client);

int wirechat_client_connect(wirechat_client_t *client);
int wirechat_client_join(wirechat_client_t *client, const char *room);
int wirechat_client_send(wirechat_client_t *client, const char *room, const char *text);
int wirechat_client_close(wirechat_client_t *client);

void wirechat_client_set_on_message(wirechat_client_t *client, wirechat_on_message_cb cb, void *user_data);

#ifdef __cplusplus
}
#endif

#endif /* WIRECHAT_H */
