FROM debian:buster

RUN apt update
RUN apt upgrade -y
RUN DEBIAN_FRONTEND=noninteractive apt install -y \
    bash \
    wget \
    git \
    make \
    pkg-config \
    libgmp3-dev \
    file \
    gcc \
    libnetfilter-conntrack-dev \
    autoconf \
    libtool \
    libmnl-dev && \
    wget http://ftp.br.debian.org/debian/pool/main/i/iptables/iptables_1.6.0+snapshot20161117-6_amd64.deb && \
    wget http://ftp.br.debian.org/debian/pool/main/i/iptables/libip4tc0_1.6.0+snapshot20161117-6_amd64.deb && \
    wget http://ftp.br.debian.org/debian/pool/main/i/iptables/libip6tc0_1.6.0+snapshot20161117-6_amd64.deb && \
    wget http://ftp.br.debian.org/debian/pool/main/i/iptables/libiptc0_1.6.0+snapshot20161117-6_amd64.deb && \
    wget http://ftp.br.debian.org/debian/pool/main/i/iptables/libxtables12_1.6.0+snapshot20161117-6_amd64.deb && \
    apt install ./lib*.deb -y  --allow-downgrades && \
    apt install ./iptables_1.6.0+snapshot20161117-6_amd64.deb -y --allow-downgrades && \
    apt-mark hold iptables && \
    git clone git://git.netfilter.org/libnftnl && \
    cd /libnftnl && \
    ./autogen.sh && \
    ./configure && \
    make && \
    make install && \
    cd / && \
    rm -Rf /libnftnl && \
    git clone git://git.netfilter.org/iptables && \
    cd /iptables && \
    git checkout "tags/v1.8.5" -b "v1.8.5" && \
    ./autogen.sh && \
    ./configure --prefix=/usr      \
                --enable-libipq    \
                --libdir=/usr/lib64 \
                --with-xtlibdir=/lib/libxtables  && \
    make && \
    make install && \
    cd /  && \
    rm -Rf /iptables && \
    rm -r *.deb && \
    DEBIAN_FRONTEND=noninteractive apt remove -y \
        wget \
        git \
        make \
        pkg-config \
        libgmp3-dev \
        file \
        gcc \
        libnetfilter-conntrack-dev \
        autoconf \
        libtool \
        libmnl-dev && \
    DEBIAN_FRONTEND=noninteractive apt autoremove -y

WORKDIR /app

COPY entrypoint.sh entrypoint.sh
COPY katharanp katharanp