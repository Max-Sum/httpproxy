FROM golang:alpine
MAINTAINER Max Sum <max@lolyculture.com>

# Build app
COPY . "$GOPATH/src/httpproxy"
WORKDIR $GOPATH/src/httpproxy
VOLUME $GOPATH/src/httpproxy/cnfg

RUN apk add --no-cache git \
    && go get -t httpproxy \
    && go build server.go

EXPOSE 80
CMD ["$GOPATH/src/httpproxy/server"]
