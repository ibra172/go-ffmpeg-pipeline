export PROJECT_ROOT=$(shell pwd)

VENV = ${PROJECT_ROOT}/tests/venv
PYTHON = $(VENV)/bin/python3
PIP = $(VENV)/bin/pip
PYTEST = $(VENV)/bin/pytest

.PHONY: test
test:
	python3 -m venv $(VENV)
# 	$(PIP) install pytest requests
	$(PYTEST) ${PROJECT_ROOT}/tests/tests1.py -k "task" -v


IMAGE_NAME = media-pipeline-app

docker-build:
	docker build -f ${PROJECT_ROOT}/cmd/app/Dockerfile -t $(IMAGE_NAME) .

docker-run:
	docker run -p 8000:8000 --name media-pipeline $(IMAGE_NAME)
