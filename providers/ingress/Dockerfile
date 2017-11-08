FROM alpine:3.6

RUN apk add --no-cache \
    bash \
    iptables

COPY ingress-controller-sidecar /root/ingress-controller-sidecar

ENTRYPOINT ["/root/ingress-controller-sidecar"]
