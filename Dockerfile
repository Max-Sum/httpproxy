FROM golang:alpine
MAINTAINER Max Sum <max@lolyculture.com>

# Build app
COPY . "$GOPATH/src/httpproxy"
WORKDIR $GOPATH/src/httpproxy

RUN apk add --no-cache git \
    && mkdir /config \
    && mv $GOPATH/src/httpproxy/config/*.json /config \
    && go get -t httpproxy \
    && go build server.go

VOLUME /config

EXPOSE 80
CMD ["$GOPATH/src/httpproxy/server", "-c", "/config/config.json"]
