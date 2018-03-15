FROM golang:alpine
MAINTAINER Max Sum <max@lolyculture.com>
# Default settings
ENV PROXY_LISTEN="tcp://:80" \
    WEB_LISTEN="tcp://:8080" \
    REVERSE_PROXY=false \
    REVERSE_PROXY_ADDR="" \
    PROXY_AUTHORIZATION=true \
    PROXY_FAILOVER_ADDR="" \
    LOG_LEVEL=0 \
    ADMIN_PASSWORD="proxy" \
    PROXY_USER="proxy" \
    PROXY_PASS="proxy"

# Build app
COPY . "$GOPATH/src/httpproxy"
WORKDIR $GOPATH/src/httpproxy
# Build server
RUN apk add --no-cache git gettext \
    && go get -t httpproxy \
    && apk del git \
    && go build server.go

EXPOSE 80
CMD envsubst < config/config.template > config.json && ./server
