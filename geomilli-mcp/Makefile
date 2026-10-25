VERSION ?= $(shell cat ../VERSION 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X github.com/geomilli/geomilli-mcp/internal/version.Version=$(VERSION)"

.PHONY: build test run-stdio run-http clean

build:
	mkdir -p bin
	go build $(LDFLAGS) -o bin/geomilli-mcp ./cmd/geomilli-mcp

test:
	go test -race -v ./...

run-stdio: build
	./bin/geomilli-mcp --transport stdio

run-http: build
	./bin/geomilli-mcp --transport http --http-addr 127.0.0.1:8090

clean:
	rm -rf bin/
