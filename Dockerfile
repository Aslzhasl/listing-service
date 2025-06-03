# ─────────────────────────────────────────────────────────────────────────────
# Multi-stage Dockerfile для Go-based listing-service
# ─────────────────────────────────────────────────────────────────────────────

# 1. Build stage: собираем бинарь
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o listing-service ./

# 2. Runtime stage: минимальный образ для запуска
FROM alpine:3.17
RUN apk add --no-cache ca-certificates

WORKDIR /root/
COPY --from=builder /app/listing-service .

EXPOSE 8080

ENTRYPOINT ["./listing-service"]
