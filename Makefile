export PROJECT_ROOT=$(shell pwd)

VENV = ${PROJECT_ROOT}/tests/venv
PYTHON = $(VENV)/bin/python3
PIP = $(VENV)/bin/pip
PYTEST = $(VENV)/bin/pytest

.PHONY: test

test-env-up:
	python3 -m venv $(VENV)
#   $(PIP) install pytest requests

hw1-test:
	$(PYTEST) ${PROJECT_ROOT}/tests/tests1.py -k "task" -v

hw2-test:
	$(PYTEST) ${PROJECT_ROOT}/tests/tests2.py -v

IMAGE_NAME = media-pipeline-app
CONTAINER_NAME = media-pipeline

docker-build:
	docker build -f ${PROJECT_ROOT}/cmd/app/Dockerfile -t $(IMAGE_NAME) .

docker-run:
	docker run --env-file .env -p 8000:8000 --name $(CONTAINER_NAME) $(IMAGE_NAME)

docker-stop:
	docker stop $(CONTAINER_NAME)

docker-start:
	docker start $(CONTAINER_NAME)

docker-rm:
	docker rm $(CONTAINER_NAME)
