# syntax=docker/dockerfile:1.7-labs

# --- Build stage ---
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем приложение для amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /out/qms_speedtest_exporter ./cmd/app

# --- Runtime stage ---
FROM alpine:3

# Устанавливаем необходимые зависимости
RUN apk add --no-cache \
    ca-certificates \
    libc6-compat \
    jq \
    coreutils \
    wget

WORKDIR /app

# Копируем собранное приложение
COPY --from=builder /out/qms_speedtest_exporter /app/qms_speedtest_exporter

# Копируем бинарник qms_lib
COPY bin/qms_lib /app/bin/qms_lib

# Делаем файлы исполняемыми
RUN chmod +x /app/qms_speedtest_exporter /app/bin/qms_lib

# Создаем директорию для данных
RUN mkdir -p /app/data

EXPOSE 8080

ENTRYPOINT ["/app/qms_speedtest_exporter"]