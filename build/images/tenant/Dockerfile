FROM golang:1.18.3-alpine AS build

WORKDIR /edgenet

COPY go.mod .
RUN go mod download

COPY . ./
ENV CGO_ENABLED=0
RUN go build -o tenant ./cmd/tenant/

FROM alpine:3.16.2

RUN adduser -D -u 8118 edgenet
USER edgenet:edgenet

WORKDIR /edgenet/tenant/
COPY --from=build --chown=edgenet:edgenet /edgenet/tenant ./

CMD ["./tenant"]