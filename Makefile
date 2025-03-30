PROJECT_NAME := postery
CMD_DIR := ./cmd/server
BIN_DIR := ./bin

# Docker команды
.PHONY: docker-build docker-postgres docker-memory docker-stop

docker-build:
	@echo "Сборка Docker образов..."
	docker-compose build

docker-postgres:
	@echo "Запуск PostgreSQL версии в Docker..."
	docker-compose up -d app

docker-postgres-logs:
	@echo "Запуск PostgreSQL версии в Docker с логами..."
	docker-compose up app

docker-memory:
	@echo "Запуск in-memory версии в Docker..."
	docker-compose up -d memory

docker-memory-logs:
	@echo "Запуск in-memory версии в Docker c логами..."
	docker-compose up memory

docker-stop:
	@echo "Остановка Docker контейнеров..."
	docker-compose down

# Локальный запуск
.PHONY: run-postgres run-memory

run-postgres:
	@echo "Запуск с PostgreSQL..."
	go run $(CMD_DIR) --storage=postgres

run-memory:
	@echo "Запуск с in-memory хранилищем..."
	go run $(CMD_DIR) --storage=memory

# Тесты
.PHONY: test test-race test-clear
test:
	@echo "Подробный запуск тестов..."
	go test -v ./...

test-race:
	@echo "Подробный запуск тестов с проверкой на гонки данных..."
	go test -v -race ./...

test-clear:
	@echo "Запуск тестов..."
	go test ./...

.PHONY: help
help:
	@echo "Доступные команды:"
	@echo "  make docker-build  			 - Собрать Docker образы"
	@echo "  make docker-postgres 			 - Запустить PostgreSQL версию в Docker"
	@echo "  make docker-postgres-logs 		 - Запустить PostgreSQL версию в Docker с логами"
	@echo "  make docker-memory   			 - Запустить in-memory версию в Docker"
	@echo "  make docker-memory-logs  		 - Запустить in-memory версию в Docker с логами"
	@echo "  make docker-stop    			 - Остановить Docker контейнеры"
	@echo "  make run-postgres   			 - Запустить локально с PostgreSQL"
	@echo "  make run-memory     			 - Запустить локально с in-memory хранилищем"
	@echo "  make test-clear          		 - Запустить тесты c без флагов"
	@echo "  make test         			     - Запустить тесты c флагом -v"
	@echo "  make test-race         		 - Запустить тесты c флагами -v -race"


