"""Modbus implementation for Bifrost."""

import asyncio
from typing import Any, Sequence

from bifrost_core.base import Reading
from bifrost_core.typing import JsonDict, Timestamp, Value
from bifrost_core import Tag
from pymodbus.client import AsyncModbusTcpClient
from pymodbus.exceptions import ModbusException

from .plc import PLC, PLCConnection


class ModbusConnection(PLCConnection):
    """Represents a connection to a Modbus device."""

    def __init__(self, host: str, port: int = 502):
        super().__init__(host, port)
        self.client = AsyncModbusTcpClient(host, port)

    async def __aenter__(self) -> "ModbusConnection":
        await self.client.connect()
        self._is_connected = self.client.is_socket_open()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
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
                result = await self.connection.client.read_holding_registers(address, 1)
                if isinstance(result, ModbusException):
                    raise result
                timestamp = Timestamp(asyncio.get_running_loop().time())
                readings[tag] = Reading(tag=tag, value=result.registers[0], timestamp=timestamp)
            except (ModbusException, ValueError) as e:
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
                await self.connection.client.write_register(address, value)
            except (ModbusException, ValueError) as e:
                # In a real implementation, we would log this error.
                pass

    async def get_info(self) -> JsonDict:
        """Get information about the Modbus device."""
        return {
            "host": self.connection.host,
            "port": self.connection.port,
            "is_connected": self.connection.is_connected,
        }
