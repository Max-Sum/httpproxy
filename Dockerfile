FROM golang:alpine
MAINTAINER Max Sum <max@lolyculture.com>

# Build app
ADD . $GOPATH/src/httpproxy
WORKDIR $GOPATH/src/httpproxy

RUN go get -t httpproxy
RUN go build httpproxy

EXPOSE 8080
CMD ["$GOPATH/src/httpproxy/httpproxy"]
