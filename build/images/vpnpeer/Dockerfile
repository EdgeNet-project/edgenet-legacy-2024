FROM golang:1.21.3-alpine AS build

WORKDIR /edgenet

COPY go.mod .
RUN go mod download

COPY . ./
ENV CGO_ENABLED=0
RUN go build -o vpnpeer ./cmd/vpnpeer/

FROM alpine:3.18.4

USER root

WORKDIR /edgenet/vpnpeer/
COPY --from=build --chown=edgenet:edgenet /edgenet/vpnpeer ./

CMD ["./vpnpeer"]