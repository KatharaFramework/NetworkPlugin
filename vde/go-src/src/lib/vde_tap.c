#define _GNU_SOURCE
#include "vde_tap.h"
#include <poll.h>
#include <errno.h>
#include <stdio.h>
#include <fcntl.h>
#include <stdlib.h>
#include <signal.h>
#include <unistd.h>
#include <net/if.h>
#include <string.h>
#include <pthread.h>
#include <sys/wait.h>
#include <sys/stat.h>
#include <sys/ioctl.h>
#include <sys/types.h>
#include <libvdeplug.h>
#include <sys/signalfd.h>
#include <linux/if_tun.h>

struct vde_tap {
    pthread_mutex_t mutex;
    int plugged;
    char *tap;
    char *url;
};

static int open_tap_fd(char *name) {
    struct ifreq *ifr = (struct ifreq *) malloc(sizeof(struct ifreq));
    int fd = -1;
    if ((fd = open("/dev/net/tun", O_RDWR | O_CLOEXEC)) < 0)
        return -1;

    memset(ifr, 0, sizeof(struct ifreq));
    ifr->ifr_flags = IFF_TAP | IFF_NO_PI;
    snprintf(ifr->ifr_name, sizeof(ifr->ifr_name), "%s", name);

    if (ioctl(fd, TUNSETIFF, (void *) ifr) < 0) {
        free(ifr);
        close(fd);
        return -1;
    }

    free(ifr);

    return fd;
}

void *plug_vde(void *arg) {
    int tapfd = 0;
    VDECONN *conn = NULL;
    struct vde_tap *tap_info = (struct vde_tap *) arg;
    if ((tapfd = open_tap_fd(tap_info->tap)) == -1)
        goto exit_failure;

    if ((conn = vde_open(tap_info->url, "kathara", NULL)) == NULL) {
        close(tapfd);
        goto exit_failure;
    }
    pthread_mutex_unlock(&tap_info->mutex);

    int n = 0, i = 0;
    sigset_t mask;
    char buf[VDE_ETHBUFSIZE];
    struct pollfd pfd[] = {{-1,    POLLIN, 0},
                           {tapfd, POLLIN, 0},
                           {-1,    POLLIN, 0}};
    sigemptyset(&mask);
    sigaddset(&mask, SIGUSR1);
    pthread_sigmask(SIG_BLOCK, &mask, NULL);
    pfd[0].fd = vde_datafd(conn);
    pfd[2].fd = signalfd(-1, &mask, SFD_CLOEXEC);
    while (ppoll(pfd, 3, NULL, &mask) >= 0) {
        if (pfd[0].revents & POLLIN) {
            n = vde_recv(conn, buf, VDE_ETHBUFSIZE, 0);
            if (n == 0)
                goto terminate;

            write(tapfd, buf, n);
        }

        if (pfd[1].revents & POLLIN) {
            n = read(tapfd, buf, VDE_ETHBUFSIZE);
            if (n == 0)
                goto terminate;

            vde_send(conn, buf, n, 0);
        }

        if (pfd[2].revents & POLLIN) {
            goto terminate;
        }
    }

    terminate:
    vde_close(conn);
    close(tapfd);
    pthread_exit(NULL);

    exit_failure:
    tap_info->plugged = -1;
    pthread_mutex_unlock(&tap_info->mutex);
    pthread_exit(NULL);
}

uintptr_t vde_tap_plug(char *name, char *sock) {
    pthread_t *th_ptr;
    struct vde_tap tap_info = {PTHREAD_MUTEX_INITIALIZER, 0, name, sock};

    if ((th_ptr = malloc(sizeof(pthread_t))) == NULL)
        return 0;

    pthread_mutex_lock(&tap_info.mutex);
    if (pthread_create(th_ptr, NULL, plug_vde, &tap_info) == 0) {
        pthread_mutex_lock(&tap_info.mutex);

        if (tap_info.plugged != 0) {
            pthread_join(*th_ptr, NULL);
            free(th_ptr);
            th_ptr = NULL;
        }
    } else {
        free(th_ptr);
        th_ptr = NULL;
    }

    pthread_mutex_unlock(&tap_info.mutex);
    pthread_mutex_destroy(&tap_info.mutex);
    return (uintptr_t) th_ptr;
}

void vde_tap_unplug(uintptr_t th_ptr) {
    if (th_ptr != 0) {
        pthread_kill(*((pthread_t *) th_ptr), SIGUSR1);
        pthread_join(*((pthread_t *) th_ptr), NULL);
    }

    free((pthread_t *) th_ptr);
}