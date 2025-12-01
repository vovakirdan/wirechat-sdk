#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include "../src/wirechat.h"

static volatile sig_atomic_t stop_flag = 0;

static void on_signal(int sig) {
    (void)sig;
    stop_flag = 1;
}

static void on_message(void *user_data, const wirechat_message_event_t *ev) {
    (void)user_data;
    printf("[%s] %s: %s\n", ev->room, ev->user, ev->text);
}

int main(void) {
    signal(SIGINT, on_signal);
    signal(SIGTERM, on_signal);

    wirechat_config_t cfg = {
        .url = "ws://localhost:8080/ws",
        .token = "",
        .timeout_ms = 5000,
    };

    wirechat_client_t *client = wirechat_client_new(&cfg);
    if (!client) {
        fprintf(stderr, "failed to create client\n");
        return 1;
    }
    wirechat_client_set_on_message(client, on_message, NULL);

    if (wirechat_client_connect(client) != 0) {
        fprintf(stderr, "connect failed\n");
        wirechat_client_free(client);
        return 1;
    }

    const char *room = "general";
    wirechat_client_join(client, room);
    printf("Connected. Type messages, /quit to exit.\n");

    char line[512];
    while (!stop_flag && fgets(line, sizeof(line), stdin)) {
        size_t n = strlen(line);
        if (n && (line[n - 1] == '\n' || line[n - 1] == '\r')) {
            line[n - 1] = '\0';
        }
        if (strcmp(line, "/quit") == 0) {
            break;
        }
        if (line[0] == '\0') {
            continue;
        }
        if (wirechat_client_send(client, room, line) != 0) {
            fprintf(stderr, "send failed\n");
        }
    }

    wirechat_client_close(client);
    wirechat_client_free(client);
    return 0;
}
