.PHONY: all build run test clean docker-build

APP_NAME := geomilli-api
BIN_DIR := bin

all: test build

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) ./cmd/api

run:
	go run ./cmd/api

test:
	go test -v -race ./...

docker-build:
	docker build -t $(APP_NAME):latest .

clean:
	rm -rf $(BIN_DIR)
