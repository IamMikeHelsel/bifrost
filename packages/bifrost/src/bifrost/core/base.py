"""Base classes for connections and protocols."""

import asyncio
from abc import ABC, abstractmethod
from typing import Any, Protocol, runtime_checkable


class BaseConnection(ABC):
    """Base class for all protocol connections using async context manager pattern."""

    def __init__(self, host: str, port: int, **kwargs: Any) -> None:
        self.host = host
        self.port = port
        self.config = kwargs
        self._connected = False
        self._lock = asyncio.Lock()

    @abstractmethod
    async def connect(self) -> None:
        """Establish connection to the target device."""

    @abstractmethod
    async def disconnect(self) -> None:
        """Close connection to the target device."""

    @abstractmethod
    async def is_connected(self) -> bool:
        """Check if connection is active."""
        return self._connected

    async def __aenter__(self) -> "BaseConnection":
        """Async context manager entry."""
        await self.connect()
        return self

    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        """Async context manager exit."""
        await self.disconnect()


@runtime_checkable
class BaseProtocol(Protocol):
    """Protocol interface for industrial communication protocols."""

    name: str
    version: str

    async def read(self, address: Any, count: int = 1) -> Any:
        """Read data from the specified address."""
        ...

    async def write(self, address: Any, value: Any) -> None:
        """Write data to the specified address."""
        ...

    async def validate_address(self, address: Any) -> bool:
        """Validate if the address format is correct for this protocol."""
        ...
