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
        super().__init__(host, port, "modbus")
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
                address, num_registers = self._parse_address(tag.address)
                result = await self.connection.client.read_holding_registers(
                    address=address, count=num_registers, slave=1
                )
                if isinstance(result, ModbusException):
                    raise result
                if hasattr(result, 'isError') and result.isError():
                    continue  # Skip this tag on error
                timestamp = Timestamp(
                    int(asyncio.get_running_loop().time() * 1_000_000_000)
                )
                value = result.registers
                if num_registers > 1:
                    converted_value = [self._convert_to_python(v, tag.data_type) for v in value]
                else:
                    converted_value = self._convert_to_python(value[0], tag.data_type)
                readings[tag] = Reading(
                    tag=tag, value=converted_value, timestamp=timestamp
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
                address, _ = self._parse_address(tag.address)
                if isinstance(value, list):
                    await self.connection.client.write_registers(
                        address=address, values=value, slave=1
                    )
                elif isinstance(value, int):
                    await self.connection.client.write_register(
                        address=address, value=value, slave=1
                    )
                else:
                    raise ValueError("Modbus write value must be an integer or a list of integers")
            except (ModbusException, ValueError, Exception):
                # In a real implementation, we would log this error.
                # Catch all exceptions to handle connection errors gracefully
                pass

    def _parse_address(self, address: str) -> tuple[int, int]:
        """Parse a Modbus address string.

        Args:
            address: The address string to parse.

        Returns:
            A tuple containing the address and the number of registers to read.
        """
        if ":" in address:
            addr, num_registers = address.split(":")
            return int(addr) - 40001, int(num_registers)
        return int(address) - 40001, 1

    async def get_info(self) -> JsonDict:
        """Get information about the Modbus device."""
        return {
            "host": self.connection.host,
            "port": self.connection.port,
            "is_connected": self.connection.is_connected,
        }
