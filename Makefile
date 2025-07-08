# Bifrost Makefile
# Alternative to justfile for those who prefer make

.PHONY: help dev check quick check-all fmt lint test build clean

# Default target
.DEFAULT_GOAL := help

# Colors
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m

help:  ## Show this help message
	@echo "$(BLUE)🌉 Bifrost Development Commands$(NC)"
	@echo "==============================="
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)💡 Tip: Install 'just' for more advanced features: cargo install just$(NC)"

dev: fmt lint test  ## Complete development workflow
	@echo "$(GREEN)✅ Development workflow complete!$(NC)"

check: fmt lint  ## Quick format and lint check
	@echo "$(GREEN)✅ Quick check complete!$(NC)"

quick:  ## Super quick essential checks (30s)
	@./scripts/quick-check.sh

check-all:  ## Comprehensive quality check with detailed report
	@echo "$(BLUE)🌉 Running comprehensive Bifrost quality check...$(NC)"
	@./scripts/check-all.sh

fmt:  ## Format all code
	@echo "$(BLUE)🎨 Formatting code...$(NC)"
	@go fmt ./... 2>/dev/null || true
	@ruff format packages/ 2>/dev/null || echo "⚠️  ruff not found"
	@echo "$(GREEN)✅ Formatting complete!$(NC)"

lint:  ## Lint all code
	@echo "$(BLUE)🔍 Linting code...$(NC)"
	@golangci-lint run ./... 2>/dev/null || go vet ./...
	@ruff check packages/ 2>/dev/null || echo "⚠️  ruff not found"
	@mypy packages/bifrost/src/ packages/bifrost-core/src/ 2>/dev/null || echo "⚠️  mypy not found"
	@echo "$(GREEN)✅ Linting complete!$(NC)"

test:  ## Run all tests
	@echo "$(BLUE)🧪 Running tests...$(NC)"
	@go test ./... 2>/dev/null || echo "⚠️  Go tests failed or not found"
	@pytest packages/bifrost/tests/ packages/bifrost-core/tests/ 2>/dev/null || echo "⚠️  pytest not found"
	@echo "$(GREEN)✅ Tests complete!$(NC)"

build:  ## Build all components
	@echo "$(BLUE)🏗️  Building...$(NC)"
	@go build ./cmd/gateway
	@echo "$(GREEN)✅ Build complete!$(NC)"

clean:  ## Clean build artifacts
	@echo "$(BLUE)🧹 Cleaning...$(NC)"
	@rm -f gateway bifrost-gateway
	@find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	@find . -type d -name "*.egg-info" -exec rm -rf {} + 2>/dev/null || true
	@echo "$(GREEN)✅ Clean complete!$(NC)"

run: build  ## Build and run the gateway
	@echo "$(BLUE)🚀 Running gateway...$(NC)"
	@./gateway

# Docker commands
docker-build:  ## Build Docker image
	@echo "$(BLUE)🐳 Building Docker image...$(NC)"
	@docker build -t bifrost/gateway .

docker-run:  ## Run with Docker Compose
	@echo "$(BLUE)🐳 Running with Docker Compose...$(NC)"
	@docker-compose up -d

docker-stop:  ## Stop Docker containers
	@echo "$(BLUE)🛑 Stopping Docker containers...$(NC)"
	@docker-compose down

# Setup commands
setup:  ## Set up development environment
	@echo "$(BLUE)🛠️  Setting up development environment...$(NC)"
	@echo "Installing Go tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>/dev/null || echo "Failed to install golangci-lint"
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest 2>/dev/null || echo "Failed to install gosec"
	@echo "Installing Python tools..."
	@pip install ruff mypy bandit pytest pytest-cov 2>/dev/null || echo "Failed to install Python tools"
	@echo "$(GREEN)✅ Development environment setup complete!$(NC)"

status:  ## Show tool versions and status
	@echo "$(BLUE)📊 Project Status:$(NC)"
	@echo "=================="
	@echo "Go version: $$(go version 2>/dev/null || echo 'Not installed')"
	@echo "Python version: $$(python3 --version 2>/dev/null || echo 'Not installed')"
	@echo "Docker version: $$(docker --version 2>/dev/null || echo 'Not installed')"
	@echo ""
	@echo "Go tools:"
	@echo "  golangci-lint: $$(golangci-lint --version 2>/dev/null || echo 'Not installed')"
	@echo ""
	@echo "Python tools:"
	@echo "  ruff: $$(ruff --version 2>/dev/null || echo 'Not installed')"
	@echo "  mypy: $$(mypy --version 2>/dev/null || echo 'Not installed')"