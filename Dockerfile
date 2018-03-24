FROM golang:alpine AS build-env
LABEL maintainer "Max Sum <max@lolyculture.com>"

# Build app
ADD . "$GOPATH/src/httpproxy"
WORKDIR $GOPATH/src/httpproxy
# Build server
RUN apk add --update git \
    && go get -t httpproxy \
    && go build server.go

# final stage
FROM alpine

COPY --from=build-env /go/src/httpproxy/server /
ADD ./server.sh /
ADD ./static /
ADD ./views /

RUN apk add --no-cache gettext

CMD ["/server.sh"]