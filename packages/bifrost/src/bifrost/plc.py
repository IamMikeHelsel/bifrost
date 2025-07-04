"""Unified PLC interface."""

from typing import Any

from bifrost_core import BaseConnection, Tag


class PLCConnection:
    """Unified interface for PLC connections."""

    def __init__(self, connection: BaseConnection):
        self._connection = connection

    async def connect(self) -> None:
        """Connect to the PLC."""
        await self._connection.connect()

    async def disconnect(self) -> None:
        """Disconnect from the PLC."""
        await self._connection.disconnect()

    async def read_tags(self, tags: list[Tag]) -> list[Any]:
        """Read multiple tags from the PLC."""
        addresses = [tag.address for tag in tags]
        data_types = [tag.data_type for tag in tags]
        return await self._connection.read_multiple(addresses, data_types)

    async def read_tag(self, tag: Tag) -> Any:
        """Read a single tag from the PLC."""
        return await self._connection.read_single(tag.address, tag.data_type)

    async def write_tag(self, tag: Tag, value: Any) -> None:
        """Write a value to a tag."""
        await self._connection.write_single(tag.address, value, tag.data_type)

    @property
    def is_connected(self) -> bool:
        """Check if PLC is connected."""
        return self._connection.is_connected

    async def __aenter__(self) -> "PLCConnection":
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        await self.disconnect()
