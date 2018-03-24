FROM golang:alpine AS build-env
LABEL maintainer "Max Sum <max@lolyculture.com>"

# Build app
ADD . "$GOPATH/src/httpproxy"
WORKDIR $GOPATH/src/httpproxy
# Build server
RUN apk add --update git \
    && go get -t httpproxy \
    && go build -o /server server.go

# final stage
FROM alpine

COPY --from=build-env /server /app/
ADD ./server.sh /app/
ADD ./config/config.template /app/
ADD ./static /app/
ADD ./views /app/

RUN apk add --no-cache gettext \
    && chmod 777 /app

WORKDIR /app

CMD ["./server.sh"]