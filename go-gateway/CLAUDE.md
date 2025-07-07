# CLAUDE.md - Bifrost Go Gateway

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The Bifrost Go Gateway is a high-performance industrial protocol gateway written in Go. It bridges OT (Operational Technology) equipment with modern IT infrastructure by providing unified APIs for industrial protocols (Modbus, Ethernet/IP, etc.) with exceptional performance and reliability.

### Mission Statement

Bridge the gap between industrial equipment and modern IT infrastructure. Provide a reliable, high-performance gateway that enables seamless integration of industrial protocols with cloud and edge computing platforms.

### Target Users

- **Industrial Engineers** integrating legacy equipment with modern systems
- **System Integrators** building comprehensive industrial automation solutions
- **DevOps Engineers** deploying industrial gateways in cloud and edge environments
- **Software Developers** building applications that interact with industrial equipment

## Architecture

This is a Go-based microservice with the following key components:

- **Gateway Core**: Main server handling HTTP/gRPC APIs and protocol routing
- **Protocol Handlers**: Modbus, Ethernet/IP, and other industrial protocol implementations
- **Performance Optimizations**: Connection pooling, batch processing, and memory optimization
- **Monitoring**: Comprehensive metrics, logging, and health checks
- **Security**: TLS encryption, authentication, and authorization

## Development Workflow

### Task Runner

This project uses `just` as a modern task runner instead of make. Common commands:

```bash
# Development cycle
just dev          # format + lint + test
just check        # quick format + lint + typecheck

# Individual tasks
just fmt          # format all code (Go + Starlark)
just lint         # lint all code (Go + Starlark)
just build        # build the binary
just test         # run tests
just run          # build and run

# Setup
just dev-setup    # install all development tools
```

### Code Style and Formatting

This project maintains strict code quality standards across multiple languages:

#### Go Style Guide

- **Formatter**: `gofmt` for consistent Go formatting
- **Linter**: `golangci-lint` for comprehensive Go linting
- **Standards**: Follow standard Go conventions and best practices
- **Import Organization**: Automatic import sorting and cleanup

#### Starlark Style Guide

- **Formatter**: `buildifier` for Bazel/Starlark file formatting
- **Linter**: `buildifier` with warning-level linting enabled
- **File Types**: `BUILD`, `BUILD.bazel`, `*.bzl`, `WORKSPACE`, `WORKSPACE.bazel`
- **Standards**: Follow Bazel best practices and Google's Starlark style guide

**Starlark Formatting Rules**:
- Use 4-space indentation
- Sort rule attributes alphabetically where possible
- Use consistent naming conventions for targets
- Include descriptive comments for complex rules
- Prefer explicit visibility declarations

#### Justfile Commands for Style

```bash
# Format all code
just fmt              # Format Go + Starlark
just fmt-go           # Format Go only
just fmt-starlark     # Format Starlark only

# Lint all code
just lint             # Lint Go + Starlark
just lint-go          # Lint Go only
just lint-starlark    # Lint Starlark only

# Check formatting
just check            # Check all formatting
just check-starlark   # Check Starlark formatting only
```

### Pre-commit Hooks

This project uses pre-commit hooks to ensure code quality:

- **Go formatting** with `gofmt`
- **Go linting** with `golangci-lint`
- **Starlark formatting** with `buildifier`
- **Security scanning** with `detect-secrets`
- **General code quality** checks (trailing whitespace, large files, etc.)

Setup pre-commit hooks:
```bash
just pre-commit-setup
```

### Development Tools

**Required Tools**:
- `go` (1.22+)
- `just` (task runner)
- `buildifier` (Starlark formatting)
- `golangci-lint` (Go linting)

**Optional Tools**:
- `govulncheck` (security scanning)
- `gosec` (security analysis)
- `pre-commit` (pre-commit hooks)

Install all tools:
```bash
just dev-setup
```

## Code Quality Standards

### Go Standards

- Use `context.Context` for cancellation and timeouts
- Implement proper error handling with wrapped errors
- Use structured logging with appropriate log levels
- Follow Go naming conventions
- Write comprehensive tests with race detection
- Use dependency injection for testability

### Starlark Standards

- Use descriptive target names
- Organize BUILD files logically
- Include proper dependencies
- Use appropriate visibility settings
- Document complex rules and macros
- Follow Bazel performance best practices

### Performance Requirements

- **Latency**: < 1ms for Modbus register reads
- **Throughput**: 10,000+ operations/second
- **Memory**: < 100MB base memory usage
- **CPU**: Efficient CPU usage with proper goroutine management

## CI/CD Pipeline

The project uses GitHub Actions for CI/CD:

- **Format Check**: Validates Go and Starlark formatting
- **Lint**: Runs `golangci-lint` and `buildifier`
- **Test**: Runs comprehensive test suite with race detection
- **Security**: Vulnerability scanning with Trivy and Gosec
- **Build**: Multi-platform binary builds
- **Docker**: Container image builds for multiple architectures
- **Deploy**: Automated deployment to staging and production

## VS Code Integration

The project includes VS Code settings for:

- **Go language server** with proper configuration
- **Starlark/Bazel support** with buildifier integration
- **Automatic formatting** on save for all supported languages
- **Recommended extensions** for optimal development experience

## Key Files

- `justfile`: Modern task runner with all development commands
- `go.mod/go.sum`: Go module dependencies
- `BUILD.bazel`: Bazel build files for the project
- `WORKSPACE.bazel`: Bazel workspace configuration
- `.pre-commit-config.yaml`: Pre-commit hook configuration
- `.vscode/settings.json`: VS Code workspace settings
- `.github/workflows/ci.yml`: CI/CD pipeline configuration

## Development Best Practices

1. **Code Quality**: Run `just dev` before committing changes
2. **Testing**: Write tests for all new functionality
3. **Documentation**: Update documentation for significant changes
4. **Security**: Never commit secrets or sensitive information
5. **Performance**: Profile and benchmark performance-critical code
6. **Monitoring**: Include appropriate metrics and logging
7. **Error Handling**: Implement comprehensive error handling
8. **Concurrency**: Use Go's concurrency primitives safely

## Bazel Integration

This project uses Bazel for build management:

- **BUILD files**: Define build targets and dependencies
- **Workspace**: Configure external dependencies and toolchains
- **Gazelle**: Automatic BUILD file generation for Go
- **Rules**: Custom build rules for specific requirements

## Contributing Guidelines

When working on this project:

1. Use the provided development tools and workflow
2. Follow the established code style guidelines
3. Ensure all tests pass and code is properly formatted
4. Update documentation for significant changes
5. Use semantic commit messages
6. Submit pull requests with clear descriptions

## Security Considerations

- **TLS**: All network communications use TLS encryption
- **Authentication**: Implement proper authentication mechanisms
- **Authorization**: Use role-based access control
- **Secrets Management**: Never hardcode secrets in source code
- **Vulnerability Scanning**: Regular security scans in CI/CD pipeline

This project prioritizes security, performance, and maintainability while providing a robust industrial protocol gateway solution.