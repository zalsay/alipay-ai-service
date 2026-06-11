FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .

RUN apk add --no-cache git bash make && \
    make build

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /app/dist/alipay-ai-service .
EXPOSE 8080
ENTRYPOINT ["./alipay-ai-service"]
