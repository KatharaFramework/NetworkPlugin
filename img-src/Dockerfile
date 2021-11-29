FROM alpine

RUN apk add -U \
    iptables

WORKDIR /app

COPY entrypoint.sh entrypoint.sh
COPY katharanp katharanp
RUN chmod +x entrypoint.sh katharanp
