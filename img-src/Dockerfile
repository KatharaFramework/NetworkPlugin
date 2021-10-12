FROM debian:buster

RUN apt update
RUN apt upgrade -y
RUN DEBIAN_FRONTEND=noninteractive apt install -y \
    bash \
    iptables
RUN apt clean

WORKDIR /app

COPY entrypoint.sh entrypoint.sh
COPY katharanp katharanp