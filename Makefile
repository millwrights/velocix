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

## Build for all macOS targets
build-all: build-amd64 build-arm64

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
