SHELL := /bin/bash

BIN_DIR := bin
BIN_NAME := dedup
BIN := $(BIN_DIR)/$(BIN_NAME)

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X 'deduplicator/internal/cli.Version=$(VERSION)' \
	-X 'deduplicator/internal/cli.Commit=$(COMMIT)' \
	-X 'deduplicator/internal/cli.BuildDate=$(DATE)'

.PHONY: all build test test-race test-e2e lint vet tidy fmt clean run help

all: build

build: ## Build the dedup binary into ./bin/dedup
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/dedup

test: ## Run unit tests
	go test ./... -count=1

test-race: ## Run tests with the race detector
	go test ./... -race -count=1

test-e2e: build ## Run end-to-end tests (compiles binary first)
	go test -tags=e2e ./test/e2e/... -count=1

vet: ## go vet
	go vet ./...

lint: ## golangci-lint (requires the tool installed)
	golangci-lint run

tidy: ## go mod tidy
	go mod tidy

fmt: ## gofmt/goimports
	gofmt -s -w .

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)

run: build ## Build and run with args: make run ARGS="report ./some/dir"
	$(BIN) $(ARGS)

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS=":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'
