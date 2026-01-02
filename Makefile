.PHONY: build install test lint clean

BINARY := riff
BUILD_DIR := ./cmd/riff
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X github.com/tessro/riff/internal/cli.Version=$(VERSION) \
           -X github.com/tessro/riff/internal/cli.Commit=$(COMMIT) \
           -X github.com/tessro/riff/internal/cli.BuildDate=$(BUILD_DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(BUILD_DIR)

install:
	go install $(BUILD_DIR)

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY)
	go clean
