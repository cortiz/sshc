BINARY_NAME=sshc
VERSION?=0.1.0
BUILD_DIR=bin

.PHONY: all build clean test lint help

all: build

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

clean:
	rm -rf $(BUILD_DIR)

test:
	go test -v ./...

lint:
	golangci-lint run

help:
	@echo "Makefile for $(BINARY_NAME)"
	@echo ""
	@echo "Usage:"
	@echo "  make build    Build the binary"
	@echo "  make clean    Remove build artifacts"
	@echo "  make test     Run tests"
	@echo "  make lint     Run linter"
	@echo "  make help     Show this help message"
