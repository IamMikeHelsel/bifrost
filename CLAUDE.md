# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bifrost is a high-performance Python framework for industrial edge computing that bridges OT (Operational Technology) equipment with modern IT infrastructure. The project aims to provide unified APIs for industrial protocols (Modbus, OPC UA, Ethernet/IP, S7) with Rust-powered performance optimizations.

## Architecture

### Package Architecture

**Hybrid Modular Approach** - Multiple packages in a single monorepo:

```
bifrost/
├── packages/
│   ├── bifrost-core/     # Core abstractions (~10MB)
│   ├── bifrost/          # Main package + Modbus + CLI (~50MB)  
│   ├── bifrost-opcua/    # OPC UA implementation (~100MB)
│   ├── bifrost-analytics/# Edge analytics + Rust (~80MB)
│   ├── bifrost-cloud/    # Cloud connectors (~60MB)
│   ├── bifrost-protocols/# Additional protocols (~40MB)
│   └── bifrost-all/      # Meta-package (~300MB total)
├── tools/               # Build and release scripts
├── docs/               # Documentation
└── examples/           # Usage examples
```

**Installation Patterns**:
- **Basic**: `uv add bifrost` (Modbus + CLI)
- **OPC UA**: `uv add bifrost[opcua]`  
- **Analytics**: `uv add bifrost[analytics]`
- **Everything**: `uv add bifrost[all]`

### Key Design Principles

- **Async-first**: All I/O operations use asyncio
- **Context managers**: Resource management via `async with`
- **Type safety**: Full type hints and runtime validation using Pydantic
- **Performance-critical native code**: Rust extensions via PyO3 for protocol parsing and data processing
- **Unified API**: Protocol-agnostic interfaces for different industrial protocols

## Development Status

This is currently a **planning phase project** with extensive specifications but minimal implementation. The repository contains:

- High-level vision and architecture documents
- Detailed technical specifications
- Development roadmap spanning 12-18 months
- No actual code implementation yet

## Performance Targets

The project aims for:

- OPC UA: 10,000+ tags/second read throughput
- Modbus TCP: < 1ms round-trip for single register
- Stream Processing: 100,000+ events/second on Raspberry Pi 4
- Memory Usage: < 100MB base footprint

## Planned Development Phases

1. **Foundation** (Months 1-2): Project infrastructure and core architecture
1. **PLC Communication MVP** (Months 2-4): Modbus implementation with Rust engine
1. **OPC UA Integration** (Months 4-7): High-performance OPC UA client/server
1. **Edge Analytics Engine** (Months 6-9): Time-series processing for edge devices
1. **Cloud Bridge Framework** (Months 8-11): Edge-to-cloud connectivity
1. **Additional Protocol Support** (Months 10-12): Ethernet/IP, S7, etc.
1. **Production Hardening** (Months 11-14): Testing, documentation, deployment tools

## Technology Stack

### Core Technologies

- **Python**: 3.13+ (leveraging latest performance improvements)
- **Package Manager**: uv (10-100x faster than pip)
- **Linting**: ruff (10-100x faster than black/flake8)
- **Task Runner**: just (cross-platform, better than make)
- **Rust**: PyO3 for native extensions, maturin for builds
- **Async Framework**: asyncio with uvloop for production
- **CLI Framework**: Rich + Typer + Textual for beautiful interfaces
- **Type System**: Pydantic for validation

### Target Platforms

- **Operating Systems**: Linux (primary), Windows, macOS
- **Architectures**: x86_64, ARM64 (including Raspberry Pi)
- **Deployment**: Edge devices, industrial PCs, cloud environments

## Library Strategy

**Open Source First**: Leverage high-quality, maintained libraries where possible

- Prefer permissive licensing (MIT, Apache 2.0, BSD)
- Prioritize actively maintained projects
- Evaluate performance and security track records

**Key Dependencies**:

- Rich, Typer, Textual for CLI/TUI interfaces
- asyncio-mqtt, aiomodbus for protocol support
- uvloop, orjson, msgpack for performance
- pydantic for data validation

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
```

### Package Development

Work is organized in the `packages/` directory with each package having:
- Independent `pyproject.toml` configuration
- Focused dependencies and scope
- Smart imports with helpful error messages
- Synchronized versioning across all packages

## Contributing Guidelines

When implementing this project:

1. Follow async-first patterns for all I/O operations
1. Use context managers for resource management
1. Implement comprehensive type hints
1. Write performance-critical code in Rust with PyO3 bindings
1. Maintain unified APIs across different protocols
1. Focus on edge device constraints (memory, CPU, network)
1. Create beautiful, intuitive CLI interfaces with rich visual feedback
1. Include comprehensive error handling and logging
1. Write extensive tests including performance benchmarks
1. Use the modern toolchain (uv, ruff, just) for development

## Key Files to Reference

- `README.md`: Project vision and overview
- `bifrost_spec.md`: Detailed technical specifications and API examples
- `bifrost_dev_roadmap.md`: Development timeline and implementation phases
- `PROJECT_STRUCTURE.md`: Detailed package architecture and installation patterns
- `justfile`: Modern task runner with all development commands
- `pyproject.toml`: Workspace configuration and tooling setup
- `packages/*/pyproject.toml`: Individual package configurations
