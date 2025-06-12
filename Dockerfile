# Используем официальный образ Go для сборки
FROM golang:1.21 AS builder

WORKDIR /app

# Копируем go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

# Финальный минимальный образ
FROM alpine:latest

WORKDIR /root/

# Копируем бинарник из builder
COPY --from=builder /app/app .

# Копируем example.env для примера (реальный .env не коммитим)
COPY example.env .

# Vercel будет предоставлять PORT через переменную окружения
ENV PORT=8080

# Запуск приложения
CMD ["./app"] 