# Bifrost Industrial Gateway Makefile
.PHONY: build test run clean fmt lint deps

# Build settings
BINARY_NAME=bifrost-gateway
BINARY_DIR=bin
MAIN_PATH=cmd/gateway/main.go

# Go settings
GO_FLAGS=-ldflags="-s -w" -trimpath

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	go build $(GO_FLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build $(GO_FLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build $(GO_FLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_DIR)/$(BINARY_NAME)

# Run with custom config
run-config: build
	@echo "Running $(BINARY_NAME) with custom config..."
	./$(BINARY_DIR)/$(BINARY_NAME) -config gateway.yaml

# Run in development mode
dev:
	@echo "Running in development mode..."
	go run $(MAIN_PATH) -log-level debug

# Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -v -bench=. -benchmem ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BINARY_DIR)
	go clean

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	go install $(MAIN_PATH)

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t bifrost-gateway .

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 -p 9090:9090 bifrost-gateway

# Performance test
perf-test:
	@echo "Running performance tests..."
	go test -v -bench=BenchmarkModbus -benchtime=10s ./internal/protocols/

# Help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  build-all   - Build for all platforms"
	@echo "  run         - Build and run the application"
	@echo "  run-config  - Run with custom configuration"
	@echo "  dev         - Run in development mode"
	@echo "  test        - Run tests"
	@echo "  bench       - Run benchmarks"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  deps        - Install dependencies"
	@echo "  clean       - Clean build artifacts"
	@echo "  install     - Install the binary"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run  - Run Docker container"
	@echo "  perf-test   - Run performance tests"
	@echo "  help        - Show this help message"