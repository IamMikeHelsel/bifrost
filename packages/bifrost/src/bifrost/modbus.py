"""Modbus implementation for Bifrost."""

import asyncio
from collections.abc import Sequence
from types import TracebackType
from typing import Any

from bifrost_core.base import Reading
from bifrost_core.typing import JsonDict, Tag, Timestamp, Value
from pymodbus.client import AsyncModbusTcpClient
from pymodbus.exceptions import ModbusException
from pymodbus.pdu import ReadHoldingRegistersResponse, WriteSingleRegisterResponse

from .plc import PLC, PLCConnection


class ModbusConnection(PLCConnection):
    """Represents a connection to a Modbus device."""

    def __init__(self, host: str, port: int = 502):
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
                # This is a simplified example. A real implementation would parse the tag
                # to determine the address, function code, and data type.
                address = int(tag)
                result: ReadHoldingRegistersResponse = (
                    await self.connection.client.read_holding_registers(
                        address=address, count=1
                    )
                )
                if isinstance(result, ModbusException):
                    raise result
                timestamp = Timestamp(
                    int(asyncio.get_running_loop().time() * 1_000_000_000)
                )
                readings[tag] = Reading(
                    tag=tag, value=result.registers[0], timestamp=timestamp
                )
            except (ModbusException, ValueError):
                # In a real implementation, we would log this error.
                pass
        return readings

    async def write(self, values: dict[Tag, Value]) -> None:
        """Write one or more values to the Modbus device."""
        for tag, value in values.items():
            try:
                # This is a simplified example. A real implementation would parse the tag
                # to determine the address, function code, and data type.
                address = int(tag)
                # Ensure value is an int for write_register
                if not isinstance(value, int):
                    raise ValueError("Modbus write value must be an integer")
                result: WriteSingleRegisterResponse = (
                    await self.connection.client.write_register(
                        address=address, value=value
                    )
                )
                if isinstance(result, ModbusException):
                    raise result
            except (ModbusException, ValueError):
                # In a real implementation, we would log this error.
                pass

    async def get_info(self) -> JsonDict:
        """Get information about the Modbus device."""
        return {
            "host": self.connection.host,
            "port": self.connection.port,
            "is_connected": self.connection.is_connected,
        }
