# bifrost-core

Core abstractions and base classes for the Bifrost industrial IoT framework.

## Installation

```bash
uv add bifrost-core
```

## Usage

```python
from bifrost_core import BaseConnection, DataPoint

# Use base classes to implement custom protocols
class MyProtocol(BaseConnection):
    async def read_tag(self, address: str) -> DataPoint:
        # Implementation here
        pass
```

## What's Included

- **BaseConnection**: Abstract base class for all protocol implementations
- **BaseProtocol**: Interface for protocol-specific operations
- **DataPoint**: Unified data model for industrial data
- **EventBus**: Event system for connection lifecycle management
- **ConnectionPool**: Connection pooling and management
- **Type System**: Common type definitions and enums

## Dependencies

- Python 3.13+
- Pydantic 2.5+ (data validation)
- typing-extensions 4.8+ (enhanced type hints)

## License

MIT License - see LICENSE file for details.
