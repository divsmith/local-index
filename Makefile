.PHONY: build test test-fast test-clean lint fmt clean install

# Build the CLI tool
build:
	go build -o bin/code-search ./src

# Run all tests
test:
	go test -v ./src/... ./tests/unit/ ./tests/contract/

# Run fast tests for development (skip integration tests)
test-fast:
	go test -v ./src/... ./tests/unit/ ./tests/contract/ -short

# Clean up test artifacts
test-clean:
	@echo "Cleaning up test artifacts..."
	@find ./tests -name ".code-search-index*" -type f -delete 2>/dev/null || true
	@rm -rf ./tests/contract/resources/*/tmp* 2>/dev/null || true
	@rm -rf ./tests/integration/resources/*/tmp* 2>/dev/null || true
	@echo "Test artifacts cleaned up"

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./src/... ./tests/unit/ ./tests/contract/
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Install the CLI tool
install: build
	cp bin/code-search $(shell go env GOPATH)/bin/

# Development setup
setup: deps
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2; \
	fi
	@if ! command -v goimports &> /dev/null; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@echo "Development setup complete!"