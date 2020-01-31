FROM golang:alpine AS builder

RUN apk update && \
    apk add git build-base && \
    rm -rf /var/cache/apk/* && \
    mkdir -p "$GOPATH/src/headnode"

ADD . "$GOPATH/src/headnode"

RUN cd "$GOPATH/src/headnode" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /go/bin/emailverification ./cmd/emailverification/



FROM alpine:latest

WORKDIR /root/cmd/emailverification/

COPY --from=builder /go/bin/emailverification .

CMD ["./emailverification"]
