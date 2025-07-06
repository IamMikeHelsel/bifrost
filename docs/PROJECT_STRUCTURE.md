# Bifrost Project Structure

## Package Architecture Strategy

### Hybrid Modular Approach

Bifrost uses a hybrid modular packaging strategy that balances simplicity with flexibility:

- **Core Package**: Essential abstractions and common protocols
- **Extension Packages**: Heavy dependencies and specialized features
- **Meta Packages**: Convenience bundles for different use cases

## Package Structure

### Core Packages

#### `bifrost-core`

**Purpose**: Lightweight foundation with essential abstractions\
**Dependencies**: Minimal (asyncio, pydantic, typing)\
**Size**: ~10MB

```python
# Core abstractions
from bifrost_core import (
    BaseConnection, BaseProtocol, DataPoint, 
    Pipeline, ConnectionPool, EventBus
)
```

#### `bifrost`

**Purpose**: Main package with most common functionality\
**Dependencies**: bifrost-core + pymodbus + rich + typer\
**Size**: ~50MB

```python
# Complete basic functionality
from bifrost import (
    ModbusConnection, PLCConnection, 
    CLI, discover_devices, connect
)
```

### Extension Packages

#### `bifrost-opcua`

**Purpose**: High-performance OPC UA implementation\
**Dependencies**: bifrost-core + asyncua + open62541 bindings\
**Size**: ~100MB (includes native libraries)

```python
from bifrost_opcua import OPCUAClient, OPCUAServer
```

#### `bifrost-analytics`

**Purpose**: Edge analytics and time-series processing\
**Dependencies**: bifrost-core + Rust components + numpy\
**Size**: ~80MB

```python
from bifrost_analytics import (
    TimeSeriesEngine, StreamProcessor, 
    AnomalyDetector, Pipeline
)
```

#### `bifrost-cloud`

**Purpose**: Cloud connectivity and bridge framework\
**Dependencies**: bifrost-core + cloud SDKs + messaging\
**Size**: ~60MB

```python
from bifrost_cloud import (
    CloudBridge, AWSConnector, AzureConnector,
    MQTTBridge, BufferedQueue
)
```

#### `bifrost-protocols`

**Purpose**: Additional protocol implementations\
**Dependencies**: bifrost-core + protocol-specific libraries\
**Size**: ~40MB

```python
from bifrost_protocols import (
    EthernetIPConnection, S7Connection,
    DNP3Connection, BACnetConnection
)
```

### Meta Packages

#### `bifrost-web`

**Purpose**: Web API and dashboard components (optional)\
**Dependencies**: bifrost-core + FastAPI + web frameworks\
**Size**: ~30MB

```python
from bifrost_web import WebAPI, Dashboard, MonitoringApp

# REST API for integration
api = WebAPI()
app = api.create_fastapi_app()

# Web dashboard for monitoring
dashboard = Dashboard()
await dashboard.serve(port=8080)
```

#### `bifrost-all`

**Purpose**: Complete installation for full development\
**Dependencies**: All bifrost packages\
**Size**: ~330MB total

```python
# Everything available
from bifrost import *
from bifrost_opcua import *
from bifrost_analytics import *
from bifrost_cloud import *
from bifrost_protocols import *
from bifrost_web import *
```

## Monorepo Structure

```
bifrost/
├── packages/
│   ├── bifrost-core/
│   │   ├── src/bifrost_core/
│   │   ├── tests/
│   │   ├── pyproject.toml
│   │   └── README.md
│   ├── bifrost/
│   │   ├── src/bifrost/
│   │   ├── tests/
│   │   ├── pyproject.toml
│   │   └── README.md
│   ├── bifrost-opcua/
│   │   ├── src/bifrost_opcua/
│   │   ├── native/          # Rust code
│   │   ├── tests/
│   │   ├── pyproject.toml
│   │   └── README.md
│   ├── bifrost-analytics/
│   │   ├── src/bifrost_analytics/
│   │   ├── native/          # Rust code
│   │   ├── tests/
│   │   ├── pyproject.toml
│   │   └── README.md
│   ├── bifrost-cloud/
│   │   ├── src/bifrost_cloud/
│   │   ├── tests/
│   │   ├── pyproject.toml
│   │   └── README.md
│   ├── bifrost-protocols/
│   │   ├── src/bifrost_protocols/
│   │   ├── tests/
│   │   ├── pyproject.toml
│   │   └── README.md
│   └── bifrost-all/
│       ├── pyproject.toml    # Meta-package
│       └── README.md
├── tools/
│   ├── build-all.py
│   ├── release.py
│   └── sync-versions.py
├── docs/
├── examples/
├── scripts/
├── .github/
├── justfile                 # Task runner
├── pyproject.toml          # Workspace config
├── uv.lock                 # Lock file
└── README.md
```

## Installation Patterns

### For Different Use Cases

#### Edge Gateway (Minimal)

```bash
uv add bifrost-core bifrost-protocols
# ~50MB, just what you need
```

#### Basic Development

```bash
uv add bifrost
# ~50MB, Modbus + CLI included
```

#### OPC UA Development

```bash
uv add bifrost bifrost-opcua
# ~150MB, core + OPC UA
```

#### Analytics Platform

```bash
uv add bifrost bifrost-analytics bifrost-cloud
# ~200MB, processing + cloud
```

#### Web Development

```bash
uv add bifrost bifrost-web
# ~80MB, core + web APIs
```

#### Full Development

```bash
uv add bifrost-all
# ~330MB, everything
```

### Smart Import System

Each package includes smart imports with helpful error messages:

```python
# In bifrost/__init__.py
try:
    from bifrost_opcua import OPCUAClient
except ImportError:
    def OPCUAClient(*args, **kwargs):
        raise ImportError(
            "OPC UA support requires: pip install bifrost-opcua\n"
            "Or for everything: pip install bifrost-all"
        )
```

## Build System

### Modern Python Toolchain

- **Package Manager**: `uv` (10-100x faster than pip)
- **Linting**: `ruff` (10-100x faster than black/flake8)
- **Testing**: `pytest` with `pytest-asyncio`
- **Task Runner**: `just` (cross-platform, better than make)
- **Python-Rust**: `maturin` for seamless integration

### Cross-Package Development

```bash
# Install all packages in development mode
just dev-install

# Run tests across all packages
just test-all

# Build all packages
just build-all

# Release all packages with synchronized versions
just release
```

## Dependency Management

### Dependency Isolation

Each package has minimal, focused dependencies:

```toml
# bifrost-core/pyproject.toml
[tool.uv.dependencies]
python = ">=3.13"
pydantic = "^2.0"
typing-extensions = "^4.0"

# bifrost/pyproject.toml
[tool.uv.dependencies]
python = ">=3.13"
bifrost-core = "^1.0"
pymodbus = "^3.0"
rich = "^13.0"
typer = "^0.12"

# bifrost-opcua/pyproject.toml
[tool.uv.dependencies]
python = ">=3.13"
bifrost-core = "^1.0"
asyncua = "^1.0"
```

### Version Synchronization

All bifrost packages maintain synchronized versions:

- `bifrost-core`: 1.0.0
- `bifrost`: 1.0.0 (depends on bifrost-core ^1.0)
- `bifrost-opcua`: 1.0.0 (depends on bifrost-core ^1.0)
- etc.

## User Experience

### Progressive Installation

Users can start minimal and add features:

```bash
# Start simple
uv add bifrost

# Add OPC UA later
uv add bifrost-opcua

# Add analytics
uv add bifrost-analytics

# Or get everything
uv add bifrost-all
```

### Clear Documentation

Each package has focused documentation:

- `bifrost-core`: Architecture and extending
- `bifrost`: Getting started and basic usage
- `bifrost-opcua`: OPC UA specific examples
- `bifrost-analytics`: Edge processing examples
- `bifrost-cloud`: Cloud integration patterns

## Benefits

### For Users

- **Minimal installs**: Only pay for what you use
- **Fast installs**: Smaller packages, pre-built wheels
- **Clear dependencies**: Know what you're getting
- **Progressive complexity**: Start simple, add features

### For Maintainers

- **Single repo**: Coordinated development
- **Shared tooling**: Consistent build/test/release
- **Isolated concerns**: Clear package boundaries
- **Synchronized releases**: All packages move together

### For Contributors

- **Focused changes**: Work on specific packages
- **Clear interfaces**: Well-defined package boundaries
- **Modern tooling**: Fast, reliable development environment
- **Good testing**: Comprehensive test coverage
