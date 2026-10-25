.PHONY: all build run test clean docker-build

APP_NAME := geomilli-api
BIN_DIR := bin
VERSION ?= $(shell if [ -f ../VERSION ]; then cat ../VERSION; else echo "dev"; fi)

all: test build

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w -X github.com/geomilli/geomilli-api/internal/version.Version=$(VERSION)" -o $(BIN_DIR)/$(APP_NAME) ./cmd/api

run:
	go run ./cmd/api

test:
	go test -v -race ./...

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(APP_NAME):latest .

clean:
	rm -rf $(BIN_DIR)
