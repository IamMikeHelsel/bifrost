"""Core abstractions for Bifrost."""

from abc import ABC, abstractmethod
from collections.abc import Sequence
from enum import Enum
from types import TracebackType
from typing import Generic

from pydantic import BaseModel, Field, model_validator

from .typing import JsonDict, Tag, Timestamp, Value


class ConnectionState(Enum):
    """Represents the state of a connection."""

    DISCONNECTED = "disconnected"
    CONNECTING = "connecting"
    CONNECTED = "connected"
    DISCONNECTING = "disconnecting"


class DeviceInfo(BaseModel):
    """Represents information about a discovered device."""

    device_id: str = Field(..., description="Unique identifier for the device.")
    protocol: str = Field(
        ..., description="The communication protocol used (e.g., 'modbus.tcp')."
    )
    host: str = Field(
        ..., description="The IP address or hostname of the device."
    )
    port: int = Field(..., description="The port number of the device.")
    name: str | None = Field(
        None, description="Human-readable name of the device."
    )
    manufacturer: str | None = Field(
        None, description="Manufacturer of the device."
    )
    model: str | None = Field(None, description="Model of the device.")
    description: str | None = Field(
        None, description="A brief description of the device."
    )
    # Add more fields as needed for specific protocols or device types

    @model_validator(mode="after")
    def set_default_name(self) -> "DeviceInfo":
        """Set default name if not provided."""
        if self.name is None:
            self.name = self.device_id
        return self


class Reading(BaseModel, Generic[Value]):
    """Represents a single data point read from a device."""

    tag: Tag = Field(
        ...,
        description="The unique identifier for the data point (e.g., a PLC tag).",
    )
    value: Value = Field(..., description="The value read from the device.")
    timestamp: Timestamp = Field(
        ..., description="The nanosecond timestamp of when the value was read."
    )


class BaseConnection(ABC):
    """Abstract base class for a connection to a device or service."""

    @abstractmethod
    async def __aenter__(self) -> "BaseConnection":
        """Enter the async context manager."""
        raise NotImplementedError

    @abstractmethod
    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: TracebackType | None,
    ) -> None:
        """Exit the async context manager."""
        raise NotImplementedError

    @property
    @abstractmethod
    def is_connected(self) -> bool:
        """Return True if the connection is active."""
        raise NotImplementedError


class BaseDevice(ABC, Generic[Value]):
    """Abstract base class for a device."""

    def __init__(self, connection: BaseConnection):
        """Initialize the device with a connection.

        Args:
            connection: The connection to use for communicating with the device.
        """
        self.connection = connection

    @abstractmethod
    async def read(self, tags: Sequence[Tag]) -> dict[Tag, Reading[Value]]:
        """Read one or more values from the device."""
        raise NotImplementedError

    @abstractmethod
    async def write(self, values: dict[Tag, Value]) -> None:
        """Write one or more values to the device."""
        raise NotImplementedError

    @abstractmethod
    async def get_info(self) -> JsonDict:
        """Get information about the device."""
        raise NotImplementedError
