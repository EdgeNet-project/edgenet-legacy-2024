FROM golang:alpine AS builder

RUN apk update && \
    apk add git build-base && \
    rm -rf /var/cache/apk/* && \
    mkdir -p "$GOPATH/src/edgenet"

ADD . "$GOPATH/src/edgenet"

RUN cd "$GOPATH/src/edgenet" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /go/bin/edgenet ./cmd/daemons/



FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/cmd/daemons/

COPY --from=builder /go/bin/edgenet .

EXPOSE 8181

CMD ["./edgenet"]
