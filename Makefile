# Makefile for Leet CLI

BINARY_NAME := leet
TUI_BINARY_NAME := leet-tui
GO := go
GOPATH_BIN := $(shell $(GO) env GOPATH)/bin

.PHONY: all build build-cli build-tui install test vet clean run run-tui help

all: build

## build: Build the leet binary in current directory
build:
	@echo "==> Building $(BINARY_NAME)..."
	$(GO) build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/leet
	@echo "✔ Binary built: ./$(BINARY_NAME)"

build-cli: build

build-tui:
	@echo "==> Building $(TUI_BINARY_NAME)..."
	$(GO) build -ldflags="-s -w" -o $(TUI_BINARY_NAME) ./cmd/leet-tui
	@echo "✔ Binary built: ./$(TUI_BINARY_NAME)"

LOCAL_BIN := $(HOME)/.local/bin

## install: Build and install leet into GOPATH/bin and ~/.local/bin
install:
	@echo "==> Installing $(BINARY_NAME) to $(GOPATH_BIN)..."
	@mkdir -p $(GOPATH_BIN)
	$(GO) build -ldflags="-s -w" -o $(GOPATH_BIN)/$(BINARY_NAME) ./cmd/leet
	@if [ -d "$(LOCAL_BIN)" ]; then \
		cp -f $(GOPATH_BIN)/$(BINARY_NAME) $(LOCAL_BIN)/$(BINARY_NAME); \
		echo "✔ Installed to $(LOCAL_BIN)/$(BINARY_NAME)"; \
	fi
	@echo "✔ Installed to $(GOPATH_BIN)/$(BINARY_NAME)"
	@echo "Run 'leet --version' or 'leet init' to get started!"

## test: Run all automated unit tests
test:
	@echo "==> Running tests..."
	$(GO) test -v ./...

## vet: Run go vet on codebase
vet:
	@echo "==> Running go vet..."
	$(GO) vet ./...

## fmt: Format codebase with gofmt
fmt:
	@echo "==> Formatting code..."
	gofmt -w .

## run: Run the TUI directly with go run
run:
	$(GO) run ./cmd/leet $(ARGS)

run-tui:
	$(GO) run ./cmd/leet-tui

## clean: Remove build artifacts and temporary binaries
clean:
	@echo "==> Cleaning build artifacts..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME).exe $(TUI_BINARY_NAME) $(TUI_BINARY_NAME).exe
	@$(GO) run ./cmd/leet clean 2>/dev/null || true
	@echo "✔ Workspace clean."

## help: Display this help message
help:
	@echo "Leet CLI - Available make targets:"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
