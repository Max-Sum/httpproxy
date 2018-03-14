FROM golang:alpine
MAINTAINER Max Sum <max@lolyculture.com>

# Build app
COPY . "$GOPATH/src/httpproxy"
WORKDIR $GOPATH/src/httpproxy
# Patch go library
RUN mv patch/request.diff $GOROOT/src/net/http \
    && cd $GOROOT/src/net/http \
    && patch < request.diff \
    && go install net
# Build server
RUN apk add --no-cache git \
    && mkdir /config \
    && mv $GOPATH/src/httpproxy/config/*.json /config \
    && go get -t httpproxy \
    && apk del git \
    && go build server.go

VOLUME /config

EXPOSE 80
CMD ["./server", "-c", "/config/config.json"]
