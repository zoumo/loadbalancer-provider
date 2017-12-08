FROM alpine:3.6

RUN apk add --no-cache \
    bash \
    keepalived \
    ipvsadm \
    iptables

COPY bin/ipvsdr /root/ipvsdr
COPY build/ipvsdr/keepalived.tmpl /root/keepalived.tmpl
COPY build/ipvsdr/keepalived.conf /etc/keepalived/keepalived.conf

ENTRYPOINT ["/root/ipvsdr"]
