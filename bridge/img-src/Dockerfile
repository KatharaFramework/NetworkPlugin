FROM alpine:3.18

RUN apk add -U --no-cache \
    iptables \
    ip6tables \
    nftables \
    dpkg

WORKDIR /app

COPY entrypoint.sh entrypoint.sh
COPY katharanp katharanp
RUN update-alternatives --install /sbin/iptables iptables /sbin/iptables-legacy 1 && \
    update-alternatives --install /sbin/iptables iptables /sbin/iptables-nft 2 && \
    update-alternatives --install /sbin/ip6tables ip6tables /sbin/ip6tables-legacy 1 && \
    update-alternatives --install /sbin/ip6tables ip6tables /sbin/ip6tables-nft 2 && \
    chmod +x entrypoint.sh katharanp
