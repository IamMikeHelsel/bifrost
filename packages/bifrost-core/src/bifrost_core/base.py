"""Core abstractions for Bifrost."""

from abc import ABC, abstractmethod
from typing import Generic, Sequence

from pydantic import BaseModel, Field

from .typing import JsonDict, Tag, Timestamp, Value


class Reading(BaseModel, Generic[Value]):
    """Represents a single data point read from a device."""

    tag: Tag = Field(..., description="The unique identifier for the data point (e.g., a PLC tag).")
    value: Value = Field(..., description="The value read from the device.")
    timestamp: Timestamp = Field(..., description="The nanosecond timestamp of when the value was read.")


class BaseConnection(ABC):
    """Abstract base class for a connection to a device or service."""

    @abstractmethod
    async def __aenter__(self) -> "BaseConnection":
        """Enter the async context manager."""
        raise NotImplementedError

    @abstractmethod
    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
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
