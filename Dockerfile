# 1ый этап — берём базовый образ с компилятором Go
FROM golang:1.23 AS builder

# создаем рабочую дирректорию
WORKDIR /app


# Настраиваем прокси для более надежной загрузки зависимостей
ENV GOPROXY=https://proxy.golang.org,direct

# копируем и загружаем только файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# копируем весь код (включая уже сгенерированные GraphQL файлы)
COPY . .

# Обновляем go.mod и go.sum, чтобы убедиться, что все зависимости учтены
RUN go mod tidy

# Явно загружаем все зависимости снова после обновления
RUN go mod download

# Статическая сборка бинарника
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./cmd/server

# 2ой этап — финальный образ (используем минимальный образ)
FROM alpine:3.19

WORKDIR /app

# Добавляем SSL сертификаты для возможности HTTPS запросов
RUN apk --no-cache add ca-certificates

# Копируем только готовый бинарник из предыдущего этапа
COPY --from=builder /app/main .

# Создаем пустой .env файл, чтобы godotenv.Load() не выдавал ошибку
RUN touch .env

EXPOSE 8080

CMD ["./main"]