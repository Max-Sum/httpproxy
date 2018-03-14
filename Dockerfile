FROM golang:alpine
MAINTAINER Max Sum <max@lolyculture.com>

# Build app
RUN mkdir $GOPATH/src/httpproxy
COPY ["cache", "config", "lib", "proxy", "static", "views", "server.go", "$GOPATH/src/httpproxy"]
WORKDIR $GOPATH/src/httpproxy
VOLUME $GOPATH/src/httpproxy/config

RUN go get -t httpproxy
RUN go build server.go

EXPOSE 80
CMD ["$GOPATH/src/httpproxy/server"]
