FROM golang:alpine AS builder

RUN apk update && \
    apk add git build-base && \
    rm -rf /var/cache/apk/* && \
    mkdir -p "$GOPATH/src/headnode"

ADD . "$GOPATH/src/headnode"

RUN cd "$GOPATH/src/headnode" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /go/bin/headnode ./cmd/daemons/



FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/cmd/daemons/

COPY --from=builder /go/bin/headnode .

EXPOSE 8181

CMD ["./headnode"]
