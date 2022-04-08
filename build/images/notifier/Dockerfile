FROM golang:1.16.0-alpine AS builder

RUN apk update && \
    apk add git build-base && \
    rm -rf /var/cache/apk/* && \
    mkdir -p "$GOPATH/src/github.com/EdgeNet-project/edgenet"

ADD . "$GOPATH/src/github.com/EdgeNet-project/edgenet"

RUN cd "$GOPATH/src/github.com/EdgeNet-project/edgenet" && \
    CGO_ENABLED=0 go build -a -o /go/bin/notifier ./cmd/notifier/



FROM alpine:latest

WORKDIR /root/cmd/notifier/

COPY ./assets/templates/ /root/assets/templates/
COPY --from=builder /go/bin/notifier .

CMD ["./notifier"]
