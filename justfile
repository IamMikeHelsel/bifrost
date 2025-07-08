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

# Run tests with Bazel
test-bazel:
    @echo "🧪 Running tests with Bazel..."
    bazel test //packages/...
    @echo "✅ All tests passed with Bazel!"

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

# Build all packages with Bazel
build-bazel:
    @echo "🔨 Building all packages with Bazel..."
    bazel build //packages/...
    @echo "✅ All packages built with Bazel!"

# Build wheels with Bazel
build-wheels:
    @echo "📦 Building distribution wheels with Bazel..."
    bazel build //packages/...:wheel
    @echo "✅ Wheels built with Bazel!"


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

# Clean Bazel build cache
clean-bazel:
    @echo "🧹 Cleaning Bazel build cache..."
    bazel clean
    @echo "✅ Bazel cache cleaned!"

# Clean all Bazel artifacts (including analysis cache)
clean-bazel-all:
    @echo "🧹 Cleaning all Bazel artifacts..."
    bazel clean --expunge
    @echo "✅ All Bazel artifacts cleaned!"

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

# Generate documentation
docs:
    @echo "📚 Generating documentation..."
    uv run sphinx-build docs docs/_build/html
    @echo "📖 Documentation generated in docs/_build/html"

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
    @echo "  test          Run all tests (legacy)"
    @echo "  test-bazel    Run all tests with Bazel"
    @echo "  check         Quick format + lint + typecheck"
    @echo "  quick         Super quick essential checks (30s)"
    @echo "  check-all     Comprehensive quality check with detailed report"
    @echo "  dev           Full development cycle"
    @echo ""
    @echo "Build:"
    @echo "  build         Build all packages (legacy)"
    @echo "  build-bazel   Build all packages with Bazel"
    @echo "  build-wheels  Build distribution wheels with Bazel"
    @echo "  clean         Clean build artifacts"
    @echo "  clean-bazel   Clean Bazel build cache"
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
    @echo "Bazel Commands:"
    @echo "  bazel-dev     Bazel development cycle (build + test)"
    @echo "  deps TARGET   Show dependencies for a target"
    @echo "  rdeps TARGET  Show reverse dependencies for a target"
    @echo "  cache-stats   Show Bazel cache statistics"
    @echo "  run TARGET    Run a specific Bazel target"
    @echo ""
    @echo "Use 'just <command>' to run any command"

# Default command shows help
default: help

# === Bazel-specific commands ===

# Query dependencies for a target
deps TARGET:
    @echo "🔍 Analyzing dependencies for {{TARGET}}..."
    bazel query "deps({{TARGET}})"

# Query reverse dependencies for a target  
rdeps TARGET:
    @echo "🔍 Analyzing reverse dependencies for {{TARGET}}..."
    bazel query "rdeps(//..., {{TARGET}})"

# Show Bazel cache statistics
cache-stats:
    @echo "📊 Bazel cache statistics..."
    bazel info

# Run a specific Bazel target
run TARGET:
    @echo "🚀 Running Bazel target {{TARGET}}..."
    bazel run {{TARGET}}

# Build specific package
build-package PACKAGE:
    @echo "🔨 Building package {{PACKAGE}} with Bazel..."
    bazel build //packages/{{PACKAGE}}:{{PACKAGE}}

# Test specific package
test-package PACKAGE:
    @echo "🧪 Testing package {{PACKAGE}} with Bazel..."
    bazel test //packages/{{PACKAGE}}:tests

# Bazel development workflow (build + test)
bazel-dev:
    @echo "🔄 Running Bazel development cycle..."
    just build-bazel
    just test-bazel
    @echo "✅ Bazel development cycle complete!"