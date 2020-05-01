FROM golang:alpine AS builder

RUN apk update && \
    apk add git build-base && \
    rm -rf /var/cache/apk/* && \
    mkdir -p "$GOPATH/src/edgenet"

ADD . "$GOPATH/src/edgenet"

RUN cd "$GOPATH/src/edgenet" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /go/bin/nodelabeler ./cmd/nodelabeler/



FROM alpine:latest

WORKDIR /root/cmd/nodelabeler/

COPY --from=builder /go/bin/nodelabeler .

CMD ["./nodelabeler"]
