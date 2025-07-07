"""Modbus implementation for Bifrost.

This module provides Modbus TCP client functionality with async support,
including connection management and read/write operations for holding registers.
"""

import asyncio
from collections.abc import Sequence
from enum import IntEnum
from types import TracebackType
from typing import Any

from pymodbus.client import AsyncModbusTcpClient
from pymodbus.exceptions import ModbusException


class ModbusFunctionCode(IntEnum):
    READ_COILS = 1
    READ_DISCRETE_INPUTS = 2
    READ_HOLDING_REGISTERS = 3
    READ_INPUT_REGISTERS = 4
    WRITE_SINGLE_COIL = 5
    WRITE_SINGLE_REGISTER = 6
    WRITE_MULTIPLE_COILS = 15
    WRITE_MULTIPLE_REGISTERS = 16

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
        """Connects to the Modbus device upon entering the async context."""
        connected = await self.client.connect()
        self._is_connected = connected
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: TracebackType | None,
    ) -> None:
        """Closes the Modbus connection upon exiting the async context."""
        if hasattr(self.client, "close"):
            if asyncio.iscoroutinefunction(self.client.close):
                await self.client.close()
            else:
                self.client.close()
        self._is_connected = False


class ModbusDevice(PLC):
    """Represents a Modbus device."""

    def __init__(self, connection: ModbusConnection):
        """Initializes the ModbusDevice with a ModbusConnection."""
        super().__init__(connection)
        self.connection: ModbusConnection  # For type hinting

    async def read(self, tags: Sequence[Tag]) -> dict[Tag, Reading[Value]]:
        """Read one or more values from the Modbus device."""
        readings: dict[Tag, Reading[Value]] = {}
        for tag in tags:
            try:
                function_code, address, count = self._parse_address(tag.address)
                value: Any  # Allow mixed types for different Modbus operations

                if function_code == 1:  # Read Coils
                    result = await self.connection.client.read_coils(
                        address=address, count=count, slave=1
                    )
                    value = result.bits
                elif function_code == ModbusFunctionCode.READ_DISCRETE_INPUTS:  # Read Discrete Inputs
                    result = await self.connection.client.read_discrete_inputs(
                        address=address, count=count, slave=1
                    )
                    value = result.bits
                elif function_code == ModbusFunctionCode.READ_HOLDING_REGISTERS:  # Read Holding Registers
                    result = (
                        await self.connection.client.read_holding_registers(
                            address=address, count=count, slave=1
                        )
                    )
                    value = result.registers
                elif function_code == ModbusFunctionCode.READ_INPUT_REGISTERS:  # Read Input Registers
                    result = await self.connection.client.read_input_registers(
                        address=address, count=count, slave=1
                    )
                    value = result.registers
                else:
                    raise ValueError(
                        f"Unsupported Modbus function code: {function_code}"
                    )

                if isinstance(result, ModbusException):
                    raise result
                if hasattr(result, "isError") and result.isError():
                    continue  # Skip this tag on error
                timestamp = Timestamp(
                    int(asyncio.get_running_loop().time() * 1_000_000_000)
                )

                if isinstance(value, list):
                    converted_value = [
                        self._convert_to_python(v, tag.data_type) for v in value
                    ]
                else:
                    converted_value = self._convert_to_python(
                        value, tag.data_type
                    )
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
                function_code, address, _ = self._parse_address(tag.address)

                if function_code == ModbusFunctionCode.WRITE_SINGLE_COIL:  # Write Coils
                    if isinstance(value, bool):
                        await self.connection.client.write_coil(
                            address=address, value=value, slave=1
                        )
                    elif isinstance(value, list) and all(
                        isinstance(v, bool) for v in value
                    ):
                        await self.connection.client.write_coils(
                            address=address, values=value, slave=1
                        )
                    else:
                        raise ValueError(
                            "Coil write value must be a boolean or a list of booleans"
                        )
                elif function_code == ModbusFunctionCode.WRITE_SINGLE_REGISTER:  # Write Holding Registers
                    if isinstance(value, int):
                        await self.connection.client.write_register(
                            address=address, value=value, slave=1
                        )
                    elif isinstance(value, list) and all(
                        isinstance(v, int) for v in value
                    ):
                        await self.connection.client.write_registers(
                            address=address, values=value, slave=1
                        )
                    else:
                        raise ValueError(
                            "Register write value must be an integer or a list of integers"
                        )
                else:
                    raise ValueError(
                        f"Unsupported Modbus function code for writing: {function_code}"
                    )

            except (ModbusException, ValueError, Exception):
                # In a real implementation, we would log this error.
                # Catch all exceptions to handle connection errors gracefully
                pass

    def _parse_address(self, address: str) -> tuple[int, int, int]:
        """Parse a Modbus address string and determine the function code.

        Args:
            address: The address string to parse (e.g., "40001", "00001:10").

        Returns:
            A tuple containing (function_code, address, count).
        """
        parts = address.split(":")
        addr_str = parts[0]
        count = int(parts[1]) if len(parts) > 1 else 1

        if addr_str.startswith("0"):
            # Coils (0xxxx)
            function_code = 1  # Read Coils
            address = int(addr_str) - 1
        elif addr_str.startswith("1"):
            # Discrete Inputs (1xxxx)
            function_code = 2  # Read Discrete Inputs
            address = int(addr_str) - 10001
        elif addr_str.startswith("3"):
            # Input Registers (3xxxx)
            function_code = 4  # Read Input Registers
            address = int(addr_str) - 30001
        elif addr_str.startswith("4"):
            # Holding Registers (4xxxx)
            function_code = 3  # Read Holding Registers
            address = int(addr_str) - 40001
        else:
            raise ValueError(f"Invalid Modbus address format: {addr_str}")

        return function_code, address, count

    async def get_info(self) -> JsonDict:
        """Get information about the Modbus device."""
        return {
            "host": self.connection.host,
            "port": self.connection.port,
            "is_connected": self.connection.is_connected,
        }
