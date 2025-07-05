"""Modbus protocol implementation using pymodbus."""

import builtins
from typing import Any
from urllib.parse import urlparse

from bifrost_core import (
    BaseConnection,
    BaseProtocol,
    ConnectionError,
    ConnectionState,
    ConnectionStateEvent,
    DataReceivedEvent,
    ErrorEvent,
    ProtocolError,
    ProtocolType,
    TimeoutError,
    emit_event,
)
from pymodbus.client import AsyncModbusTcpClient
from pymodbus.exceptions import ConnectionException, ModbusException
from pymodbus.pdu import ExceptionResponse


class ModbusConnection(BaseConnection):
    """Base class for Modbus connections with real pymodbus implementation."""

    def __init__(self, host: str, port: int = 502, **kwargs):
        super().__init__(host, port, **kwargs)
        self.unit_id = kwargs.get("unit_id", 1)
        self._client: AsyncModbusTcpClient | None = None

    async def connect(self) -> None:
        """Connect to Modbus device."""
        try:
            self._state = ConnectionState.CONNECTING
            emit_event(
                ConnectionStateEvent(
                    self.connection_id,
                    ConnectionState.DISCONNECTED,
                    ConnectionState.CONNECTING,
                )
            )

            self._client = AsyncModbusTcpClient(
                host=self.host,
                port=self.port,
                timeout=self.timeout,
                retries=self.retry_attempts,
            )

            # Connect and verify
            connected = await self._client.connect()
            if not connected:
                raise ConnectionError(f"Failed to connect to {self.host}:{self.port}")

            self._state = ConnectionState.CONNECTED
            emit_event(
                ConnectionStateEvent(
                    self.connection_id,
                    ConnectionState.CONNECTING,
                    ConnectionState.CONNECTED,
                )
            )

        except Exception as e:
            self._state = ConnectionState.FAILED
            emit_event(ErrorEvent(self.connection_id, e))
            if isinstance(e, ModbusException | ConnectionException):
                raise ConnectionError(f"Modbus connection failed: {e}") from e
            raise

    async def disconnect(self) -> None:
        """Disconnect from Modbus device."""
        if self._client:
            try:
                await self._client.close()
            except Exception as e:
                emit_event(ErrorEvent(self.connection_id, e))
            finally:
                self._client = None

        old_state = self._state
        self._state = ConnectionState.DISCONNECTED
        emit_event(
            ConnectionStateEvent(
                self.connection_id, old_state, ConnectionState.DISCONNECTED
            )
        )

    async def read_raw(self, address: str, count: int = 1) -> list[Any]:
        """Read raw data from Modbus device."""
        if not self._client or not self.is_connected:
            raise ConnectionError("Not connected to Modbus device")

        try:
            # Parse address format: "40001" or "holding:40001" or "coil:1"
            register_type, reg_address = self._parse_address(address)

            # Perform the actual Modbus read based on register type
            if register_type == "coil":
                response = await self._client.read_coils(
                    reg_address, count, slave=self.unit_id
                )
            elif register_type == "discrete":
                response = await self._client.read_discrete_inputs(
                    reg_address, count, slave=self.unit_id
                )
            elif register_type == "input":
                response = await self._client.read_input_registers(
                    reg_address, count, slave=self.unit_id
                )
            elif register_type == "holding":
                response = await self._client.read_holding_registers(
                    reg_address, count, slave=self.unit_id
                )
            else:
                raise ProtocolError(f"Unknown register type: {register_type}")

            # Check for errors
            if response.isError():
                if isinstance(response, ExceptionResponse):
                    raise ProtocolError(f"Modbus exception: {response}")
                else:
                    raise ProtocolError(f"Modbus read error: {response}")

            # Extract values
            if register_type in ("coil", "discrete"):
                values = response.bits[:count]
            else:
                values = response.registers[:count]

            # Emit data received event
            emit_event(
                DataReceivedEvent(self.connection_id, address, values, register_type)
            )

            return values

        except ModbusException as e:
            error = ProtocolError(f"Modbus read error at {address}: {e}")
            emit_event(ErrorEvent(self.connection_id, error))
            raise error from e
        except builtins.TimeoutError as e:
            error = TimeoutError(f"Modbus read timeout at {address}")
            emit_event(ErrorEvent(self.connection_id, error))
            raise error from e
        except Exception as e:
            emit_event(ErrorEvent(self.connection_id, e))
            raise

    async def write_raw(self, address: str, values: list[Any]) -> None:
        """Write raw data to Modbus device."""
        if not self._client or not self.is_connected:
            raise ConnectionError("Not connected to Modbus device")

        try:
            register_type, reg_address = self._parse_address(address)

            if len(values) == 1:
                # Single write
                value = values[0]
                if register_type == "coil":
                    response = await self._client.write_coil(
                        reg_address, bool(value), slave=self.unit_id
                    )
                elif register_type == "holding":
                    response = await self._client.write_register(
                        reg_address, int(value), slave=self.unit_id
                    )
                else:
                    raise ProtocolError(f"Cannot write to {register_type} registers")
            else:
                # Multiple write
                if register_type == "coil":
                    response = await self._client.write_coils(
                        reg_address, [bool(v) for v in values], slave=self.unit_id
                    )
                elif register_type == "holding":
                    response = await self._client.write_registers(
                        reg_address, [int(v) for v in values], slave=self.unit_id
                    )
                else:
                    raise ProtocolError(f"Cannot write to {register_type} registers")

            # Check for errors
            if response.isError():
                if isinstance(response, ExceptionResponse):
                    raise ProtocolError(f"Modbus exception: {response}")
                else:
                    raise ProtocolError(f"Modbus write error: {response}")

        except ModbusException as e:
            error = ProtocolError(f"Modbus write error at {address}: {e}")
            emit_event(ErrorEvent(self.connection_id, error))
            raise error from e
        except Exception as e:
            emit_event(ErrorEvent(self.connection_id, e))
            raise

    def _parse_address(self, address: str) -> tuple[str, int]:
        """Parse address string into register type and address."""
        if ":" in address:
            register_type, addr_str = address.split(":", 1)
            reg_address = int(addr_str)
        else:
            # Default parsing based on address range (Modbus convention)
            reg_address = int(address)
            if 1 <= reg_address <= 9999:
                register_type = "coil"
            elif 10001 <= reg_address <= 19999:
                register_type = "discrete"
                reg_address -= 10000  # Convert to 0-based
            elif 30001 <= reg_address <= 39999:
                register_type = "input"
                reg_address -= 30000  # Convert to 0-based
            elif 40001 <= reg_address <= 49999:
                register_type = "holding"
                reg_address -= 40000  # Convert to 0-based
            else:
                # Assume holding register for any other address
                register_type = "holding"

        return register_type, reg_address

    async def health_check(self) -> bool:
        """Perform health check by reading a single coil."""
        try:
            # Try to read coil 0 (minimal operation)
            await self.read_raw("coil:0", count=1)
            return True
        except Exception:
            return False

    @property
    def protocol(self) -> str:
        """Get protocol name."""
        return "modbus"

    @property
    def connection_string(self) -> str:
        """Get connection string."""
        return f"modbus://{self.host}:{self.port}/{self.unit_id}"

    def __str__(self) -> str:
        """String representation."""
        return f"{self.__class__.__name__}(host='{self.host}', port={self.port}, unit_id={self.unit_id})"

    async def read_batch(self, addresses: list[str]) -> dict[str, list[Any]]:
        """Read multiple addresses in batch for better performance.
        
        Args:
            addresses: List of addresses to read (e.g., ["40001", "40002", "coil:1"])
            
        Returns:
            Dictionary mapping addresses to their values
            
        Raises:
            ConnectionError: If not connected
            ProtocolError: If Modbus operation fails
        """
        if not self._client or not self.is_connected:
            raise ConnectionError("Not connected to Modbus device")
            
        results = {}
        
        # Group addresses by register type for efficient batch reads
        address_groups = self._group_addresses_by_type(addresses)
        
        try:
            for register_type, address_list in address_groups.items():
                if not address_list:
                    continue
                    
                # Sort addresses to enable range optimization
                sorted_addresses = sorted(address_list, key=lambda x: x[1])  # Sort by register address
                
                # Read ranges efficiently
                for start_addr, end_addr, original_addresses in self._optimize_address_ranges(sorted_addresses):
                    count = end_addr - start_addr + 1
                    
                    # Perform batch read
                    if register_type == "coil":
                        response = await self._client.read_coils(start_addr, count, slave=self.unit_id)
                    elif register_type == "discrete":
                        response = await self._client.read_discrete_inputs(start_addr, count, slave=self.unit_id)
                    elif register_type == "input":
                        response = await self._client.read_input_registers(start_addr, count, slave=self.unit_id)
                    elif register_type == "holding":
                        response = await self._client.read_holding_registers(start_addr, count, slave=self.unit_id)
                    else:
                        raise ProtocolError(f"Unknown register type: {register_type}")
                    
                    # Check for errors
                    if response.isError():
                        if isinstance(response, ExceptionResponse):
                            raise ProtocolError(f"Modbus exception: {response}")
                        else:
                            raise ProtocolError(f"Modbus read error: {response}")
                    
                    # Extract values and map back to original addresses
                    if register_type in ("coil", "discrete"):
                        values = response.bits[:count]
                    else:
                        values = response.registers[:count]
                    
                    # Map values back to original addresses
                    for original_addr, reg_addr in original_addresses:
                        value_index = reg_addr - start_addr
                        results[original_addr] = [values[value_index]]
            
            # Emit batch data received event
            emit_event(
                DataReceivedEvent(
                    self.connection_id, 
                    f"batch:{len(addresses)}", 
                    results, 
                    "batch"
                )
            )
            
            return results
            
        except ModbusException as e:
            error = ProtocolError(f"Modbus batch read error: {e}")
            emit_event(ErrorEvent(self.connection_id, error))
            raise error from e
        except builtins.TimeoutError as e:
            error = TimeoutError("Modbus batch read timeout")
            emit_event(ErrorEvent(self.connection_id, error))
            raise error from e
        except Exception as e:
            emit_event(ErrorEvent(self.connection_id, e))
            raise

    async def write_batch(self, address_values: dict[str, Any]) -> None:
        """Write multiple addresses in batch for better performance.
        
        Args:
            address_values: Dictionary mapping addresses to values
                           (e.g., {"40001": 100, "40002": 200, "coil:1": True})
                           
        Raises:
            ConnectionError: If not connected
            ProtocolError: If Modbus operation fails
        """
        if not self._client or not self.is_connected:
            raise ConnectionError("Not connected to Modbus device")
            
        # Group writes by register type
        write_groups = {}
        for address, value in address_values.items():
            register_type, reg_address = self._parse_address(address)
            
            if register_type not in write_groups:
                write_groups[register_type] = []
            write_groups[register_type].append((reg_address, value, address))
        
        try:
            for register_type, writes in write_groups.items():
                if register_type not in ("coil", "holding"):
                    raise ProtocolError(f"Cannot write to {register_type} registers")
                
                # Sort by address for range optimization
                sorted_writes = sorted(writes, key=lambda x: x[0])
                
                # Check if we can do range writes
                ranges = self._optimize_write_ranges(sorted_writes)
                
                for start_addr, values, original_addresses in ranges:
                    if len(values) == 1:
                        # Single write
                        value = values[0]
                        if register_type == "coil":
                            response = await self._client.write_coil(
                                start_addr, bool(value), slave=self.unit_id
                            )
                        else:  # holding
                            response = await self._client.write_register(
                                start_addr, int(value), slave=self.unit_id
                            )
                    else:
                        # Multiple write
                        if register_type == "coil":
                            response = await self._client.write_coils(
                                start_addr, [bool(v) for v in values], slave=self.unit_id
                            )
                        else:  # holding
                            response = await self._client.write_registers(
                                start_addr, [int(v) for v in values], slave=self.unit_id
                            )
                    
                    # Check for errors
                    if response.isError():
                        if isinstance(response, ExceptionResponse):
                            raise ProtocolError(f"Modbus exception: {response}")
                        else:
                            raise ProtocolError(f"Modbus write error: {response}")
            
        except ModbusException as e:
            error = ProtocolError(f"Modbus batch write error: {e}")
            emit_event(ErrorEvent(self.connection_id, error))
            raise error from e
        except Exception as e:
            emit_event(ErrorEvent(self.connection_id, e))
            raise

    def _group_addresses_by_type(self, addresses: list[str]) -> dict[str, list[tuple[str, int]]]:
        """Group addresses by register type for efficient batch reading."""
        groups = {"coil": [], "discrete": [], "input": [], "holding": []}
        
        for addr in addresses:
            register_type, reg_address = self._parse_address(addr)
            groups[register_type].append((addr, reg_address))
        
        return groups

    def _optimize_address_ranges(self, address_list: list[tuple[str, int]]) -> list[tuple[int, int, list[tuple[str, int]]]]:
        """Optimize address list into efficient read ranges.
        
        Returns:
            List of (start_addr, end_addr, original_addresses) tuples
        """
        if not address_list:
            return []
        
        ranges = []
        current_start = address_list[0][1]
        current_end = address_list[0][1]
        current_addresses = [address_list[0]]
        
        for i in range(1, len(address_list)):
            addr, reg_addr = address_list[i]
            
            # If address is consecutive or within small gap, extend range
            if reg_addr <= current_end + 5:  # Allow small gaps for efficiency
                current_end = max(current_end, reg_addr)
                current_addresses.append((addr, reg_addr))
            else:
                # Start new range
                ranges.append((current_start, current_end, current_addresses))
                current_start = reg_addr
                current_end = reg_addr
                current_addresses = [(addr, reg_addr)]
        
        # Add final range
        ranges.append((current_start, current_end, current_addresses))
        
        return ranges

    def _optimize_write_ranges(self, writes: list[tuple[int, Any, str]]) -> list[tuple[int, list[Any], list[str]]]:
        """Optimize write operations into efficient ranges.
        
        Returns:
            List of (start_addr, values, original_addresses) tuples
        """
        if not writes:
            return []
        
        ranges = []
        current_start = writes[0][0]
        current_values = [writes[0][1]]
        current_addresses = [writes[0][2]]
        
        for i in range(1, len(writes)):
            reg_addr, value, original_addr = writes[i]
            
            # If address is consecutive, extend range
            if reg_addr == current_start + len(current_values):
                current_values.append(value)
                current_addresses.append(original_addr)
            else:
                # Start new range
                ranges.append((current_start, current_values, current_addresses))
                current_start = reg_addr
                current_values = [value]
                current_addresses = [original_addr]
        
        # Add final range
        ranges.append((current_start, current_values, current_addresses))
        
        return ranges


class ModbusTCPConnection(ModbusConnection):
    """Modbus TCP connection implementation."""

    @classmethod
    def from_url(cls, url: str, **kwargs) -> "ModbusTCPConnection":
        """Create connection from URL string."""
        parsed = urlparse(url)
        host = parsed.hostname or "localhost"
        port = parsed.port or 502
        return cls(host=host, port=port, **kwargs)


class ModbusRTUConnection(ModbusConnection):
    """Modbus RTU connection implementation."""

    def __init__(self, port: str, baudrate: int = 9600, **kwargs):
        # For RTU, "host" is the serial port
        super().__init__(host=port, port=None, **kwargs)
        self.serial_port = port
        self.baudrate = baudrate

    @classmethod
    def from_url(cls, url: str, **kwargs) -> "ModbusRTUConnection":
        """Create RTU connection from URL string."""
        parsed = urlparse(url)
        port = parsed.path or "/dev/ttyUSB0"
        baudrate = kwargs.get("baudrate", 9600)
        return cls(port=port, baudrate=baudrate, **kwargs)


class ModbusProtocol(BaseProtocol):
    """Modbus protocol handler."""

    def get_protocol_type(self) -> ProtocolType:
        return ProtocolType.MODBUS_TCP

    async def create_connection(
        self, connection_string: str, **kwargs
    ) -> BaseConnection:
        return ModbusTCPConnection.from_url(connection_string, **kwargs)

    def parse_connection_string(self, connection_string: str) -> dict:
        parsed = urlparse(connection_string)
        return {
            "host": parsed.hostname or "localhost",
            "port": parsed.port or 502,
            "unit_id": 1,
        }
