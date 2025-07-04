"""Core base classes for Bifrost industrial IoT framework.

This module provides the fundamental abstractions that all protocol
implementations build upon.
"""

from abc import ABC, abstractmethod
from datetime import datetime
from enum import Enum
from typing import Any

from pydantic import BaseModel, ConfigDict


class ConnectionState(Enum):
    """Connection state enumeration."""

    DISCONNECTED = "disconnected"
    CONNECTING = "connecting"
    CONNECTED = "connected"
    RECONNECTING = "reconnecting"
    FAILED = "failed"


class DataType(Enum):
    """Supported data types for industrial data."""

    BOOL = "bool"
    INT16 = "int16"
    UINT16 = "uint16"
    INT32 = "int32"
    UINT32 = "uint32"
    INT64 = "int64"
    UINT64 = "uint64"
    FLOAT32 = "float32"
    FLOAT64 = "float64"
    STRING = "string"
    BYTES = "bytes"


class ProtocolType(Enum):
    """Supported protocol types."""

    MODBUS_TCP = "modbus_tcp"
    MODBUS_RTU = "modbus_rtu"
    OPCUA = "opcua"
    ETHERNET_IP = "ethernet_ip"
    S7 = "s7"
    DNP3 = "dnp3"


class DataPoint(BaseModel):
    """Represents a single data point from an industrial device."""

    model_config = ConfigDict(frozen=True)

    address: str
    value: Any
    data_type: DataType
    timestamp: datetime
    quality: str | None = None
    device_id: str | None = None

    def __str__(self) -> str:
        return f"DataPoint({self.address}={self.value}, {self.data_type.value})"


class ConnectionError(Exception):
    """Base exception for connection-related errors."""

    pass


class ProtocolError(Exception):
    """Base exception for protocol-related errors."""

    pass


class TimeoutError(ConnectionError):
    """Raised when an operation times out."""

    pass


class BaseConnection(ABC):
    """Abstract base class for all industrial device connections.

    This class provides the common interface and lifecycle management
    for connections to industrial equipment.
    """

    def __init__(
        self,
        host: str,
        port: int | None = None,
        timeout: float = 5.0,
        retry_attempts: int = 3,
        retry_delay: float = 1.0,
        **kwargs: Any,
    ) -> None:
        self.host = host
        self.port = port
        self.timeout = timeout
        self.retry_attempts = retry_attempts
        self.retry_delay = retry_delay
        self._state = ConnectionState.DISCONNECTED
        self._connection_id = f"{host}:{port}" if port else host

    @property
    def state(self) -> ConnectionState:
        """Current connection state."""
        return self._state

    @property
    def is_connected(self) -> bool:
        """Check if connection is active."""
        return self._state == ConnectionState.CONNECTED

    @property
    def connection_id(self) -> str:
        """Unique identifier for this connection."""
        return self._connection_id

    @abstractmethod
    async def connect(self) -> None:
        """Establish connection to the device."""
        pass

    @abstractmethod
    async def disconnect(self) -> None:
        """Close the connection to the device."""
        pass

    @abstractmethod
    async def read_raw(self, address: str, count: int = 1) -> list[Any]:
        """Read raw data from the device."""
        pass

    @abstractmethod
    async def write_raw(self, address: str, values: list[Any]) -> None:
        """Write raw data to the device."""
        pass

    async def read_single(self, address: str, data_type: DataType) -> DataPoint:
        """Read a single data point from the device."""
        raw_values = await self.read_raw(address, count=1)
        return DataPoint(
            address=address,
            value=self._convert_value(raw_values[0], data_type),
            data_type=data_type,
            timestamp=datetime.now(),
            device_id=self.connection_id,
        )

    async def read_multiple(
        self, addresses: list[str], data_types: list[DataType]
    ) -> list[DataPoint]:
        """Read multiple data points from the device."""
        if len(addresses) != len(data_types):
            raise ValueError("Addresses and data_types must have same length")

        # For now, read individually (protocols can optimize this)
        results = []
        for address, data_type in zip(addresses, data_types, strict=False):
            try:
                data_point = await self.read_single(address, data_type)
                results.append(data_point)
            except Exception as e:
                # Create error data point
                results.append(
                    DataPoint(
                        address=address,
                        value=None,
                        data_type=data_type,
                        timestamp=datetime.now(),
                        quality=f"error: {str(e)}",
                        device_id=self.connection_id,
                    )
                )

        return results

    async def write_single(self, address: str, value: Any, data_type: DataType) -> None:
        """Write a single value to the device."""
        converted_value = self._convert_value(value, data_type)
        await self.write_raw(address, [converted_value])

    def _convert_value(self, raw_value: Any, data_type: DataType) -> Any:
        """Convert raw value to the specified data type."""
        if data_type == DataType.BOOL:
            return bool(raw_value)
        elif data_type == DataType.INT16:
            return int(raw_value)
        elif data_type == DataType.UINT16:
            return int(raw_value) & 0xFFFF
        elif data_type == DataType.INT32:
            return int(raw_value)
        elif data_type == DataType.UINT32:
            return int(raw_value) & 0xFFFFFFFF
        elif data_type == DataType.FLOAT32:
            return float(raw_value)
        elif data_type == DataType.FLOAT64:
            return float(raw_value)
        elif data_type == DataType.STRING:
            return str(raw_value)
        else:
            return raw_value

    async def health_check(self) -> bool:
        """Perform a health check on the connection."""
        try:
            if not self.is_connected:
                return False
            # Try a simple read operation
            await self.read_raw("0", count=1)
            return True
        except Exception:
            return False

    async def __aenter__(self) -> "BaseConnection":
        """Async context manager entry."""
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Async context manager exit."""
        await self.disconnect()


class BaseProtocol(ABC):
    """Abstract base class for protocol implementations."""

    @abstractmethod
    def get_protocol_type(self) -> ProtocolType:
        """Return the protocol type this implementation handles."""
        pass

    @abstractmethod
    async def create_connection(
        self, connection_string: str, **kwargs: Any
    ) -> BaseConnection:
        """Create a new connection from a connection string."""
        pass

    @abstractmethod
    def parse_connection_string(self, connection_string: str) -> dict[str, Any]:
        """Parse a connection string into connection parameters."""
        pass
