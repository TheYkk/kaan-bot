FROM golang:1.14.6-alpine AS builder
RUN set -ex \
    && apk add --no-cache  git
WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN ./ci/build

FROM alpine:3.12
EXPOSE 8181

RUN set -ex \
    && apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /app/dist/kaan-bot /app/kaan-bot

CMD ["./kaan-bot"]
