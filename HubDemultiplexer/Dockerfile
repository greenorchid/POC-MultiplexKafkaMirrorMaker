FROM golang:1.21.6-alpine AS builder
WORKDIR /app

COPY . .


RUN apk --no-cache update && \
apk --no-cache add git gcc libc-dev

# Kafka Go client is based on the C library librdkafka
ENV CGO_ENABLED 1
ENV GOFLAGS -mod=vendor
ENV GOOS=linux
ENV GOARCH=amd64

# build
RUN go build -tags musl -o HubDemultiplexer

# multistage build, run in alpine
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/HubDemultiplexer .

CMD ["./HubDemultiplexer"]
