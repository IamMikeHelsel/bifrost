# Bazel Build System Implementation

This document describes the implemented Bazel build system for the Bifrost industrial IoT framework, addressing Issue #15 (Build System Strategy) and implementing Phase 1 of Issue #18 (Bazel Migration Plan).

## Overview

The Bifrost project has been configured with a modern Bazel build system to support:
- **Multi-language development**: Python (with future Rust support)
- **Package-specific build rules**: Different requirements per package
- **Incremental builds**: Only rebuild changed components
- **Reproducible builds**: Hermetic build environment
- **Cross-platform distribution**: Consistent builds across platforms

## Implementation Status

### ✅ Completed (Phase 1 Foundation)

1. **Bazel Workspace Configuration**
   - `MODULE.bazel`: Modern Bazel module system with Python 3.13
   - `WORKSPACE.bazel`: Minimal workspace file
   - `.bazelversion`: Pinned to Bazel 7.0.0 for consistency

2. **Python Package BUILD Files**
   - `packages/bifrost-core/BUILD.bazel`: Core abstractions (Python-only)
   - `packages/bifrost/BUILD.bazel`: Main package with CLI binary

3. **Dependency Management**
   - `requirements_lock.txt`: All PyPI dependencies
   - Configured with `@pip_deps//` repository for Bazel access

4. **Justfile Integration**
   - Updated with Bazel build commands
   - Legacy fallback commands for transition period
   - Bazel-specific query and analysis commands

5. **Validation Tools**
   - `tools/validate_build.py`: BUILD file validation script

### 📋 Package Structure

```
bifrost/
├── MODULE.bazel              # Modern Bazel modules (Python 3.13)
├── WORKSPACE.bazel           # Minimal workspace
├── .bazelversion            # Bazel version pinning
├── requirements_lock.txt     # PyPI dependencies
├── packages/
│   ├── bifrost-core/
│   │   ├── BUILD.bazel       # py_library, py_test, py_wheel
│   │   ├── src/bifrost_core/ # Python source
│   │   └── tests/            # Test files
│   └── bifrost/
│       ├── BUILD.bazel       # py_library, py_binary, py_test, py_wheel
│       ├── src/bifrost/      # Python source + CLI
│       └── tests/            # Test files + benchmarks
└── tools/
    └── validate_build.py     # Build validation
```

## Build Targets

### bifrost-core Package
- `//packages/bifrost-core:bifrost_core` - Core library
- `//packages/bifrost-core:tests` - Test suite
- `//packages/bifrost-core:wheel` - Distribution wheel

### bifrost Package  
- `//packages/bifrost:bifrost` - Main library
- `//packages/bifrost:bifrost_cli` - CLI binary
- `//packages/bifrost:tests` - Test suite
- `//packages/bifrost:wheel` - Distribution wheel

## Build Commands

### Using Justfile (Recommended)
```bash
# Build all packages
just build

# Build specific package
just build-pkg bifrost-core
just build-pkg bifrost

# Build distribution wheels
just build-wheels

# Run all tests
just test

# Run tests for specific package  
just test-pkg bifrost

# Clean builds
just clean-bazel
just clean-all
```

### Direct Bazel Commands
```bash
# Build all packages
bazel build //packages/...

# Build specific targets
bazel build //packages/bifrost-core:bifrost_core
bazel build //packages/bifrost:bifrost_cli

# Run tests
bazel test //packages/...

# Build wheels
bazel build //packages/...:wheel

# Query dependencies
bazel query "deps(//packages/bifrost:bifrost)"
```

## Key Features

### 1. Incremental Builds
- Only rebuilds changed components
- Faster development iteration
- Automatic dependency tracking

### 2. Hermetic Builds
- Reproducible across all environments
- Isolated dependency management
- Consistent build results

### 3. Package-Specific Rules
- `bifrost-core`: Pure Python library
- `bifrost`: Python library + CLI binary
- Future packages can have different build requirements

### 4. Cross-Platform Distribution
- Platform-specific wheel generation
- Consistent build process across OS
- Ready for native library integration

## Development Workflow

### Initial Setup
```bash
# Validate build system
python tools/validate_build.py

# Build everything
just build

# Run tests
just test
```

### Daily Development
```bash
# Quick development cycle
just dev

# Build specific component
just build-pkg bifrost-core

# Run specific tests
just test-pkg bifrost

# Build for distribution
just build-wheels
```

## Dependencies

### Python Dependencies (via pip_deps)
- Core: `pydantic`, `typing-extensions`
- Industrial IoT: `pymodbus`, `rich`, `typer`
- Performance: `uvloop`, `orjson`
- Testing: `pytest`, `pytest-asyncio`, `pytest-benchmark`
- Development: `mypy`, `ruff`

### Bazel Dependencies
- `rules_python` 0.40.0
- Python 3.13 toolchain
- pip_parse for PyPI integration

## Future Phases

### Phase 2: Advanced Package Support
- Add more packages as they are developed
- Integration with native libraries (open62541, snap7)
- Advanced testing and benchmarking targets

### Phase 3: CI/CD Integration  
- GitHub Actions with Bazel
- Remote build caching
- Multi-platform build matrix

### Phase 4: Performance Optimization
- Build parallelization tuning
- Memory optimization
- Incremental build validation

## Benefits Achieved

1. **Unified Build System**: Single system for all components
2. **Package Isolation**: Clear dependency boundaries
3. **Reproducible Builds**: Hermetic environment
4. **Future-Ready**: Prepared for multi-language expansion
5. **Developer Experience**: Consistent commands via justfile

## Troubleshooting

### Network Issues
If Bazel fails to download:
```bash
# Use legacy build temporarily
just build-legacy
just test-legacy
```

### Build Validation
```bash
# Check BUILD file correctness
python tools/validate_build.py

# Verify Bazel configuration
bazel info
```

### Clean Builds
```bash
# Clean cache
just clean-bazel

# Complete reset
just clean-all
```

## Migration Notes

- **Backward Compatibility**: Legacy build commands preserved
- **Gradual Migration**: Can use both systems during transition
- **Validation**: Build system validated before deployment
- **Documentation**: Complete developer guide provided

This implementation provides the foundation for Bifrost's advanced build system strategy, supporting the project's evolution from Python-only to a complex multi-language industrial IoT framework.