FROM debian:buster

RUN apt update
RUN apt upgrade -y
RUN DEBIAN_FRONTEND=noninteractive apt install -y \
    bash \
    curl \
    iptables

WORKDIR /app

COPY entrypoint.sh entrypoint.sh
COPY katharanp katharanp
