include .env
export

export PROJECT_ROOT := $(shell pwd)

.DEFAULT_GOAL := help

GO_BIN := /usr/local/go/bin/go
BIN_DIR := $(PROJECT_ROOT)/bin

.PHONY: help build proto fmt lint test run-app run-worker \
        service-deploy service-undeploy service-logs \
        test-env-up hw1-test hw2-test

## ---- local run ------------------------------------------------------------

LOCAL_AMQP_URL ?= amqp://guest:guest@localhost:5672/

run-app run-worker: export AMQP_URL := $(LOCAL_AMQP_URL)
run-app run-worker: export DATA_DIR := $(PROJECT_ROOT)/data
run-app run-worker: export SWAGGER_DIR := $(PROJECT_ROOT)/docs
run-app run-worker: export GRPC_TARGET := localhost:9090
run-app run-worker: export WATERMARK_PATH := $(PROJECT_ROOT)/resources/watermark.png

run-app: ## Запустить API локально (RabbitMQ должен быть запущен)
	$(GO_BIN) run ./cmd/app

run-worker: ## Запустить worker локально (RabbitMQ и API должны быть запущены)
	$(GO_BIN) run ./cmd/worker

## ---- build / codegen -------------------------------------------------------

build: GO_BIN ?= go

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

service-undeploy: ## Остановить все сервисы
	docker compose down

service-logs: ## Смотреть логи всех сервисов в реальном времени
	docker compose logs -f

## ---- python tests ----------------------------

VENV := $(PROJECT_ROOT)/tests/venv
PYTHON := $(VENV)/bin/python3
PIP := $(VENV)/bin/pip
PYTEST := $(VENV)/bin/pytest

test-env-up: ## Создать python venv для тестов
	python3 -m venv $(VENV)
	$(PIP) install pytest requests

hw1-test: ## Прогнать тесты tests1.py
	$(PYTEST) $(PROJECT_ROOT)/tests/tests1.py -k "task" -v

hw2-test: ## Прогнать тесты tests2.py
	$(PYTEST) $(PROJECT_ROOT)/tests/tests2.py -v

## ---- help --------------------------------------------------------------

help: ## Показать справку по командам
	@echo "=== go-ffmpeg-pipeline: доступные команды ==="
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
