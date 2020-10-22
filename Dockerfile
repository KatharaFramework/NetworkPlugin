FROM debian:buster

RUN apt update
RUN apt upgrade -y
RUN DEBIAN_FRONTEND=noninteractive apt install -y \
    bash \
    file \
    gcc \
    libnetfilter-conntrack-dev \
    make \
    git \
    autoconf \
    libtool && \
    cd "/" && \
    git clone "git://git.netfilter.org/iptables" && \
    cd "/iptables" && \
    git checkout "tags/v1.8.5" -b "v1.8.5" && \
    ./autogen.sh && \
    ./configure --prefix=/usr      \
                --sbindir=/sbin    \
                --disable-nftables \
                --enable-libipq    \
                --with-xt-lock-name=/tmp/xtables.lock \
                --libdir=/usr/lib64 \
                --with-xtlibdir=/lib/libxtables && \
    make && \
    make install && \
    cd "/" && \
    rm -Rf iptables && \
    DEBIAN_FRONTEND=noninteractive apt purge -y \
    file \
    gcc \
    libnetfilter-conntrack-dev \
    make \
    git \
    autoconf \
    libtool && \
    apt autoremove -y

WORKDIR /app

COPY entrypoint.sh entrypoint.sh
COPY katharanp katharanp
