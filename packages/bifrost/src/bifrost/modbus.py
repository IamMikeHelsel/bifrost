"""Modbus implementation for Bifrost.

This module provides Modbus TCP client functionality with async support,
including connection management and read/write operations for holding registers.
"""

import asyncio
from collections.abc import Sequence
from types import TracebackType
from typing import Any

from pymodbus.client import AsyncModbusTcpClient
from pymodbus.exceptions import ModbusException
from pymodbus.pdu.register_message import (
    ReadHoldingRegistersResponse,
    WriteSingleRegisterResponse,
)

from bifrost_core.base import Reading
from bifrost_core.typing import JsonDict, Tag, Timestamp, Value

from .plc import PLC, PLCConnection


class ModbusConnection(PLCConnection):
    """Represents a connection to a Modbus device.
    
    Manages the async Modbus TCP client connection lifecycle and provides
    context manager support for automatic connection management.
    
    Attributes:
        client: The underlying pymodbus async TCP client.
    """

    def __init__(self, host: str, port: int = 502):
        """Initialize Modbus TCP connection.
        
        Args:
            host: IP address or hostname of the Modbus device.
            port: TCP port number (default: 502).
        """
        super().__init__(host, port)
        self.client = AsyncModbusTcpClient(host=host, port=port)

    async def __aenter__(self) -> "ModbusConnection":
        connected = await self.client.connect()
        self._is_connected = connected
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: TracebackType | None,
    ) -> None:
        if hasattr(self.client, 'close'):
            if asyncio.iscoroutinefunction(self.client.close):
                await self.client.close()
            else:
                self.client.close()
        self._is_connected = False


class ModbusDevice(PLC[Any]):
    """Represents a Modbus device."""

    def __init__(self, connection: ModbusConnection):
        super().__init__(connection)
        self.connection: ModbusConnection  # For type hinting

    async def read(self, tags: Sequence[Tag]) -> dict[Tag, Reading[Value]]:
        """Read one or more values from the Modbus device."""
        readings: dict[Tag, Reading[Value]] = {}
        for tag in tags:
            try:
                # Parse Modbus address from tag.address (e.g., "40001" -> 0)
                # Modbus holding registers start at 40001, but use 0-based addressing
                address = int(tag.address) - 40001
                result = await self.connection.client.read_holding_registers(
                    address=address, count=1, slave=1
                )
                if isinstance(result, ModbusException):
                    raise result
                if hasattr(result, 'isError') and result.isError():
                    continue  # Skip this tag on error
                timestamp = Timestamp(
                    int(asyncio.get_running_loop().time() * 1_000_000_000)
                )
                readings[tag] = Reading(
                    tag=tag, value=result.registers[0], timestamp=timestamp
                )
            except (ModbusException, ValueError, Exception):
                # In a real implementation, we would log this error.
                # Catch all exceptions to handle connection errors gracefully
                pass
        return readings

    async def write(self, values: dict[Tag, Value]) -> None:
        """Write one or more values to the Modbus device."""
        for tag, value in values.items():
            try:
                # Parse Modbus address from tag.address (e.g., "40001" -> 0)
                # Modbus holding registers start at 40001, but use 0-based addressing
                address = int(tag.address) - 40001
                # Ensure value is an int for write_register
                if not isinstance(value, int):
                    raise ValueError("Modbus write value must be an integer")
                result: WriteSingleRegisterResponse = (
                    await self.connection.client.write_register(
                        address=address, value=value, slave=1
                    )
                )
                if isinstance(result, ModbusException):
                    raise result
            except (ModbusException, ValueError, Exception):
                # In a real implementation, we would log this error.
                # Catch all exceptions to handle connection errors gracefully
                pass

    async def get_info(self) -> JsonDict:
        """Get information about the Modbus device."""
        return {
            "host": self.connection.host,
            "port": self.connection.port,
            "is_connected": self.connection.is_connected,
        }
