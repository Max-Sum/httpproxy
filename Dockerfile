FROM golang:alpine AS build-env
LABEL maintainer "Max Sum <max@lolyculture.com>"

# Build app
ADD . "$GOPATH/src/httpproxy"
WORKDIR $GOPATH/src/httpproxy
# Build server
RUN apk add --no-cache git gettext \
    && go get -t httpproxy \
    && apk del git \
    && go build server.go

# final stage
FROM scratch

COPY --from=build-env /go/src/httpproxy/server /
ADD ./server.sh /
ADD ./static /
ADD ./views /

CMD ["/server.sh"]