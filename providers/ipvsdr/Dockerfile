FROM alpine:3.6

RUN apk add --no-cache \
    bash \
    keepalived \
    ipvsadm \
    iptables

COPY ipvsdr-provider /root/ipvsdr-provider
COPY keepalived.tmpl /root/keepalived.tmpl
COPY keepalived.conf /etc/keepalived/keepalived.conf

ENTRYPOINT ["/root/ipvsdr-provider"]
