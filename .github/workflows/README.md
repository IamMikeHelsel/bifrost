# GitHub Actions Workflows

This directory contains simplified, practical GitHub Actions workflows for the Bifrost project.

## Workflows

### 🧪 [`test.yml`](test.yml) - Test Runner

- **Triggers**: Push to main/dev, PRs to main
- **Purpose**: Run the full test suite with coverage reporting
- **Key Features**:
  - Runs pytest with coverage
  - Uploads results to Codecov
  - Simple and fast

### 🎨 [`quality.yml`](quality.yml) - Code Quality Checks

- **Triggers**: Push to main/dev, PRs to main
- **Purpose**: Ensure code formatting and linting standards
- **Key Features**:
  - Ruff format checking
  - Ruff linting
  - Optional type checking with mypy (non-blocking)

### 📦 [`build.yml`](build.yml) - Package Builder

- **Triggers**: Push to main, PRs to main, version tags
- **Purpose**: Build Python packages
- **Key Features**:
  - Builds all packages in the monorepo
  - Uploads artifacts for inspection
  - Validates package structure

### 🚀 [`release.yml`](release.yml) - Release Publisher

- **Triggers**: Version tags (v\*)
- **Purpose**: Create releases and publish to PyPI
- **Key Features**:
  - Creates GitHub releases automatically
  - Publishes to PyPI (requires PYPI_API_TOKEN secret)
  - Generates release notes

### 🔧 [`dev-check.yml`](dev-check.yml) - Development Helper

- **Triggers**: Push to dev, manual dispatch
- **Purpose**: Quick feedback for development
- **Key Features**:
  - Non-blocking checks
  - Helpful command suggestions
  - Quick test runs

### 💬 [`pr-helper.yml`](pr-helper.yml) - PR Assistant

- **Triggers**: Pull requests
- **Purpose**: Provide helpful feedback on PRs
- **Key Features**:
  - Comments with quality report
  - Actionable fix suggestions
  - Updates comment on each push

## Why These Workflows?

1. **Simple & Fast**: Each workflow has a single, clear purpose
1. **Developer-Friendly**: Helpful error messages and fix suggestions
1. **Non-Blocking**: Development workflows don't block on minor issues
1. **Practical**: Focus on what actually matters for the project stage

## Required Secrets

- `PYPI_API_TOKEN`: Required for publishing releases to PyPI

## Local Development

The workflows assume you're using the project's `justfile` commands:

```bash
just check  # Run all checks locally
just fmt    # Auto-format code
just lint   # Fix linting issues
just test   # Run full test suite
```
