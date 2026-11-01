include .env
export

export PROJECT_ROOT := $(shell pwd)

.DEFAULT_GOAL := help

GO_BIN := /usr/local/go/bin/go
BIN_DIR := $(PROJECT_ROOT)/bin

.PHONY: help build proto fmt lint test run-app run-worker \
        service-deploy service-undeploy service-logs \
		env-up env-down env-cleanup env-port-forward env-port-close \
		swagger-gen broker-up broker-down \
		migrate-create migrate-up migrate-down migrate-action \

## ---- local run ------------------------------------------------------------

LOCAL_AMQP_URL ?= amqp://guest:guest@localhost:5672/

run-app run-worker: export AMQP_URL := $(LOCAL_AMQP_URL)
run-app run-worker: export DATA_DIR := $(PROJECT_ROOT)/data
run-app run-worker: export SWAGGER_DIR := $(PROJECT_ROOT)/docs
run-app run-worker: export GRPC_TARGET := localhost:9090
run-app run-worker: export WATERMARK_PATH := $(PROJECT_ROOT)/resources/watermark.png
run-app: export POSTGRES_HOST := localhost

.PHONY: local-env-up local-env-down

local-env-up: ## Подготовить окружение для локального запуска
	$(MAKE) broker-up
	$(MAKE) env-up
	$(MAKE) migrate-up
	$(MAKE) env-port-forward

local-env-down: ## Остановить окружение локального запуска
	$(MAKE) env-port-close
	$(MAKE) broker-down
	$(MAKE) env-down

run-app: local-env-up ## Запустить API локально (PostgreSQL с применёнными миграциями и RabbitMQ должны быть запущены)
	$(GO_BIN) run ./cmd/app

run-worker: ## Запустить worker локально (RabbitMQ и API должны быть запущены)
	$(GO_BIN) run ./cmd/worker

## ---- build / codegen -------------------------------------------------------
build: ## Собрать бинарники app и worker в bin/
	$(GO_BIN) build -o $(BIN_DIR)/app ./cmd/app
	$(GO_BIN) build -o $(BIN_DIR)/worker ./cmd/worker

# requires: protoc, protoc-gen-go, protoc-gen-go-grpc
# install:  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#           go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
proto: ## Перегенерировать gRPC-код из .proto
	protoc \
	--go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	internal/grpc/taskpb/task.proto

## ---- quality ---------------------------------------------------------------

fmt: ## Отформатировать Go-файлы проекта
	gofmt -w .

lint: ## Прогнать go vet и проверку форматирования
	$(GO_BIN) vet ./...
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)

test: ## Прогнать Go-тесты
	$(GO_BIN) test ./... -v

## ---- docker ------------------------------------------------------------

IMAGE_NAME := media-pipeline-app
CONTAINER_NAME := media-pipeline

service-deploy: ## Собрать образы и поднять все сервисы (app, worker, rabbitmq)
	docker compose up -d --build

service-undeploy: ## Остановить и удалить контейнеры проекта
	docker compose --profile db-access down

service-logs: ## Смотреть логи всех сервисов в реальном времени
	docker compose logs -f

env-up: ## Запустить окружение проекта (PostgreSQL)
	@docker compose up -d postgres

env-down: ## Остановить PostgreSQL
	@docker compose stop postgres

env-cleanup: ## env: Очистить окружение проекта
	@read -p "Очистить все volume файлы окружения? Опасность утери данных. [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		docker compose down postgres && \
		rm -rf ${PROJECT_ROOT}/out/pgdata && \
		echo "Файлы окружения очищены"; \
	else \
		echo "Очистка окружения отменена"; \
	fi

env-port-forward: ## Открыть порты сервисов окружения
	@docker compose up -d port-forwarder

env-port-close: ## Остановить доступ к PostgreSQL с хоста
	@docker compose stop port-forwarder

swagger-gen: ## Сгенерировать Swagger-документацию через Docker
	@docker compose run --rm --build \
		--user "$(shell id -u):$(shell id -g)" \
		swagger init \
		-g cmd/app/main.go \
		-o docs \
		--parseInternal

broker-up: ## Запустить очередь сообщений RabbitMQ
	@docker compose up -d queue

broker-down: ## Остановить RabbitMQ
	@docker compose stop queue

migrate-create: ## Создать новую версию схемы данных
	@if [ -z "$(seq)" ]; then \
		echo "Отсутсвует необходимый параметр seq. Пример: make migrate-create seq=init"; \
		exit 1; \
	fi; \
	docker compose run --rm postgres-migrate \
		create \
		-ext sql \
		-dir /migrations \
		-seq "$(seq)"

migrate-up: ## Накатить миграции
	@$(MAKE) migrate-action action=up

migrate-down: ## Откатить миграции
	@$(MAKE) migrate-action action=down

migrate-action:
	@if [ -z "$(action)" ]; then \
		echo "Отсутсвует необходимый параметр action. Пример: make migrate-action action=up"; \
		exit 1; \
	fi; \
	docker compose run --rm postgres-migrate \
		-path /migrations \
		-database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable" \
		"$(action)"

## ---- help --------------------------------------------------------------

help: ## Показать справку по командам
	@echo "=== go-ffmpeg-pipeline: доступные команды ==="
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
