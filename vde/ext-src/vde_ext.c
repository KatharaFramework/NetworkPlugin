#define _GNU_SOURCE
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
#include <arpa/inet.h>
#include <string.h>
#include <linux/if_packet.h>
#include <net/if.h>
#include <linux/limits.h>

struct vde_ext {
    pthread_mutex_t mutex;
    int plugged;
    char *name;
    char *url;
};

static char pid_path[PATH_MAX];

static pthread_t *th_ptr;
void vde_ext_unplug() {
    if (th_ptr != NULL) {
        pthread_kill(*th_ptr, SIGUSR1);
    }

    free(th_ptr);
}

static void sig_handler(int sig)
{
    vde_ext_unplug();

    unlink(pid_path);

    signal(sig, SIG_DFL);
}

static void save_pidfile()
{
	int fd = open(pid_path, O_WRONLY | O_CREAT | O_EXCL, S_IRUSR | S_IWUSR | S_IRGRP | S_IROTH);
	FILE *f;
	if (fd == -1) {
        fprintf(stderr, "Error while opening pid fd.\n");
		return;
	}

	if ((f = fdopen(fd, "w")) == NULL) {
        fprintf(stderr, "Error while opening pid file.\n");
		return;
	}

	if (fprintf(f, "%ld\n", (long int) getpid()) <= 0) {
        fprintf(stderr, "Error while printing pid in file.\n");
		return;
	}

	fclose(f);
}

void *plug_vde(void *arg) {
    VDECONN *conn = NULL;
    struct vde_ext *ext_info = (struct vde_ext *) arg;

    struct ifreq ifr;	
	size_t if_name_len = strlen(ext_info->name);
	int sockfd = socket(PF_PACKET, SOCK_RAW, htons(ETH_P_ALL));
	if (sockfd < 0) {
        fprintf(stderr, "Failed to open socket for interface %s.\n", ext_info->name);
        goto exit_failure;
	}

	if (if_name_len < sizeof(ifr.ifr_name)) {
		memcpy(ifr.ifr_name, ext_info->name, if_name_len);
		ifr.ifr_name[if_name_len] = 0;
	} else {
        fprintf(stderr, "Interface name %s is longer than maximum allowed (%ld).\n", ext_info->name, sizeof(ifr.ifr_name));
		close(sockfd);
		goto exit_failure;
	}

	int res = ioctl(sockfd, SIOCGIFINDEX, &ifr);
    if (res != 0) {
        fprintf(stderr, "Unable to find interface index for name %s.\n", ext_info->name);
	    close(sockfd);
	    goto exit_failure;
    }

    struct sockaddr_ll sa;
    socklen_t fromlen = sizeof(sa);
	memset(&sa, 0, sizeof(sa));
    sa.sll_family = AF_PACKET;
    sa.sll_protocol = htons(ETH_P_ALL);
    sa.sll_ifindex = ifr.ifr_ifindex;

    fcntl(sockfd, F_SETFL, O_NONBLOCK);
	int ss = bind(sockfd, (const struct sockaddr *) &sa, sizeof(sa));
	if (ss < 0) {
        fprintf(stderr, "Unable to bind socket on interface index %d.\n", sa.sll_ifindex);
        close(sockfd);
        goto exit_failure;
    }

    if ((conn = vde_open(ext_info->url, "kathara", NULL)) == NULL) {
        close(sockfd);
        goto exit_failure;
    }

    pthread_mutex_unlock(&ext_info->mutex);

    int n = 0;
    sigset_t mask;
    char buf[VDE_ETHBUFSIZE];
    struct pollfd pfd[] = {{-1,    POLLIN, 0},
                           {sockfd, POLLIN, 0},
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

		    send(sockfd, buf, n, 0);
        }

        if (pfd[1].revents & POLLIN) {
            struct sockaddr_ll recv_addr;
	        socklen_t fromlen = sizeof(recv_addr);
            n = recvfrom(sockfd, buf, VDE_ETHBUFSIZE, MSG_TRUNC, (struct sockaddr *) &recv_addr, &fromlen);
            if (n == 0)
                goto terminate;

            if (recv_addr.sll_pkttype != PACKET_OUTGOING)
                vde_send(conn, buf, n, 0);
        }

        if (pfd[2].revents & POLLIN) {
            goto terminate;
        }
    }

    terminate:
    vde_close(conn);
    close(sockfd);
    pthread_exit(NULL);

    exit_failure:
    ext_info->plugged = -1;
    pthread_mutex_unlock(&ext_info->mutex);
    pthread_exit(NULL);
}

uintptr_t vde_ext_plug(char *name, char *sock) {
    struct vde_ext ext_info = {PTHREAD_MUTEX_INITIALIZER, 0, name, sock};

    if ((th_ptr = malloc(sizeof(pthread_t))) == NULL)
        return 0;

    pthread_mutex_lock(&ext_info.mutex);
    if (pthread_create(th_ptr, NULL, plug_vde, &ext_info) == 0) {
        pthread_mutex_lock(&ext_info.mutex);

        if (ext_info.plugged != 0) {
            pthread_join(*th_ptr, NULL);
            free(th_ptr);
            th_ptr = NULL;
        }
    } else {
        free(th_ptr);
        th_ptr = NULL;
    }

    pthread_mutex_unlock(&ext_info.mutex);
    pthread_mutex_destroy(&ext_info.mutex);
}

void usage(char **argv) {
    fprintf(stderr, "Usage: %s -s ctl_sock_path -p pid_path eth_name\n", argv[0]);
    exit(EXIT_FAILURE);
}

int main(int argc, char **argv) {
    int opt;
    char sock_path[PATH_MAX], eth_name[IFNAMSIZ];
    int daemonize = 0;
    while ((opt = getopt(argc, argv, "s:p:d")) != -1) {
        switch (opt) {
            case 's':
                strncpy(sock_path, optarg, PATH_MAX - 1);
                break;
            case 'p':
                strncpy(pid_path, optarg, PATH_MAX - 1);
                break;
            case 'd':
				daemonize = 1;
				break;
            default: /* '?' */
                usage(argv);
        }
    }

    if (optind < argc)
        strncpy(eth_name, argv[optind], IFNAMSIZ - 1);
	else
		usage(argv); 

    vde_ext_plug(eth_name, sock_path);
    if (th_ptr != NULL) {
        save_pidfile();

        signal(SIGINT, sig_handler);

        pthread_join(*th_ptr, NULL);
    } else {
        exit(EXIT_FAILURE);
    }

    return 0;
}