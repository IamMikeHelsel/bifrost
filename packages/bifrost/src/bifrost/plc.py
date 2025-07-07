"""Base classes for Programmable Logic Controllers (PLCs)."""

from abc import ABC, abstractmethod
from collections.abc import Sequence
from types import TracebackType
from typing import Any

from bifrost_core.base import BaseConnection, BaseDevice, Reading
from bifrost_core.typing import DataType, Tag, Value


class PLCConnection(BaseConnection):
    """Represents a connection to a PLC."""

    def __init__(self, host: str, port: int, protocol: str):
        """Initializes the PLCConnection."""
        self.host = host
        self.port = port
        self.protocol = protocol
        self._is_connected = False

    async def __aenter__(self) -> "PLCConnection":
        """Enters the async context manager for the PLC connection."""
        # In a real implementation, this would establish a network connection.
        self._is_connected = True
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: TracebackType | None,
    ) -> None:
        # In a real implementation, this would close the network connection.
        self._is_connected = False

    @property
    def is_connected(self) -> bool:
        return self._is_connected


class PLC(BaseDevice[Value], ABC):
    """Represents a generic PLC."""

    def __init__(self, connection: PLCConnection):
        super().__init__(connection)

    @abstractmethod
    async def read(self, tags: Sequence[Tag]) -> dict[Tag, Reading[Value]]:
        """Read one or more values from the PLC."""
        raise NotImplementedError

    @abstractmethod
    async def write(self, values: dict[Tag, Value]) -> None:
        """Write one or more values to the PLC."""
        raise NotImplementedError

    def _convert_to_python(self, value: Any, data_type: DataType) -> Any:
        """Convert a value from the PLC to a Python type."""
        if data_type in {DataType.INT16, DataType.UINT16, DataType.INT32, DataType.UINT32}:
            return int(value)
        if data_type in {DataType.FLOAT32, DataType.FLOAT64}:
            return float(value)
        if data_type == DataType.BOOLEAN:
            return bool(value)
        if data_type == DataType.STRING:
            return str(value)
        if data_type == DataType.BYTE:
            return bytes(value)
        return value
