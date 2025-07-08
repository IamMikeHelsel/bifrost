# Bifrost Bazel Build System Guide

## Overview

Bifrost has migrated to Bazel for monorepo management, supporting heterogeneous languages (Python, Rust, C/C++) with package-specific build rules.

## Quick Start

### Prerequisites
- Bazel 7.0.0+ (specified in `.bazelversion`)
- Python 3.13
- Rust toolchain (for future components)

### Basic Commands

```bash
# Build all packages
bazel build //packages/...

# Run all tests  
bazel test //packages/...

# Build specific package
bazel build //packages/bifrost-core:bifrost_core

# Build wheels
bazel build //packages/...:wheel

# Run CLI
bazel run //packages/bifrost:cli
```

### Justfile Integration

```bash
# Bazel commands through justfile
just build-bazel     # Build with Bazel
just test-bazel      # Test with Bazel  
just bazel-dev       # Full development cycle
just build-wheels    # Build distribution wheels

# Legacy uv commands still available
just build          # Legacy build
just test           # Legacy test
```

## Package Structure

### Current Packages
- `packages/bifrost-core/` - Python-only core abstractions
- `packages/bifrost/` - Main package with CLI
- `packages/bifrost-all/` - Meta-package

### Future Packages (Prepared)
- `packages/bifrost-opcua/` - OPC UA (Python + Rust + Native)
- `packages/bifrost-analytics/` - Analytics (Python + Rust)
- `packages/bifrost-cloud/` - Cloud integrations (Python-only)
- `packages/bifrost-protocols/` - Additional protocols (Python + Rust)

## BUILD File Examples

### Python-only Package
```python
load("@rules_python//python:defs.bzl", "py_library", "py_test")
load("@rules_python//python:packaging.bzl", "py_wheel")

py_library(
    name = "my_package",
    srcs = glob(["src/**/*.py"]),
    deps = ["@pypi//pydantic"],
    visibility = ["//visibility:public"],
)

py_wheel(
    name = "wheel",
    distribution = "my-package",
    version = "0.1.0",
    deps = [":my_package"],
)
```

### Python + Rust Package (Future)
```python
load("@rules_rust//rust:defs.bzl", "rust_shared_library")

rust_shared_library(
    name = "native_extension",
    srcs = glob(["native/src/**/*.rs"]),
    deps = ["@crate_index//:pyo3"],
    crate_features = ["pyo3/extension-module"],
)

py_library(
    name = "my_package",
    srcs = glob(["src/**/*.py"]),
    data = [":native_extension"],
    deps = ["@pypi//pydantic"],
)
```

## Configuration Files

### MODULE.bazel
- Python 3.13 toolchain
- Rust toolchain (edition 2021)
- pip_parse for PyPI dependencies
- crate_universe for Cargo dependencies

### .bazelrc
- Performance optimizations
- CI/CD configurations
- Platform-specific settings
- Remote caching preparation

### Cargo.toml
- Rust workspace configuration
- Common dependencies for future Rust components
- Optimized build profiles

## Development Workflow

### 1. Make Changes
```bash
# Edit code in packages/*/src/
```

### 2. Build and Test
```bash
just bazel-dev  # or individually:
just build-bazel
just test-bazel
```

### 3. Query Dependencies
```bash
just deps //packages/bifrost:bifrost
just rdeps //packages/bifrost-core:bifrost_core
```

### 4. Clean Build
```bash
just clean-bazel      # Clean build cache
just clean-bazel-all  # Clean everything
```

## Performance Benefits

### Incremental Builds
- Only changed components are rebuilt
- Parallel compilation across packages
- Shared caching between developers (future)

### Expected Improvements
- Initial build: Similar to current system
- Incremental builds: 90%+ faster
- CI builds: 50-70% faster with remote caching
- Rust compilation: Parallel across packages

## Migration Status

### ✅ Completed
- Foundation setup (Python 3.13, Rust toolchains)
- BUILD files for current packages
- Justfile integration
- Performance optimization
- Future package infrastructure

### 🔄 In Progress
- None (initial migration complete)

### 📋 Future Work
- Implement Rust components in packages
- Set up remote caching infrastructure
- Add native library integrations (open62541, snap7)
- Optimize CI/CD pipeline

## Troubleshooting

### Common Issues

**Bazel not found:**
```bash
# Bazel version specified in .bazelversion
cat .bazelversion  # Should show 7.0.0
```

**Python dependencies not found:**
```bash
# Dependencies managed through requirements_lock.txt
# Parsed by MODULE.bazel pip extension
```

**Build cache issues:**
```bash
just clean-bazel-all  # Nuclear option
```

### Query Commands
```bash
# List all targets
bazel query //...

# Show dependencies
bazel query "deps(//packages/bifrost:bifrost)"

# Show reverse dependencies  
bazel query "rdeps(//..., //packages/bifrost-core:bifrost_core)"

# Show cache info
bazel info
```

## Contributing

When adding new packages:

1. Create package directory structure
2. Add BUILD.bazel file with appropriate rules
3. Update root BUILD.bazel with aliases
4. Add tests with py_test rules
5. Update justfile if needed

For Rust components:
1. Add to Cargo.toml workspace members
2. Generate Cargo.lock: `cargo generate-lockfile`
3. Remove Cargo.* from .bazelignore
4. Update MODULE.bazel crate_universe configuration

This migration provides the foundation for Bifrost's ambitious industrial IoT framework with robust, scalable build system.