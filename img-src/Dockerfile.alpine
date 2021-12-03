FROM alpine

RUN apk add -U --no-cache \
    iptables \
    nftables \
    dpkg

WORKDIR /app

COPY entrypoint.sh entrypoint.sh
COPY katharanp katharanp
RUN update-alternatives --install /sbin/iptables iptables /sbin/iptables-legacy 1 && \
    update-alternatives --install /sbin/iptables iptables /sbin/iptables-nft 2 && \
    chmod +x entrypoint.sh katharanp
