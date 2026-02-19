GO ?= go
BINARY ?= tiny-headend
BIN_DIR ?= bin
OUTPUT := $(BIN_DIR)/$(BINARY)
PKGS := ./...
ARGS ?=
GOCACHE ?= $(CURDIR)/.cache/go-build
COVERPROFILE ?= $(CURDIR)/.cache/coverage.out

.DEFAULT_GOAL := help

.PHONY: help build test fmt vet tidy clean

help: ## Show available make targets
	@echo "Usage: make <target>"
	@echo
	@echo "Targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

build: ## Build binary to bin/tiny-headend
	@mkdir -p $(BIN_DIR)
	@mkdir -p $(GOCACHE)
	CGO_CFLAGS="-Wno-discarded-qualifiers" GOCACHE=$(GOCACHE) $(GO) build -o $(OUTPUT) .

test: ## Run unit tests
	@mkdir -p $(GOCACHE)
	GOCACHE=$(GOCACHE) $(GO) test -coverprofile=$(COVERPROFILE) $(PKGS)
	@GOCACHE=$(GOCACHE) $(GO) tool cover -func=$(COVERPROFILE) | tail -n 1

fmt: ## Format Go source files
	@mkdir -p $(GOCACHE)
	GOCACHE=$(GOCACHE) $(GO) fmt $(PKGS)

vet: ## Run go vet
	@mkdir -p $(GOCACHE)
	GOCACHE=$(GOCACHE) $(GO) vet $(PKGS)

tidy: ## Tidy and verify module dependencies
	@mkdir -p $(GOCACHE)
	GOCACHE=$(GOCACHE) $(GO) mod tidy
	GOCACHE=$(GOCACHE) $(GO) mod verify

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) .cache

run: build ## Build and run the binary
	./$(OUTPUT) $(ARGS)
