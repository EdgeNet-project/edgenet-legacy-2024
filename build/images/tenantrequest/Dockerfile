FROM golang:1.18.3-alpine AS builder

RUN apk update && \
    apk add git build-base && \
    rm -rf /var/cache/apk/* && \
    mkdir -p "$GOPATH/src/github.com/EdgeNet-project/edgenet"

ADD . "$GOPATH/src/github.com/EdgeNet-project/edgenet"

RUN cd "$GOPATH/src/github.com/EdgeNet-project/edgenet" && \
    CGO_ENABLED=0 go build -a -o /go/bin/tenantrequest ./cmd/tenantrequest/



FROM alpine:latest

WORKDIR /root/cmd/tenantrequest/

COPY ./assets/templates/ /root/assets/templates/
COPY ./assets/certs/ /root/assets/certs/
COPY --from=builder /go/bin/tenantrequest .

CMD ["./tenantrequest"]
