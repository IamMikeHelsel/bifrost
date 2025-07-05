# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bifrost is a high-performance Python framework for industrial edge computing that bridges OT (Operational Technology) equipment with modern IT infrastructure. The project aims to provide unified APIs for industrial protocols (Modbus, OPC UA, Ethernet/IP, S7) with Rust-powered performance optimizations.

### Mission Statement

Break down the walls between operational technology and information technology. Make it as easy to work with a PLC as it is to work with a REST API. Help automation professionals leverage modern tools without abandoning what works.

### Target Users

- **Control Systems Engineers** tired of duct-taping solutions together
- **Automation Engineers** who want modern development tools
- **SCADA/HMI Developers** looking for better Python libraries
- **IT Developers** who need to understand industrial equipment
- **System Integrators** seeking reliable, performant tools
- **Process Engineers** trying to get data into analytics platforms

## Architecture

### Package Architecture

**Hybrid Modular Approach** - Multiple packages in a single monorepo:

```
bifrost/
├── packages/
│   ├── bifrost-core/        # Core abstractions (~10MB)
│   │   ├── src/bifrost_core/
│   │   ├── tests/
│   │   └── pyproject.toml
│   ├── bifrost/             # Main package + Modbus + CLI (~50MB)  
│   │   ├── src/bifrost/
│   │   │   ├── core/        # Core utilities and base classes
│   │   │   ├── plc/         # PLC communication toolkit
│   │   │   │   ├── modbus/
│   │   │   │   └── drivers/
│   │   │   └── cli/         # Beautiful command-line interface
│   │   │       ├── commands/
│   │   │       ├── display/
│   │   │       └── themes/
│   │   ├── tests/
│   │   └── pyproject.toml
│   ├── bifrost-opcua/       # OPC UA implementation (~100MB)
│   │   ├── src/bifrost_opcua/
│   │   ├── native/          # Rust code
│   │   └── pyproject.toml
│   ├── bifrost-analytics/   # Edge analytics + Rust (~80MB)
│   │   ├── src/bifrost_analytics/
│   │   │   ├── timeseries/
│   │   │   ├── analytics/
│   │   │   └── pipelines/
│   │   ├── native/          # Rust code
│   │   └── pyproject.toml
│   ├── bifrost-cloud/       # Cloud connectors (~60MB)
│   │   ├── src/bifrost_cloud/
│   │   │   ├── connectors/
│   │   │   ├── buffering/
│   │   │   └── security/
│   │   └── pyproject.toml
│   ├── bifrost-protocols/   # Additional protocols (~40MB)
│   │   ├── src/bifrost_protocols/
│   │   └── pyproject.toml
│   ├── bifrost-web/         # Web API and dashboard (optional)
│   │   └── pyproject.toml
│   └── bifrost-all/         # Meta-package (~300MB total)
│       └── pyproject.toml
├── tools/                   # Build and release scripts
├── docs/                    # Documentation
├── examples/                # Usage examples
├── scripts/                 # Utility scripts
├── .github/                 # GitHub Actions workflows
├── justfile                 # Task runner
├── pyproject.toml          # Workspace config
├── uv.lock                 # Lock file
└── README.md
```

### Installation Patterns

**For Different Use Cases**:

```bash
# Edge Gateway (Minimal)
uv add bifrost-core bifrost-protocols  # ~50MB

# Basic Development
uv add bifrost                          # ~50MB, Modbus + CLI included

# OPC UA Development
uv add bifrost bifrost-opcua            # ~150MB

# Analytics Platform
uv add bifrost bifrost-analytics bifrost-cloud  # ~200MB

# Web Development
uv add bifrost bifrost-web              # ~80MB

# Full Development
uv add bifrost-all                      # ~330MB, everything
```

### Key Design Principles

- **Async-first**: All I/O operations use asyncio
- **Context managers**: Resource management via `async with`
- **Type safety**: Full type hints and runtime validation using Pydantic
- **Performance-critical native code**: Rust extensions via PyO3 for protocol parsing and data processing
- **Unified API**: Protocol-agnostic interfaces for different industrial protocols
- **Beautiful CLI**: Rich terminal interface with colors, progress bars, and interactive wizards

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

## Development Status

This is currently a **planning phase project** with extensive specifications but minimal implementation. The repository contains:

- High-level vision and architecture documents
- Detailed technical specifications
- Development roadmap spanning 12-18 months
- Initial implementation work has begun

## Performance Targets

The project aims for:

- **OPC UA**: 10,000+ tags/second read throughput
- **Modbus TCP**: < 1ms round-trip for single register
- **Stream Processing**: 100,000+ events/second on Raspberry Pi 4
- **Memory Usage**: < 100MB base footprint

## Core Components

### 1. High-Performance OPC UA Module (`bifrost.opcua`)

- Client/Server implementation with all security profiles
- 10,000+ tags/second via Rust backend
- Wraps open62541 C library with Rust safety layer

### 2. Unified PLC Communication Toolkit (`bifrost.plc`)

- **Modbus** (RTU/TCP): Rust-based engine for high-speed polling
- **Ethernet/IP (CIP)**: Modern replacement for cpppo
- **S7 (Siemens)**: Native performance via snap7 integration
- Protocol-agnostic API with automatic type conversion

### 3. Edge Analytics Engine (`bifrost.edge`)

- In-memory time-series engine (Rust-based)
- Stream processing pipeline API
- Built-in filtering, windowing, aggregation, anomaly detection
- Automatic memory management for constrained devices

### 4. Edge-to-Cloud Bridge (`bifrost.bridge`)

- Support for AWS IoT Core, Azure IoT Hub, Google Cloud IoT
- Smart buffering with disk-backed queue and compression
- Retry logic with exponential backoff
- End-to-end encryption and certificate management

### 5. Beautiful CLI (`bifrost.cli`)

**Design Philosophy**:
- Visual clarity with rich colors and formatting
- Interactive experience with progress bars and real-time feedback
- Professional aesthetics that inspire confidence

**Color Coding System**:
- 🟢 **Green**: Success states, healthy connections, normal values
- 🟡 **Yellow**: Warnings, thresholds approaching, configuration needed
- 🔴 **Red**: Errors, failed connections, critical alerts
- 🔵 **Blue**: Information, headers, navigation elements
- 🟣 **Purple**: Special states, advanced features, admin functions

**Core Commands**:
```bash
bifrost discover                    # Network device discovery
bifrost connect modbus://10.0.0.100 # Interactive connection wizard
bifrost monitor --dashboard         # Live monitoring dashboard
bifrost export --format csv         # Data export with progress
```

## Technology Stack

### Core Technologies

- **Python**: 3.13+ (leveraging latest performance improvements)
- **Package Manager**: uv (10-100x faster than pip)
- **Linting**: ruff (10-100x faster than black/flake8)
- **Task Runner**: just (cross-platform, better than make)
- **Build System**: setuptools + maturin for Rust extensions
- **Testing**: pytest + pytest-asyncio for async testing
- **Type Checking**: mypy with strict configuration
- **Documentation**: Sphinx with modern theme

### Native Performance

- **Rust**: PyO3 for native extensions, maturin for builds
- **Async Runtime**: asyncio with uvloop for production
- **Serialization**: orjson (fast JSON), msgpack (binary)
- **Compression**: zstd for efficient data storage

### CLI/TUI Stack

- **Rich**: Modern terminal formatting and colors
- **Typer**: Type-safe CLI framework with automatic help
- **Textual**: TUI framework for dashboard mode
- **Click**: Fallback for complex command structures

### Target Platforms

- **Operating Systems**: Linux (primary), Windows, macOS
- **Architectures**: x86_64, ARM64 (including Raspberry Pi)
- **Deployment**: Edge devices, industrial PCs, cloud environments

## Library Strategy

### Open Source Dependencies

**Selection Criteria**:
- Permissive licensing (MIT, Apache 2.0, BSD)
- Active maintenance and security updates
- Performance proven in production

**Core Protocol Libraries**:
- **pymodbus**: Mature Modbus library (BSD)
- **asyncua**: OPC UA client/server (LGPL - evaluate alternatives)
- **open62541**: OPC UA C library (Mozilla Public License)
- **snap7**: Siemens S7 communication (MIT)
- **cpppo**: Ethernet/IP support (GPL - need permissive alternative)

**Performance Libraries**:
- **uvloop**: High-performance event loop (Apache 2.0)
- **msgpack**: Fast serialization (Apache 2.0)
- **orjson**: Fast JSON parsing (Apache 2.0)
- **pydantic**: Data validation (MIT)

## Development Workflow

### Setup and Common Commands

```bash
# Initial setup
just dev-setup

# Development cycle
just dev          # format + lint + test
just check        # quick format + lint + typecheck

# Individual tasks  
just fmt          # format all code
just lint         # lint with auto-fix
just test         # run all tests
just build        # build all packages

# Cross-package development
just dev-install  # Install all packages in dev mode
just test-all     # Run tests across all packages
just build-all    # Build all packages
just release      # Release with synchronized versions
```

### Package Development

Work is organized in the `packages/` directory with each package having:
- Independent `pyproject.toml` configuration
- Focused dependencies and scope
- Smart imports with helpful error messages
- Synchronized versioning across all packages

### Version Synchronization

All bifrost packages maintain synchronized versions:
- `bifrost-core`: 1.0.0
- `bifrost`: 1.0.0 (depends on bifrost-core ^1.0)
- `bifrost-opcua`: 1.0.0 (depends on bifrost-core ^1.0)
- etc.

## Development Phases

1. **Foundation** (Months 1-2): Project infrastructure and core architecture
2. **PLC Communication MVP** (Months 2-4): Modbus implementation with Rust engine
3. **OPC UA Integration** (Months 4-7): High-performance OPC UA client/server
4. **Edge Analytics Engine** (Months 6-9): Time-series processing for edge devices
5. **Beautiful CLI Development** (Months 9-11): Rich terminal interface
6. **Cloud Bridge Framework** (Months 8-11): Edge-to-cloud connectivity
7. **Additional Protocol Support** (Months 10-12): Ethernet/IP, S7, etc.
8. **Production Hardening** (Months 11-14): Testing, documentation, deployment tools

## Contributing Guidelines

When implementing this project:

1. Follow async-first patterns for all I/O operations
2. Use context managers for resource management
3. Implement comprehensive type hints
4. Write performance-critical code in Rust with PyO3 bindings
5. Maintain unified APIs across different protocols
6. Focus on edge device constraints (memory, CPU, network)
7. Create beautiful, intuitive CLI interfaces with rich visual feedback
8. Include comprehensive error handling and logging
9. Write extensive tests including performance benchmarks
10. Use the modern toolchain (uv, ruff, just) for development
11. Follow existing code conventions and patterns
12. Never add comments unless explicitly requested
13. Prioritize security - never expose or log secrets

## API Design Philosophy

- **Async-First**: All I/O operations are async by default
- **Context Managers**: Resource management via `async with`
- **Type Safety**: Full type hints and runtime validation
- **Intuitive Naming**: Clear, descriptive function names
- **Progressive Disclosure**: Simple tasks simple, complex tasks possible

## Security Considerations

- **OPC UA Security**: Full implementation of security profiles
- **TLS/SSL**: For all network communications
- **Certificate Management**: Built-in PKI utilities
- **Secrets Management**: Integration with HashiCorp Vault, AWS Secrets Manager
- **Audit Logging**: Comprehensive logging for compliance

## Deployment Scenarios

1. **Edge Gateway**: Collect from multiple PLCs, perform analytics, forward to cloud
2. **SCADA Integration**: OPC UA server exposing PLC data with real-time analytics
3. **Digital Twin Synchronization**: High-frequency collection with reliable cloud sync

## Key Files to Reference

- `README.md`: Project vision and overview
- `bifrost_spec.md`: Detailed technical specifications and API examples
- `bifrost_dev_roadmap.md`: Development timeline and implementation phases
- `PROJECT_STRUCTURE.md`: Detailed package architecture and installation patterns
- `justfile`: Modern task runner with all development commands
- `pyproject.toml`: Workspace configuration and tooling setup
- `packages/*/pyproject.toml`: Individual package configurations
