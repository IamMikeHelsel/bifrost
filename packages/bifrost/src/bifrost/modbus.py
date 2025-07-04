"""Modbus protocol implementation using pymodbus."""

import asyncio
from typing import Any, List, Optional, Union
from urllib.parse import urlparse, parse_qs

from pymodbus.client import AsyncModbusTcpClient, AsyncModbusSerialClient
from pymodbus.exceptions import ModbusException, ConnectionException
from pymodbus.pdu import ExceptionResponse

from bifrost_core import (
    BaseConnection, BaseProtocol, ProtocolType, DataType,
    ConnectionError, ProtocolError, TimeoutError, ConnectionState,
    emit_event, ConnectionStateEvent, DataReceivedEvent, ErrorEvent
)


class ModbusConnection(BaseConnection):
    """Base class for Modbus connections with real pymodbus implementation."""
    
    def __init__(self, host: str, port: int = 502, **kwargs):
        super().__init__(host, port, **kwargs)
        self.unit_id = kwargs.get("unit_id", 1)
        self._client: Optional[AsyncModbusTcpClient] = None
        
    async def connect(self) -> None:
        """Connect to Modbus device."""
        try:
            self._state = ConnectionState.CONNECTING
            emit_event(ConnectionStateEvent(
                self.connection_id, 
                ConnectionState.DISCONNECTED, 
                ConnectionState.CONNECTING
            ))
            
            self._client = AsyncModbusTcpClient(
                host=self.host,
                port=self.port,
                timeout=self.timeout,
                retries=self.retry_attempts
            )
            
            # Connect and verify
            connected = await self._client.connect()
            if not connected:
                raise ConnectionError(f"Failed to connect to {self.host}:{self.port}")
            
            self._state = ConnectionState.CONNECTED
            emit_event(ConnectionStateEvent(
                self.connection_id,
                ConnectionState.CONNECTING,
                ConnectionState.CONNECTED
            ))
            
        except Exception as e:
            self._state = ConnectionState.FAILED
            emit_event(ErrorEvent(self.connection_id, e))
            if isinstance(e, (ModbusException, ConnectionException)):
                raise ConnectionError(f"Modbus connection failed: {e}")
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
        emit_event(ConnectionStateEvent(
            self.connection_id,
            old_state,
            ConnectionState.DISCONNECTED
        ))
    
    async def read_raw(self, address: str, count: int = 1) -> List[Any]:
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
            emit_event(DataReceivedEvent(
                self.connection_id,
                address,
                values,
                register_type
            ))
            
            return values
            
        except ModbusException as e:
            error = ProtocolError(f"Modbus read error at {address}: {e}")
            emit_event(ErrorEvent(self.connection_id, error))
            raise error
        except asyncio.TimeoutError as e:
            error = TimeoutError(f"Modbus read timeout at {address}")
            emit_event(ErrorEvent(self.connection_id, error))
            raise error
        except Exception as e:
            emit_event(ErrorEvent(self.connection_id, e))
            raise
    
    async def write_raw(self, address: str, values: List[Any]) -> None:
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
            raise error
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
    
    async def create_connection(self, connection_string: str, **kwargs) -> BaseConnection:
        return ModbusTCPConnection.from_url(connection_string, **kwargs)
    
    def parse_connection_string(self, connection_string: str) -> dict:
        parsed = urlparse(connection_string)
        return {
            "host": parsed.hostname or "localhost",
            "port": parsed.port or 502,
            "unit_id": kwargs.get("unit_id", 1)
        }