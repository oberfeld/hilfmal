.PHONY: build test vet fmt-check verify integration lint clean install help

BINARY_NAME=hilfmal
BINARY_PATH=cmd/$(BINARY_NAME)
OUTPUT_DIR=bin

help:
	@echo "Hilfmal Go Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build       - Build the hilfmal binary"
	@echo "  test        - Run all tests"
	@echo "  verify      - Check formatting, vet, tests, build, and Docker integration"
	@echo "  integration - Run isolated Nextcloud and Mailpit integration tests"
	@echo "  lint        - Run linter (requires golangci-lint)"
	@echo "  fmt         - Format code"
	@echo "  clean       - Remove build artifacts"
	@echo "  install     - Install binary to GOPATH/bin"
	@echo "  run         - Run with default config (for development)"
	@echo "  help        - Show this help message"

build:
	@mkdir -p $(OUTPUT_DIR)
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(OUTPUT_DIR)/$(BINARY_NAME) ./$(BINARY_PATH)
	@echo "Binary created at $(OUTPUT_DIR)/$(BINARY_NAME)"

test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

vet:
	@go vet ./...

fmt-check:
	@unformatted="$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))"; \
	if [ -n "$$unformatted" ]; then \
		echo "Go files need formatting:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

verify: fmt-check vet test build integration

integration:
	@sh tests/integration/run.sh

lint:
	@echo "Running linter..."
	@golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	@go fmt ./...

clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(OUTPUT_DIR)/$(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@go clean

install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install ./$(BINARY_PATH)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

run: build
	@echo "Running $(BINARY_NAME) with config.example.yaml..."
	@./$(OUTPUT_DIR)/$(BINARY_NAME) config.example.yaml
