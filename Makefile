APP_NAME := velocix
VERSION ?= dev
BUILD_DIR := bin
LDFLAGS := -ldflags "-X github.com/skalluru/velocix/internal/cli.Version=$(VERSION)"

.PHONY: build build-amd64 run-serve run-tui test clean deps

## Build for current macOS architecture
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/velocix

## Build for macOS Intel
build-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/velocix

## Build for macOS Apple Silicon
build-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/velocix

## Build for Linux AMD64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/velocix

## Build for Linux ARM64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./cmd/velocix

## Build for all targets
build-all: build-amd64 build-arm64 build-linux-amd64 build-linux-arm64

## Run web server
run-serve:
	go run ./cmd/velocix serve

## Run TUI
run-tui:
	go run ./cmd/velocix tui

## Run tests
test:
	go test ./...

## Install dependencies
deps:
	go mod tidy

## Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
