# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bifrost is a high-performance industrial gateway built in Go that bridges OT (Operational Technology) equipment with modern IT infrastructure. The project delivers production-ready industrial communication with a TypeScript-Go frontend and Go-based gateway backend for unified industrial protocol support.

### Mission Statement

Break down the walls between operational technology and information technology. Make it as easy to work with a PLC as it is to work with a REST API. Help automation professionals leverage modern tools without abandoning what works.

### Target Users

- **Control Systems Engineers** tired of duct-taping solutions together
- **Automation Engineers** who want modern development tools
- **SCADA/HMI Developers** looking for reliable protocol gateways
- **IT Developers** who need to understand industrial equipment
- **System Integrators** seeking reliable, performant tools
- **Process Engineers** trying to get data into analytics platforms

## Architecture

### System Architecture

**High-Performance Gateway Approach** - TypeScript-Go frontend with Go backend:

```
bifrost/
├── go-gateway/              # Go-based industrial gateway
│   ├── cmd/
│   │   ├── gateway/         # Main gateway server
│   │   └── performance_test/
│   ├── internal/
│   │   ├── protocols/       # Protocol implementations
│   │   │   ├── modbus.go
│   │   │   ├── ethernetip.go
│   │   │   └── protocol.go
│   │   ├── gateway/         # Core gateway logic
│   │   │   └── server.go
│   │   └── performance/     # Performance optimizations
│   ├── configs/             # Configuration files
│   ├── examples/            # Usage examples
│   ├── bin/                 # Compiled binaries
│   ├── Makefile
│   └── go.mod
├── vscode-extension/        # TypeScript-Go VS Code extension
│   ├── src/
│   │   ├── extension.ts     # Main extension logic
│   │   ├── services/        # Device management services
│   │   ├── providers/       # VS Code providers
│   │   └── utils/           # Utility functions
│   ├── package.json
│   └── tsconfig.json
├── virtual-devices/         # Virtual device testing framework
│   ├── simulators/          # Device simulators
│   ├── mocks/              # Lightweight mocks
│   └── scenarios/          # Industrial scenarios
├── docs/                    # Documentation
├── examples/                # Usage examples
├── .github/                 # GitHub Actions workflows
├── justfile                 # Task runner
└── README.md
```

### Installation Patterns

**For Different Use Cases**:

```bash
# Gateway Only (Production)
wget https://github.com/bifrost/gateway/releases/latest/download/bifrost-gateway-linux-amd64
chmod +x bifrost-gateway-linux-amd64
./bifrost-gateway-linux-amd64           # ~15MB single binary

# Development Setup  
git clone https://github.com/bifrost/bifrost
cd bifrost/go-gateway
make build                              # Build from source

# VS Code Extension
# Install from VS Code Marketplace: "Bifrost Industrial Gateway"
# Or: code --install-extension bifrost.industrial-gateway

# Docker Deployment
docker pull bifrost/gateway:latest
docker run -p 8080:8080 bifrost/gateway:latest

# Complete Development Environment
git clone https://github.com/bifrost/bifrost
cd bifrost
just dev-setup                          # Sets up Go + TypeScript environment
```

### Key Design Principles

- **High Performance**: Go-powered gateway with 18,879 ops/sec throughput and 53µs latency
- **Single Binary Deployment**: No runtime dependencies, easy production deployment
- **Protocol Agnostic**: Unified REST API for all industrial protocols
- **Real-time Communication**: WebSocket streaming for live data monitoring
- **Type Safety**: Go's compile-time guarantees and TypeScript-Go frontend
- **Production Ready**: Prometheus metrics, structured logging, graceful shutdown

### API Architecture

The Go gateway provides a unified REST API for all industrial protocols:

```bash
# Device Management
GET /api/devices                    # List connected devices
POST /api/devices/discover          # Discover devices on network
GET /api/devices/{id}/info          # Get device information

# Data Operations  
GET /api/tags/read                  # Read tag values
POST /api/tags/write                # Write tag values
WS /ws                              # Real-time data streaming

# Monitoring
GET /metrics                        # Prometheus metrics
GET /health                         # Health check endpoint
```

## Development Status

This is currently a **production-ready implementation** with proven performance. The repository contains:

- High-performance Go gateway with comprehensive testing
- Production deployment capabilities
- TypeScript-Go VS Code extension for development
- Virtual device testing framework for validation
- Comprehensive documentation and examples

## Performance Achieved

Based on comprehensive testing with production hardware:

- **Modbus TCP**: 18,879 ops/second with 53µs average latency (ACHIEVED)
- **Memory Usage**: < 50MB base footprint (EXCEEDED TARGET)
- **Concurrent Connections**: 1000+ simultaneous device connections
- **Network Throughput**: Optimized for industrial edge deployment
- **Response Time**: Sub-100µs for critical operations

## Core Components

### 1. Go Gateway (`go-gateway/`)

- High-performance Go implementation with proven 18,879 ops/sec throughput
- Modbus TCP/RTU support with 53µs average latency
- REST API with WebSocket streaming for real-time data
- Prometheus metrics and structured logging
- Connection pooling and concurrent device management

### 2. Protocol Handlers (`internal/protocols/`)

- **Modbus TCP/RTU**: Production-ready with connection pooling
- **Ethernet/IP (CIP)**: Native Go implementation in development
- **OPC UA**: Planned integration with security profiles
- **S7 (Siemens)**: Future protocol support
- Unified ProtocolHandler interface for all protocols

### 3. VS Code Extension (`vscode-extension/`)

- TypeScript-Go powered for 10x faster compilation
- Real-time device monitoring and management
- Industrial protocol debugging tools
- PLC programming assistance and code completion
- Integration with Go gateway via REST API

### 4. Virtual Device Framework (`virtual-devices/`)

- Comprehensive testing infrastructure for industrial scenarios
- Device simulators for Modbus, OPC UA, Ethernet/IP
- Network condition simulation (latency, packet loss)
- Performance benchmarking and validation
- Industrial scenario testing (factory floor, process control)

### 5. Gateway API & Monitoring

**REST API Endpoints**:

```bash
# Device operations
GET /api/devices                    # List connected devices
POST /api/devices/discover          # Network device discovery
GET /api/devices/{id}/info          # Device information

# Data operations
GET /api/tags/read                  # Read tag values
POST /api/tags/write                # Write tag values

# Real-time monitoring
WS /ws                              # WebSocket streaming
GET /metrics                        # Prometheus metrics
GET /health                         # Health checks
```

## Technology Stack

### Core Technologies

- **Go**: 1.22+ for high-performance gateway backend
- **TypeScript-Go**: 10x faster compilation for VS Code extension
- **Task Runner**: just (cross-platform) + Makefile for Go builds
- **Build System**: Go modules with cross-platform binary generation
- **Testing**: Go testing framework + comprehensive benchmarks
- **Documentation**: Comprehensive README and API documentation

### Gateway Performance

- **Runtime**: Native Go compilation for maximum performance
- **Networking**: Go's efficient networking stack with connection pooling  
- **Serialization**: Native Go JSON with efficient memory management
- **Logging**: Structured logging with zap framework
- **Metrics**: Prometheus integration for production monitoring

### Frontend Stack

- **TypeScript-Go**: Microsoft's experimental Go-based compiler
- **VS Code APIs**: Native VS Code extension development
- **WebSocket**: Real-time communication with Go gateway
- **REST Client**: Efficient HTTP client for API communication

### Target Platforms

- **Operating Systems**: Linux (primary), Windows, macOS
- **Architectures**: x86_64, ARM64 (including Raspberry Pi)
- **Deployment**: Edge devices, industrial PCs, cloud environments, containers

## Technology Strategy

### Go Standard Library First

**Selection Criteria**:

- Leverage Go's robust standard library for networking and concurrency
- Minimal external dependencies for reliable production deployment
- Open source, permissive licensing
- Performance and security proven in production

**Core Go Libraries**:

- **net**: Standard networking with connection pooling
- **context**: Request lifecycle and timeout management
- **sync**: Goroutine synchronization and thread safety
- **encoding/json**: Native JSON serialization
- **log/slog**: Structured logging (Go 1.21+)

**External Dependencies**:

- **gorilla/websocket**: WebSocket support (BSD-3-Clause)
- **prometheus/client_golang**: Metrics collection (Apache 2.0)
- **go.uber.org/zap**: High-performance logging (MIT)
- **github.com/spf13/cobra**: CLI framework (Apache 2.0)

## Development Workflow

### Setup and Common Commands

```bash
# Initial setup
just dev-setup                     # Set up Go + TypeScript development environment
just install-hooks                 # Install pre-commit hooks

# Root-level Development Commands
just dev                           # Full dev cycle (format + lint + test)
just check                         # Quick check (format + lint + typecheck)
just fmt                           # Format all code
just lint                          # Lint all code
just test                          # Run all tests
just test-cov                      # Run tests with coverage
just build                         # Build all packages
just build-rust                    # Build Rust components only

# Go Gateway Development
cd go-gateway
just run                           # Build and run gateway
just bench                         # Run benchmarks
just perf-test                     # Run performance tests
make dev                           # Run in development mode with hot reload
make build                         # Build production binary
make test                          # Run all tests with coverage
make bench                         # Run performance benchmarks
make lint                          # Lint Go code
make fmt                           # Format Go code

# VS Code Extension Development  
cd vscode-extension
npm install                        # Install TypeScript-Go dependencies
npm run compile                    # Compile with TypeScript-Go (10x faster)
npm run watch                      # Watch mode for development
npm run test                       # Run extension tests

# Cross-platform Builds
make build-all                     # Build for Linux, macOS, Windows (AMD64/ARM64)
make docker-build                  # Build Docker container
```

### Component Development

Work is organized by component with clear boundaries:

- **go-gateway/**: Go backend with independent versioning
- **vscode-extension/**: TypeScript frontend with VS Code integration
- **virtual-devices/**: Testing framework with device simulators
- **docs/**: Comprehensive documentation and examples

### Binary Distribution

Production deployment uses single binaries:

- `bifrost-gateway-linux-amd64`: ~15MB production binary
- `bifrost-gateway-windows-amd64.exe`: Windows deployment
- `bifrost-gateway-darwin-arm64`: macOS Apple Silicon
- Cross-platform builds automated via GitHub Actions

## Development Phases

1. **✅ Foundation Complete**: Go gateway infrastructure and architecture
1. **✅ Modbus Implementation Complete**: Production-ready Modbus TCP/RTU with proven performance
1. **🔄 VS Code Extension**: TypeScript-Go integration and industrial tooling
1. **📅 OPC UA Integration**: High-performance OPC UA client/server
1. **📅 Ethernet/IP Support**: Native Go implementation for Allen-Bradley PLCs
1. **📅 Edge Analytics**: Real-time data processing and analytics
1. **📅 Cloud Connectors**: AWS IoT, Azure IoT Hub, Google Cloud IoT
1. **📅 Additional Protocols**: S7, DNP3, BACnet support

## Contributing Guidelines

When implementing this project:

1. Follow Go best practices for concurrent programming with goroutines and channels
1. Use context.Context for request lifecycle and timeout management
1. Implement comprehensive error handling with Go's explicit error patterns
1. Write performance-critical code natively in Go for maximum efficiency
1. Maintain unified REST APIs across different industrial protocols
1. Focus on edge device constraints (memory, CPU, network bandwidth)
1. Create production-ready binaries with minimal dependencies
1. Include comprehensive error handling and structured logging
1. Write extensive tests including performance benchmarks and integration tests
1. Use the Go toolchain (go mod, go test, go build) with Make for development
1. Follow existing Go code conventions and patterns
1. Never add comments unless explicitly requested
1. Prioritize security - never expose or log secrets, use secure defaults

## API Design Philosophy

- **REST-First**: Clean HTTP APIs for cross-platform integration
- **Context Lifecycle**: Request scoping and timeout management via context.Context
- **Type Safety**: Go's compile-time type checking and interface contracts
- **Intuitive Naming**: Clear, descriptive endpoint and method names
- **Progressive Disclosure**: Simple operations simple, complex operations possible

## Security Considerations

- **TLS/SSL**: HTTPS endpoints with proper certificate validation
- **Input Validation**: Comprehensive request validation and sanitization
- **Authentication**: Token-based authentication for production deployments
- **Network Security**: Configurable firewall rules and network access controls
- **Audit Logging**: Structured logging for security events and compliance

## Deployment Scenarios

1. **Edge Gateway**: Single binary deployment collecting from multiple PLCs with cloud forwarding
1. **Industrial IoT Hub**: Central gateway serving multiple applications via REST API
1. **Development Environment**: VS Code extension connecting to local or remote gateways
1. **Container Deployment**: Docker containers for cloud and Kubernetes environments

## Key Files to Reference

- `README.md`: Project vision and overview  
- `go-gateway/README.md`: Go gateway documentation and API reference
- `go-gateway/TEST_RESULTS.md`: Performance test results and benchmarks
- `vscode-extension/TYPESCRIPT_GO_ANALYSIS.md`: TypeScript-Go integration analysis
- `docs/PROJECT_STRUCTURE.md`: Updated project structure documentation
- `virtual-devices/README.md`: Virtual device testing framework documentation
- `justfile`: Modern task runner with all development commands
- `go-gateway/Makefile`: Go-specific build and development commands
