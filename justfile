# Bifrost Project Justfile
# Modern task runner for cross-platform development

# Install uv if not available
install-uv:
    @which uv > /dev/null || curl -LsSf https://astral.sh/uv/install.sh | sh

# Install pre-commit hooks
install-hooks:
    @echo "🪝 Installing pre-commit hooks..."
    uv run pre-commit install
    @echo "✅ Pre-commit hooks installed!"

# Set up development environment
dev-setup: install-uv install-hooks
    @echo "🔧 Setting up Bifrost development environment..."
    uv python install 3.12
    uv sync --all-extras --dev
    @echo "✅ Development environment ready!"

# Install all packages in development mode
dev-install: dev-setup
    @echo "📦 Installing all packages in development mode..."
    uv pip install -e packages/bifrost-core
    uv pip install -e packages/bifrost
    # TODO: Add these packages as they are created:
    # uv pip install -e packages/bifrost-opcua
    # uv pip install -e packages/bifrost-analytics
    # uv pip install -e packages/bifrost-cloud
    # uv pip install -e packages/bifrost-protocols
    @echo "✅ All packages installed in development mode"

# Format all code (Google style: 80 characters)
fmt:
    @echo "🎨 Formatting Python code (Google style)..."
    uv run ruff format --line-length 80 .
    @echo "📝 Formatting markdown..."
    mdformat .
    @echo "✅ All code formatted!"

# Format files and run pre-commit on all files
fmt-all:
    @echo "🎨 Running pre-commit on all files..."
    uv run pre-commit run --all-files
    @echo "✅ All files formatted!"

# Watch for changes and auto-format markdown
watch-md:
    @echo "👀 Watching markdown files for changes..."
    @echo "Press Ctrl+C to stop"
    @while true; do \
        find . -name "*.md" -newer .last-format 2>/dev/null | while read file; do \
            echo "📝 Formatting $$file"; \
            mdformat "$$file"; \
        done; \
        touch .last-format; \
        sleep 2; \
    done

# Lint all code
lint:
    @echo "🔍 Linting Python code..."
    uv run ruff check . --fix
    @echo "✅ All code linted!"

# Type check Python code
typecheck:
    @echo "🔬 Type checking Python code..."
    uv run mypy packages/*/src
    @echo "✅ Type checking complete!"

# Run all tests
test:
    @echo "🧪 Running Python tests..."
    uv run pytest packages/*/tests -v
    @echo "✅ All tests passed!"

# Run tests with coverage
test-cov:
    @echo "🧪 Running tests with coverage..."
    uv run pytest packages/*/tests --cov=packages --cov-report=html --cov-report=term
    @echo "📊 Coverage report generated in htmlcov/"

# Build all packages
build:
    @echo "🔨 Building all packages..."
    uv run python tools/build-all.py
    @echo "✅ All packages built!"


# Clean all build artifacts
clean:
    @echo "🧹 Cleaning build artifacts..."
    find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
    find . -type d -name "*.egg-info" -exec rm -rf {} + 2>/dev/null || true
    find . -type d -name "build" -exec rm -rf {} + 2>/dev/null || true
    find . -type d -name "dist" -exec rm -rf {} + 2>/dev/null || true
    find . -name "*.pyc" -delete 2>/dev/null || true
    find . -name "*.pyo" -delete 2>/dev/null || true
    @echo "✅ Build artifacts cleaned!"

# Run security audit
audit:
    @echo "🔒 Running security audit..."
    uv run pip-audit
    @echo "✅ Security audit complete!"

# Update all dependencies
update:
    @echo "📦 Updating Python dependencies..."
    uv sync --upgrade
    @echo "✅ Dependencies updated!"

# Benchmark performance
bench:
    @echo "⚡ Running benchmarks..."
    uv run pytest packages/*/tests/benchmarks -v --benchmark-only
    @echo "📊 Benchmark results saved"

# Run virtual device tests
test-virtual-devices:
    @echo "🖥️ Running virtual device tests..."
    cd virtual-devices && python -m pytest tests/ -v
    cd go-gateway && go test -v ./internal/protocols/... -tags=virtual
    @echo "✅ Virtual device tests completed"

# Run all benchmarks for release cards
benchmark-all:
    @echo "⚡ Running comprehensive benchmarks..."
    @echo "🐍 Python benchmarks..."
    uv run pytest packages/*/tests/benchmarks -v --benchmark-only --benchmark-json=test-results/python-benchmarks.json
    @echo "🚀 Go performance tests..."
    cd go-gateway && go run cmd/performance_test/main.go --test-type comprehensive --output test-results/go-benchmarks.json
    @echo "📊 All benchmarks completed"

# Generate documentation
docs:
    @echo "📚 Generating documentation..."
    uv run sphinx-build docs docs/_build/html
    @echo "📖 Documentation generated in docs/_build/html"

# Generate release card
release-card version="v0.1.0":
    @echo "🚀 Generating release card for {{version}}..."
    python tools/generate-release-card.py --version {{version}} --test-results test-results --verbose

# Deploy documentation and release cards
deploy-docs:
    @echo "🌐 Deploying documentation..."
    python tools/deploy-docs.py --verbose

# Create mock test results for testing
mock-test-results:
    @echo "🧪 Creating mock test results..."
    ./scripts/create-mock-test-results.sh

# Test release card integration
test-release-cards:
    @echo "🧪 Testing release card CI/CD integration..."
    ./scripts/test-release-card-integration.sh

# Serve documentation locally
docs-serve: docs
    @echo "🌐 Serving documentation at http://localhost:8000"
    uv run python -m http.server 8000 -d docs/_build/html

# Run full CI pipeline locally
ci: fmt lint typecheck test audit
    @echo "🚀 CI pipeline completed successfully!"

# Prepare for release
release-prep: ci build
    @echo "🚀 Running release preparation..."
    uv run python tools/sync-versions.py
    uv run python tools/release.py --dry-run
    @echo "✅ Ready for release!"

# Publish to PyPI (requires manual confirmation)
release: release-prep
    @echo "📦 Publishing to PyPI..."
    uv run python tools/release.py --publish
    @echo "🎉 Release published!"

# Development shortcuts
dev: dev-install fmt lint test
alias d := dev

# Quick check (fast feedback)
check: fmt lint typecheck
alias c := check

# Super quick check (essential checks only)
quick:
    @./scripts/quick-check.sh
alias q := quick

# Comprehensive quality check with detailed report
check-all:
    @echo "🌉 Running comprehensive Bifrost quality check..."
    @./scripts/check-all.sh

# Alias for the comprehensive check
alias qa := check-all

# Google Style Guide specific commands
google-check:
    @echo "📏 Checking Google Python Style Guide compliance..."
    uv run ruff check --select D,PL,C90,N,ERA,PIE,SIM,RET,ARG .
    @echo "✅ Google style check complete!"

google-fix:
    @echo "🔧 Applying auto-fixable Google style violations..."
    uv run ruff check --fix --select I,UP,SIM,PIE,ERA .
    @echo "✅ Auto-fixes applied!"

google-docstring-check:
    @echo "📖 Checking docstring compliance..."
    uv run ruff check --select D .
    @echo "✅ Docstring check complete!"

google-migrate: google-fix fmt lint
    @echo "🎯 Google style migration step completed!"

# Show help
help:
    @echo "Bifrost Development Commands:"
    @echo ""
    @echo "Setup:"
    @echo "  dev-setup     Set up development environment"
    @echo "  dev-install   Install packages in development mode"
    @echo "  install-hooks Install pre-commit hooks"
    @echo ""
    @echo "Development:"
    @echo "  fmt           Format all code (Python, Markdown)"
    @echo "  fmt-all       Run pre-commit on all files"
    @echo "  watch-md      Watch and auto-format markdown files"
    @echo "  lint          Lint all code with auto-fix"
    @echo "  typecheck     Run type checking"
    @echo "  test          Run all tests"
    @echo "  check         Quick format + lint + typecheck"
    @echo "  quick         Super quick essential checks (30s)"
    @echo "  check-all     Comprehensive quality check with detailed report"
    @echo "  dev           Full development cycle"
    @echo ""
    @echo "Build:"
    @echo "  build         Build all packages"
    @echo "  clean         Clean build artifacts"
    @echo ""
    @echo "Quality:"
    @echo "  audit         Run security audit"
    @echo "  bench         Run performance benchmarks"
    @echo "  ci            Full CI pipeline"
    @echo ""
    @echo "Documentation:"
    @echo "  docs          Generate documentation"
    @echo "  docs-serve    Serve documentation locally"
    @echo ""
    @echo "Release:"
    @echo "  release-prep  Prepare for release"
    @echo "  release       Publish to PyPI"
    @echo ""
    @echo "Use 'just <command>' to run any command"

# Default command shows help
default: help