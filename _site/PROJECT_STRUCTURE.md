# Bifrost Project Structure

## Gateway Architecture Strategy

### High-Performance Gateway Approach

Bifrost uses a high-performance gateway architecture that prioritizes production deployment and developer experience:

- **Go Gateway**: Single binary backend with production-ready performance
- **TypeScript-Go Frontend**: VS Code extension with 10x faster compilation
- **Virtual Testing**: Comprehensive device simulation framework

## Component Structure

### Core Components

#### `go-gateway/`

**Purpose**: High-performance industrial gateway backend\
**Technology**: Go 1.22+ with native compilation\
**Size**: ~15MB single binary

```bash
# Production deployment
./bifrost-gateway-linux-amd64
# Serves REST API on :8080, WebSocket on /ws
```

#### `vscode-extension/`

**Purpose**: TypeScript-Go powered development environment\
**Technology**: TypeScript-Go (10x faster compilation)\
**Integration**: VS Code extension marketplace

```typescript
// VS Code extension features
export class DeviceProvider implements vscode.TreeDataProvider<DeviceItem>
export class GatewayClient  // REST API integration
export class WebSocketService  // Real-time data streaming
```

#### `virtual-devices/`

**Purpose**: Comprehensive testing and simulation framework\
**Technology**: Python simulators with Go integration\
**Coverage**: Device simulators, network conditions, performance testing

```bash
# Device simulation
python virtual-devices/simulators/modbus/modbus_server.py
python virtual-devices/simulators/opcua/opcua_server.py
```

### Protocol Implementations

#### `internal/protocols/` (Go Gateway)

**Current**: Production-ready Modbus TCP/RTU implementation\
**Performance**: 18,879 ops/sec with 53µs latency\
**Future**: OPC UA, Ethernet/IP, S7 support

```go
type ProtocolHandler interface {
    Connect(device *Device) error
    ReadTag(device *Device, tag *Tag) (interface{}, error)
    WriteTag(device *Device, tag *Tag, value interface{}) error
    // ... comprehensive protocol interface
}
```

## Repository Structure

```
bifrost/
├── go-gateway/              # Go-based industrial gateway
│   ├── cmd/
│   │   ├── gateway/         # Main server binary
│   │   └── performance_test/
│   ├── internal/
│   │   ├── protocols/       # Protocol implementations
│   │   │   ├── modbus.go    # Production-ready Modbus
│   │   │   ├── ethernetip.go # Future Ethernet/IP
│   │   │   └── protocol.go  # Protocol interface
│   │   ├── gateway/         # Core gateway logic
│   │   │   └── server.go
│   │   └── performance/     # Performance optimizations
│   ├── configs/             # Configuration files
│   ├── examples/            # Usage examples
│   ├── bin/                 # Compiled binaries
│   ├── Makefile             # Go build system
│   └── go.mod
├── vscode-extension/        # TypeScript-Go VS Code extension
│   ├── src/
│   │   ├── extension.ts     # Main extension logic
│   │   ├── services/        # Device management
│   │   │   ├── deviceManager.ts
│   │   │   └── gatewayClient.ts
│   │   ├── providers/       # VS Code providers
│   │   └── utils/           # Utility functions
│   ├── package.json
│   └── tsconfig.json
├── virtual-devices/         # Testing framework
│   ├── simulators/          # Device simulators
│   │   ├── modbus/          # Modbus TCP/RTU simulators
│   │   ├── opcua/           # OPC UA server simulators
│   │   └── ethernetip/      # Ethernet/IP simulators
│   ├── mocks/               # Lightweight mocks
│   ├── scenarios/           # Industrial scenarios
│   │   ├── factory_floor/
│   │   ├── process_control/
│   │   └── scada/
│   └── benchmarks/          # Performance testing
├── docs/                    # Documentation
├── examples/                # Usage examples
├── .github/                 # GitHub Actions workflows
├── justfile                 # Task runner
└── README.md
```

## Deployment Patterns

### For Different Use Cases

#### Production Gateway

```bash
# Single binary deployment
wget https://github.com/bifrost/gateway/releases/latest/download/bifrost-gateway-linux-amd64
chmod +x bifrost-gateway-linux-amd64
./bifrost-gateway-linux-amd64
# ~15MB, no dependencies
```

#### Development Environment

```bash
# Complete development setup
git clone https://github.com/bifrost/bifrost
cd bifrost
just dev-setup  # Sets up Go + TypeScript-Go environment

# Go gateway development
cd go-gateway && make dev

# VS Code extension development
cd vscode-extension && npm run watch
```

#### Docker Deployment

```bash
# Container deployment
docker pull bifrost/gateway:latest
docker run -p 8080:8080 -p 9090:9090 bifrost/gateway:latest

# Kubernetes
kubectl apply -f k8s/bifrost-gateway.yaml
```

#### Virtual Testing

```bash
# Start device simulators
cd virtual-devices/simulators/modbus
python modbus_server.py --port 502

# Run performance tests
cd go-gateway
make test && make bench
```

### Configuration Management

Gateway uses YAML configuration with environment variable overrides:

```yaml
# gateway.yaml
gateway:
  port: 8080
  grpc_port: 9090
  max_connections: 1000
  data_buffer_size: 10000

protocols:
  modbus:
    default_timeout: 5s
    max_connections: 100
    connection_timeout: 10s
```

## Build System

### Go-First Toolchain

- **Go Modules**: Native dependency management
- **Make**: Cross-platform build automation
- **Task Runner**: `just` for development workflows
- **Testing**: Go testing framework with benchmarks
- **Cross-compilation**: Automated multi-platform builds

### Development Workflows

```bash
# Go gateway development
cd go-gateway
make dev          # Development mode with hot reload
make build        # Production binary
make test         # Run all tests with coverage
make bench        # Performance benchmarks

# VS Code extension development
cd vscode-extension
npm install       # TypeScript-Go dependencies
npm run compile   # 10x faster compilation
npm run watch     # Development watch mode

# Cross-platform builds
make build-all    # Linux, macOS, Windows (AMD64/ARM64)
make docker-build # Container images
```

## Performance Characteristics

### Binary Deployment

- **Gateway Binary**: ~15MB single executable
- **Memory Usage**: < 50MB base footprint
- **Startup Time**: Sub-second initialization
- **Dependencies**: Zero runtime dependencies

### Performance Metrics

- **Throughput**: 18,879 operations/second (proven)
- **Latency**: 53µs average response time
- **Concurrency**: 1000+ simultaneous connections
- **Reliability**: 100% success rate in testing

## User Experience

### Deployment Simplicity

Users can choose their deployment model:

```bash
# Production deployment
./bifrost-gateway-linux-amd64  # Single binary, no dependencies

# Development environment
code --install-extension bifrost.industrial-gateway

# Container deployment
docker run bifrost/gateway:latest

# Development from source
git clone && just dev-setup
```

### Clear Documentation

Each component has focused documentation:

- `go-gateway/`: Production deployment and API reference
- `vscode-extension/`: Development environment setup
- `virtual-devices/`: Testing and simulation framework
- `docs/`: Architecture and specifications

## Benefits

### For Users

- **Single Binary**: No dependency management or runtime issues
- **High Performance**: Native Go speed with proven benchmarks
- **Production Ready**: Comprehensive monitoring and error handling
- **Developer Friendly**: VS Code integration with TypeScript-Go

### For Operators

- **Simple Deployment**: Single binary with YAML configuration
- **Observability**: Prometheus metrics and structured logging
- **Reliability**: Comprehensive error handling and graceful shutdown
- **Scalability**: 1000+ concurrent connections tested

### For Developers

- **Modern Stack**: Go backend with TypeScript-Go frontend
- **Fast Iteration**: 10x faster compilation and hot reload
- **Comprehensive Testing**: Virtual device framework
- **Clear Architecture**: Well-defined interfaces and separation of concerns
