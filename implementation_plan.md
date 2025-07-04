# Bifrost Implementation Plan

## Executive Summary

This document outlines the complete implementation strategy for the Bifrost industrial IoT framework, incorporating modern tooling recommendations and a hybrid modular packaging approach. The plan addresses the polyglot nature of the project (Python + Rust + C/C++) while ensuring optimal developer experience and user adoption.

## 1. Architecture Overview

### 1.1 Final Package Structure

```
bifrost/
├── packages/
│   ├── bifrost-core/           # Core abstractions (Python only)
│   ├── bifrost/               # Main package (Python + Rust)
│   ├── bifrost-opcua/         # OPC UA implementation (Python + C)
│   ├── bifrost-analytics/     # Edge analytics (Python + Rust)
│   ├── bifrost-cloud/         # Cloud connectors (Python only)
│   ├── bifrost-cli/           # CLI interface (Python only)
│   └── bifrost-all/           # Meta-package
├── rust/
│   ├── modbus-engine/         # Modbus implementation
│   ├── analytics-core/        # Time-series processing
│   └── shared/               # Common utilities
├── native/
│   └── opcua-wrapper/        # OPC UA C library bindings
├── docs/
├── examples/
├── tests/
├── .github/
├── justfile
└── README.md
```

### 1.2 Technology Stack

**Package Management:** `uv` (10-100x faster than pip)
**Linting/Formatting:** `ruff` (10-100x faster than black/flake8)
**Testing:** `pytest` + `pytest-cov` + `pytest-asyncio`
**Type Checking:** `mypy`
**Task Runner:** `just` (cross-platform make alternative)
**Build System:** `maturin` (Python + Rust integration)
**CI/CD:** GitHub Actions with modern tooling

## 2. Phase 1: Foundation Setup (Weeks 1-4)

### 2.1 Repository Structure Setup

**Week 1: Core Infrastructure**
```bash
# Initialize repository structure
mkdir -p packages/{bifrost-core,bifrost,bifrost-opcua,bifrost-analytics,bifrost-cloud,bifrost-cli,bifrost-all}
mkdir -p rust/{modbus-engine,analytics-core,shared}
mkdir -p native/opcua-wrapper
mkdir -p {docs,examples,tests}

# Install modern tooling
curl -LsSf https://astral.sh/uv/install.sh | sh
cargo install just
```

**Week 2: Build System Configuration**

Create root `pyproject.toml`:
```toml
[build-system]
requires = ["maturin>=1.0,<2.0"]
build-backend = "maturin"

[project]
name = "bifrost-workspace"
version = "0.1.0"
description = "Industrial IoT framework for Python"
authors = [{name = "Bifrost Team", email = "team@bifrost.dev"}]
license = {text = "MIT"}
readme = "README.md"
requires-python = ">=3.8"

[project.urls]
homepage = "https://github.com/bifrost-dev/bifrost"
repository = "https://github.com/bifrost-dev/bifrost"
documentation = "https://docs.bifrost.dev"

[tool.ruff]
target-version = "py38"
line-length = 88
select = ["E", "W", "F", "I", "B", "C4", "UP", "RUF"]

[tool.ruff.format]
quote-style = "double"
indent-style = "space"

[tool.pytest.ini_options]
minversion = "6.0"
addopts = [
    "--cov=bifrost",
    "--cov-report=term-missing",
    "--cov-report=html",
    "--asyncio-mode=auto",
]
asyncio_mode = "auto"
testpaths = ["tests"]

[tool.mypy]
python_version = "3.8"
strict = true
warn_unused_configs = true
```

**Week 3: Core Package Development**

`packages/bifrost-core/pyproject.toml`:
```toml
[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "bifrost-core"
dynamic = ["version"]
description = "Core abstractions for Bifrost industrial IoT framework"
authors = [{name = "Bifrost Team", email = "team@bifrost.dev"}]
license = {text = "MIT"}
readme = "README.md"
requires-python = ">=3.8"
dependencies = [
    "pydantic>=2.0",
    "typing-extensions>=4.0",
    "asyncio-mqtt>=0.16",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.0",
    "pytest-cov>=4.0",
    "pytest-asyncio>=0.21",
    "mypy>=1.0",
    "ruff>=0.1.0",
]

[tool.hatch.version]
path = "bifrost_core/__init__.py"
```

**Week 4: Task Runner Setup**

Root `justfile`:
```bash
# justfile
default:
    just --list

# Development setup
setup:
    uv pip install -e "packages/bifrost-core[dev]"
    uv pip install -e "packages/bifrost[dev]"
    pre-commit install

# Code quality
format:
    ruff format .
    ruff check --fix .

lint:
    ruff check .
    mypy packages/

test:
    pytest tests/

# Rust tasks
rust-format:
    find rust -name "*.rs" -path "*/src/*" | xargs rustfmt

rust-lint:
    cd rust/modbus-engine && cargo clippy -- -D warnings
    cd rust/analytics-core && cargo clippy -- -D warnings

rust-test:
    cd rust/modbus-engine && cargo test
    cd rust/analytics-core && cargo test

# Multi-language tasks
format-all:
    just format
    just rust-format

lint-all:
    just lint
    just rust-lint

test-all:
    just test
    just rust-test

# Build tasks
build-core:
    cd packages/bifrost-core && uv pip install -e .

build-main:
    cd packages/bifrost && maturin develop

build-all:
    just build-core
    just build-main

# Package distribution
package COMPONENT:
    cd packages/{{COMPONENT}} && python -m build

package-all:
    just package bifrost-core
    just package bifrost
    just package bifrost-opcua
    just package bifrost-analytics
    just package bifrost-cloud
    just package bifrost-cli

# Clean up
clean:
    find . -name "*.pyc" -delete
    find . -name "__pycache__" -delete
    find . -type d -name "*.egg-info" -exec rm -rf {} +
    find . -name "*.so" -delete
    rm -rf dist/ build/ target/
```

### 2.2 Core Abstractions Implementation

**packages/bifrost-core/bifrost_core/base.py**:
```python
from abc import ABC, abstractmethod
from typing import Any, Dict, List, Optional, Union, AsyncIterator
from pydantic import BaseModel, Field
from enum import Enum
import asyncio

class DataType(str, Enum):
    """Standard industrial data types"""
    BOOL = "bool"
    INT16 = "int16"
    INT32 = "int32"
    FLOAT32 = "float32"
    FLOAT64 = "float64"
    STRING = "string"

class Tag(BaseModel):
    """Represents a data point in industrial equipment"""
    name: str
    address: Union[str, int]
    data_type: DataType
    description: Optional[str] = None
    unit: Optional[str] = None
    
class DataPoint(BaseModel):
    """A timestamped data reading"""
    tag: Tag
    value: Any
    timestamp: float = Field(default_factory=lambda: asyncio.get_event_loop().time())
    quality: str = "good"

class ConnectionConfig(BaseModel):
    """Base configuration for device connections"""
    host: str
    port: int
    timeout: float = 5.0
    retries: int = 3

class BaseConnection(ABC):
    """Abstract base class for device connections"""
    
    def __init__(self, config: ConnectionConfig):
        self.config = config
        self._connected = False
    
    async def __aenter__(self):
        await self.connect()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.disconnect()
    
    @abstractmethod
    async def connect(self) -> None:
        """Establish connection to device"""
        pass
    
    @abstractmethod
    async def disconnect(self) -> None:
        """Close connection to device"""
        pass
    
    @abstractmethod
    async def read_tags(self, tags: List[Tag]) -> List[DataPoint]:
        """Read multiple tags from device"""
        pass
    
    @abstractmethod
    async def write_tags(self, data_points: List[DataPoint]) -> None:
        """Write multiple data points to device"""
        pass
    
    @property
    def connected(self) -> bool:
        return self._connected

class BaseProtocol(ABC):
    """Abstract base class for protocol implementations"""
    
    @abstractmethod
    def create_connection(self, config: ConnectionConfig) -> BaseConnection:
        """Create a connection instance for this protocol"""
        pass
```

## 3. Phase 2: Modbus Implementation (Weeks 5-8)

### 3.1 Rust Modbus Engine

**rust/modbus-engine/Cargo.toml**:
```toml
[package]
name = "modbus-engine"
version = "0.1.0"
edition = "2021"

[dependencies]
pyo3 = { version = "0.20", features = ["extension-module"] }
tokio = { version = "1.0", features = ["full"] }
serde = { version = "1.0", features = ["derive"] }
modbus = "0.5"
thiserror = "1.0"

[lib]
name = "modbus_engine"
crate-type = ["cdylib"]

[lints.rust]
unsafe_code = "forbid"

[lints.clippy]
enum_glob_use = "deny"
pedantic = "warn"
unwrap_used = "deny"
```

**rust/modbus-engine/src/lib.rs**:
```rust
use pyo3::prelude::*;
use std::collections::HashMap;
use tokio::net::TcpStream;
use modbus::tcp::Ctx;

#[pyclass]
struct ModbusConnection {
    ctx: Option<Ctx>,
    host: String,
    port: u16,
}

#[pymethods]
impl ModbusConnection {
    #[new]
    fn new(host: String, port: u16) -> Self {
        Self {
            ctx: None,
            host,
            port,
        }
    }
    
    #[pyo3(asyncio)]
    async fn connect(&mut self) -> PyResult<()> {
        let stream = TcpStream::connect(format!("{}:{}", self.host, self.port))
            .await
            .map_err(|e| PyErr::new::<pyo3::exceptions::PyConnectionError, _>(e.to_string()))?;
        
        self.ctx = Some(Ctx::new_tcp(stream));
        Ok(())
    }
    
    #[pyo3(asyncio)]
    async fn read_holding_registers(&mut self, address: u16, count: u16) -> PyResult<Vec<u16>> {
        let ctx = self.ctx.as_mut()
            .ok_or_else(|| PyErr::new::<pyo3::exceptions::PyRuntimeError, _>("Not connected"))?;
        
        let values = ctx.read_holding_registers(address, count)
            .await
            .map_err(|e| PyErr::new::<pyo3::exceptions::PyIOError, _>(e.to_string()))?;
        
        Ok(values)
    }
}

#[pymodule]
fn modbus_engine(_py: Python, m: &PyModule) -> PyResult<()> {
    m.add_class::<ModbusConnection>()?;
    Ok(())
}
```

### 3.2 Python Modbus Implementation

**packages/bifrost/bifrost/modbus.py**:
```python
from typing import List, Optional
from bifrost_core.base import BaseConnection, BaseProtocol, ConnectionConfig, Tag, DataPoint, DataType
from .modbus_engine import ModbusConnection as RustModbusConnection
import asyncio

class ModbusConfig(ConnectionConfig):
    """Modbus-specific configuration"""
    slave_id: int = 1
    
class ModbusConnection(BaseConnection):
    """High-performance Modbus connection using Rust backend"""
    
    def __init__(self, config: ModbusConfig):
        super().__init__(config)
        self.config = config
        self._rust_connection = RustModbusConnection(config.host, config.port)
    
    async def connect(self) -> None:
        """Connect to Modbus device"""
        await self._rust_connection.connect()
        self._connected = True
    
    async def disconnect(self) -> None:
        """Disconnect from Modbus device"""
        # Rust connection handles cleanup automatically
        self._connected = False
    
    async def read_tags(self, tags: List[Tag]) -> List[DataPoint]:
        """Read multiple tags efficiently"""
        if not self._connected:
            raise RuntimeError("Not connected to device")
        
        # Group tags by address range for bulk reads
        results = []
        for tag in tags:
            if isinstance(tag.address, int):
                # Read single register for now - optimize for bulk reads later
                raw_values = await self._rust_connection.read_holding_registers(
                    tag.address, 1
                )
                value = self._convert_value(raw_values[0], tag.data_type)
                results.append(DataPoint(tag=tag, value=value))
        
        return results
    
    async def write_tags(self, data_points: List[DataPoint]) -> None:
        """Write multiple data points"""
        # Implementation for writing tags
        pass
    
    def _convert_value(self, raw_value: int, data_type: DataType) -> any:
        """Convert raw Modbus value to typed value"""
        if data_type == DataType.INT16:
            return raw_value if raw_value < 32768 else raw_value - 65536
        elif data_type == DataType.FLOAT32:
            # Handle float conversion (requires 2 registers)
            pass
        elif data_type == DataType.BOOL:
            return bool(raw_value)
        return raw_value

class ModbusProtocol(BaseProtocol):
    """Modbus protocol implementation"""
    
    def create_connection(self, config: ConnectionConfig) -> BaseConnection:
        if not isinstance(config, ModbusConfig):
            # Convert generic config to Modbus config
            config = ModbusConfig(**config.dict())
        return ModbusConnection(config)
```

## 4. Phase 3: Package Distribution (Weeks 9-12)

### 4.1 Individual Package Configurations

**packages/bifrost/pyproject.toml**:
```toml
[build-system]
requires = ["maturin>=1.0,<2.0"]
build-backend = "maturin"

[project]
name = "bifrost"
dynamic = ["version"]
description = "Industrial IoT framework for Python"
authors = [{name = "Bifrost Team", email = "team@bifrost.dev"}]
license = {text = "MIT"}
readme = "README.md"
requires-python = ">=3.8"
dependencies = [
    "bifrost-core>=0.1.0",
    "pymodbus>=3.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.0",
    "pytest-cov>=4.0",
    "pytest-asyncio>=0.21",
    "mypy>=1.0",
    "ruff>=0.1.0",
]

[tool.maturin]
features = ["pyo3/extension-module"]
bindings = "pyo3"
```

### 4.2 Meta-Package Configuration

**packages/bifrost-all/pyproject.toml**:
```toml
[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "bifrost-all"
dynamic = ["version"]
description = "Complete Bifrost industrial IoT framework"
authors = [{name = "Bifrost Team", email = "team@bifrost.dev"}]
license = {text = "MIT"}
readme = "README.md"
requires-python = ">=3.8"
dependencies = [
    "bifrost-core>=0.1.0",
    "bifrost>=0.1.0",
    "bifrost-opcua>=0.1.0",
    "bifrost-analytics>=0.1.0",
    "bifrost-cloud>=0.1.0",
    "bifrost-cli>=0.1.0",
]

[tool.hatch.version]
path = "bifrost_all/__init__.py"
```

### 4.3 CI/CD Pipeline

**.github/workflows/ci.yml**:
```yaml
name: CI/CD

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  lint-and-format:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Install uv
      uses: astral-sh/setup-uv@v1
    
    - name: Set up Python
      run: uv python install 3.11
    
    - name: Install dependencies
      run: uv pip install -e "packages/bifrost-core[dev]"
    
    - name: Lint and format check
      run: |
        uv run ruff check .
        uv run ruff format --check .
        uv run mypy packages/
    
    - name: Install Rust
      uses: dtolnay/rust-toolchain@stable
      with:
        components: rustfmt, clippy
    
    - name: Rust lint and format
      run: |
        cd rust/modbus-engine && cargo fmt --check
        cd rust/modbus-engine && cargo clippy -- -D warnings

  test:
    needs: lint-and-format
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
        python-version: ["3.8", "3.9", "3.10", "3.11", "3.12"]
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Install uv
      uses: astral-sh/setup-uv@v1
    
    - name: Set up Python ${{ matrix.python-version }}
      run: uv python install ${{ matrix.python-version }}
    
    - name: Install Rust
      uses: dtolnay/rust-toolchain@stable
    
    - name: Install dependencies
      run: |
        uv pip install -e "packages/bifrost-core[dev]"
        uv pip install -e "packages/bifrost[dev]"
    
    - name: Test with pytest
      run: uv run pytest tests/ --cov=bifrost --cov-report=xml
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.xml

  build-wheels:
    needs: test
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Install uv
      uses: astral-sh/setup-uv@v1
    
    - name: Install Rust
      uses: dtolnay/rust-toolchain@stable
    
    - name: Build wheels
      run: |
        uv pip install maturin
        cd packages/bifrost && maturin build --release
    
    - name: Upload wheels
      uses: actions/upload-artifact@v3
      with:
        name: wheels-${{ matrix.os }}
        path: packages/bifrost/dist/

  publish:
    needs: build-wheels
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Install uv
      uses: astral-sh/setup-uv@v1
    
    - name: Download all wheels
      uses: actions/download-artifact@v3
      with:
        path: dist/
    
    - name: Publish to PyPI
      run: |
        uv pip install twine
        twine upload dist/*/*.whl
      env:
        TWINE_USERNAME: __token__
        TWINE_PASSWORD: ${{ secrets.PYPI_TOKEN }}
```

## 5. User Experience Optimization

### 5.1 Installation Patterns

**Quick Start Documentation**:
```markdown
# Bifrost Installation Guide

## For Edge Deployments (Minimal)
```bash
# Install core with Modbus support
pip install bifrost

# Example usage
python -c "
from bifrost import connect
async def main():
    async with connect('modbus://192.168.1.100') as plc:
        data = await plc.read_tags(['temperature', 'pressure'])
        print(data)
"
```

## For Full Development Environment
```bash
# Install everything
pip install bifrost-all

# Or install selectively
pip install bifrost[opcua,analytics,cloud,cli]
```

## For CLI-only Usage
```bash
# Install standalone CLI
pip install bifrost-cli

# Use immediately
bifrost discover
bifrost connect modbus://192.168.1.100
```
```

### 5.2 Smart Import System

**packages/bifrost/__init__.py**:
```python
"""
Bifrost Industrial IoT Framework

Provides unified access to industrial protocols with high performance.
"""

from bifrost_core.base import Tag, DataPoint, DataType
from .protocols import connect, list_protocols

# Smart imports with helpful error messages
def __getattr__(name):
    """Handle dynamic imports with helpful error messages"""
    
    if name == 'opcua':
        try:
            from . import opcua
            return opcua
        except ImportError:
            raise ImportError(
                "OPC UA support not installed. "
                "Install with: pip install bifrost[opcua]"
            )
    
    elif name == 'analytics':
        try:
            from . import analytics
            return analytics
        except ImportError:
            raise ImportError(
                "Analytics support not installed. "
                "Install with: pip install bifrost[analytics]"
            )
    
    elif name == 'cloud':
        try:
            from . import cloud
            return cloud
        except ImportError:
            raise ImportError(
                "Cloud support not installed. "
                "Install with: pip install bifrost[cloud]"
            )
    
    raise AttributeError(f"module 'bifrost' has no attribute '{name}'")

# Always available
__all__ = ['connect', 'list_protocols', 'Tag', 'DataPoint', 'DataType']
```

## 6. Timeline and Milestones

### 6.1 Development Timeline

**Phase 1: Foundation (Weeks 1-4)**
- ✅ Repository structure and tooling setup
- ✅ Core abstractions implementation
- ✅ Build system configuration
- ✅ CI/CD pipeline setup

**Phase 2: Modbus MVP (Weeks 5-8)**
- 🔄 Rust Modbus engine implementation
- 🔄 Python Modbus wrapper
- 🔄 Performance benchmarking
- 🔄 Documentation and examples

**Phase 3: Package Distribution (Weeks 9-12)**
- 📅 Individual package configurations
- 📅 Meta-package setup
- 📅 PyPI publishing pipeline
- 📅 User experience optimization

**Phase 4: Additional Protocols (Weeks 13-16)**
- 📅 OPC UA implementation
- 📅 Edge analytics engine
- 📅 Cloud connectors
- 📅 CLI interface

### 6.2 Success Metrics

**Technical Metrics:**
- Build time: < 5 minutes for all platforms
- Package size: < 30MB for core package
- Installation success rate: > 95%
- Performance: 10x faster than pure Python alternatives

**User Experience Metrics:**
- Time to first success: < 10 minutes
- Documentation clarity: Community feedback
- Support ticket volume: < 5 per week after v0.5

## 7. Risk Mitigation

### 7.1 Technical Risks

**Rust/Python Integration Complexity**
- Mitigation: Start with simple Modbus implementation
- Fallback: Pure Python implementation for unsupported platforms

**Performance Targets**
- Mitigation: Continuous benchmarking in CI
- Validation: Performance regression tests

**Dependency Management**
- Mitigation: Lock files and compatibility matrices
- Automation: Automated dependency updates with testing

### 7.2 User Adoption Risks

**Complex Installation**
- Mitigation: Pre-built wheels for all platforms
- Documentation: Clear installation guides per use case

**Learning Curve**
- Mitigation: Progressive disclosure in API design
- Support: Comprehensive examples and tutorials

## 8. Conclusion

This implementation plan provides a comprehensive roadmap for building Bifrost with modern tooling and packaging strategies. The hybrid modular approach balances flexibility with usability, while the modern toolchain (uv, ruff, just, maturin) provides significant performance improvements and better developer experience.

Key advantages of this approach:
- **Performance**: 10-100x faster tooling improves development velocity
- **Flexibility**: Users can install only what they need
- **Maintainability**: Clear separation of concerns and automated tooling
- **Scalability**: Architecture supports growth and additional protocols
- **User Experience**: Smart imports and helpful error messages

The plan addresses the polyglot nature of the project while maintaining a cohesive Python-centric user experience, making industrial protocols as accessible as modern web APIs.