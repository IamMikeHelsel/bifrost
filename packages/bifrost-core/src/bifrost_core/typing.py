"""Type definitions and utilities for Bifrost."""

from typing import Any, Dict, List, Optional, Protocol, Union
from datetime import datetime

from .base import DataType, ProtocolType


# Type aliases for common patterns
Address = str
Value = Any
TagName = str
DeviceId = str
ConnectionString = str


class Tag:
    """Represents a tag definition for reading from industrial devices."""
    
    def __init__(
        self,
        name: TagName,
        address: Address,
        data_type: DataType,
        description: Optional[str] = None,
        scaling_factor: Optional[float] = None,
        offset: Optional[float] = None,
        units: Optional[str] = None,
        read_only: bool = False,
    ):
        self.name = name
        self.address = address
        self.data_type = data_type
        self.description = description
        self.scaling_factor = scaling_factor
        self.offset = offset
        self.units = units
        self.read_only = read_only
    
    def apply_scaling(self, raw_value: Value) -> Value:
        """Apply scaling and offset to raw value."""
        if self.scaling_factor is None and self.offset is None:
            return raw_value
        
        value = float(raw_value)
        if self.scaling_factor is not None:
            value *= self.scaling_factor
        if self.offset is not None:
            value += self.offset
        
        # Convert back to original type if it was an integer
        if self.data_type in (DataType.INT16, DataType.UINT16, 
                             DataType.INT32, DataType.UINT32,
                             DataType.INT64, DataType.UINT64):
            return int(value)
        
        return value
    
    def __str__(self) -> str:
        return f"Tag({self.name}, {self.address}, {self.data_type.value})"
    
    def __repr__(self) -> str:
        return (f"Tag(name='{self.name}', address='{self.address}', "
                f"data_type={self.data_type}, read_only={self.read_only})")


class DeviceInfo:
    """Information about a discovered or connected device."""
    
    def __init__(
        self,
        device_id: DeviceId,
        protocol: ProtocolType,
        host: str,
        port: Optional[int] = None,
        name: Optional[str] = None,
        manufacturer: Optional[str] = None,
        model: Optional[str] = None,
        firmware_version: Optional[str] = None,
        last_seen: Optional[datetime] = None,
        additional_info: Optional[Dict[str, Any]] = None,
    ):
        self.device_id = device_id
        self.protocol = protocol
        self.host = host
        self.port = port
        self.name = name or device_id
        self.manufacturer = manufacturer
        self.model = model
        self.firmware_version = firmware_version
        self.last_seen = last_seen or datetime.now()
        self.additional_info = additional_info or {}
    
    @property
    def connection_string(self) -> str:
        """Generate connection string for this device."""
        if self.port:
            return f"{self.protocol.value}://{self.host}:{self.port}"
        else:
            return f"{self.protocol.value}://{self.host}"
    
    def __str__(self) -> str:
        return f"Device({self.name}, {self.protocol.value}, {self.host})"


class ReadRequest:
    """Request to read data from a device."""
    
    def __init__(
        self,
        tags: List[Tag],
        device_id: Optional[DeviceId] = None,
        timeout: Optional[float] = None,
        priority: int = 0,
    ):
        self.tags = tags
        self.device_id = device_id
        self.timeout = timeout
        self.priority = priority
        self.created_at = datetime.now()
    
    @property
    def tag_count(self) -> int:
        return len(self.tags)
    
    def __str__(self) -> str:
        return f"ReadRequest({self.tag_count} tags)"


class WriteRequest:
    """Request to write data to a device."""
    
    def __init__(
        self,
        tag: Tag,
        value: Value,
        device_id: Optional[DeviceId] = None,
        timeout: Optional[float] = None,
        priority: int = 0,
    ):
        self.tag = tag
        self.value = value
        self.device_id = device_id
        self.timeout = timeout
        self.priority = priority
        self.created_at = datetime.now()
        
        if tag.read_only:
            raise ValueError(f"Tag {tag.name} is marked as read-only")
    
    def __str__(self) -> str:
        return f"WriteRequest({self.tag.name}={self.value})"


class PollingConfig:
    """Configuration for polling operations."""
    
    def __init__(
        self,
        interval_ms: int = 1000,
        max_batch_size: int = 100,
        enabled: bool = True,
        on_error_interval_ms: Optional[int] = None,
        max_consecutive_errors: int = 5,
    ):
        self.interval_ms = interval_ms
        self.max_batch_size = max_batch_size
        self.enabled = enabled
        self.on_error_interval_ms = on_error_interval_ms or (interval_ms * 2)
        self.max_consecutive_errors = max_consecutive_errors
    
    @property
    def interval_seconds(self) -> float:
        return self.interval_ms / 1000.0
    
    @property
    def on_error_interval_seconds(self) -> float:
        return self.on_error_interval_ms / 1000.0


# Protocol-specific type hints
class ConnectionFactory(Protocol):
    """Protocol for connection factory functions."""
    
    async def __call__(self, **kwargs: Any) -> "BaseConnection":
        """Create a new connection with the given parameters."""
        ...


class ProtocolHandler(Protocol):
    """Protocol for protocol-specific handlers."""
    
    def get_protocol_type(self) -> ProtocolType:
        """Get the protocol type this handler supports."""
        ...
    
    async def create_connection(self, connection_string: str, **kwargs: Any) -> "BaseConnection":
        """Create a connection from a connection string."""
        ...
    
    def parse_connection_string(self, connection_string: str) -> Dict[str, Any]:
        """Parse connection string into parameters."""
        ...


# Utility functions for type handling
def parse_address(address: str, protocol: ProtocolType) -> Dict[str, Any]:
    """Parse an address string based on protocol type."""
    if protocol == ProtocolType.MODBUS_TCP:
        # Modbus addresses: "40001", "coil:1", "holding:40001"
        if ":" in address:
            register_type, addr = address.split(":", 1)
            return {"register_type": register_type, "address": int(addr)}
        else:
            # Default to holding register
            return {"register_type": "holding", "address": int(address)}
    
    elif protocol == ProtocolType.OPCUA:
        # OPC UA node IDs: "ns=2;i=1", "ns=2;s=Temperature"
        return {"node_id": address}
    
    else:
        # Generic address
        return {"address": address}


def validate_data_type_conversion(value: Any, target_type: DataType) -> bool:
    """Check if a value can be converted to the target data type."""
    try:
        if target_type == DataType.BOOL:
            bool(value)
        elif target_type in (DataType.INT16, DataType.UINT16, 
                           DataType.INT32, DataType.UINT32,
                           DataType.INT64, DataType.UINT64):
            int(value)
        elif target_type in (DataType.FLOAT32, DataType.FLOAT64):
            float(value)
        elif target_type == DataType.STRING:
            str(value)
        return True
    except (ValueError, TypeError):
        return False


def get_default_value(data_type: DataType) -> Any:
    """Get a default value for a data type."""
    defaults = {
        DataType.BOOL: False,
        DataType.INT16: 0,
        DataType.UINT16: 0,
        DataType.INT32: 0,
        DataType.UINT32: 0,
        DataType.INT64: 0,
        DataType.UINT64: 0,
        DataType.FLOAT32: 0.0,
        DataType.FLOAT64: 0.0,
        DataType.STRING: "",
        DataType.BYTES: b"",
    }
    return defaults.get(data_type, None)