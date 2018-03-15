FROM golang:alpine
MAINTAINER Max Sum <max@lolyculture.com>

# Build app
COPY . "$GOPATH/src/httpproxy"
WORKDIR $GOPATH/src/httpproxy
# Build server
RUN apk add --no-cache git gettext \
    && go get -t httpproxy \
    && apk del git \
    && go build server.go

EXPOSE 80
CMD ["./server.sh"]
