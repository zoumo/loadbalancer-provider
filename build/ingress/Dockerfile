FROM alpine:3.6

RUN apk add --no-cache \
    bash \
    iptables

COPY bin/ingress /root/ingress

ENTRYPOINT ["/root/ingress"]
