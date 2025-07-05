"""Device and protocol type definitions for Bifrost."""

from enum import Enum
from typing import Any, Dict, List, Optional, Union

from pydantic import BaseModel, Field


class ProtocolType(str, Enum):
    """Supported protocol types."""
    MODBUS_TCP = "modbus_tcp"
    MODBUS_RTU = "modbus_rtu"
    OPCUA = "opcua"
    S7 = "s7"
    ETHERNET_IP = "ethernet_ip"


class DataType(str, Enum):
    """Supported data types."""
    BOOL = "bool"
    INT16 = "int16"
    INT32 = "int32"
    INT64 = "int64"
    UINT16 = "uint16"
    UINT32 = "uint32"
    UINT64 = "uint64"
    FLOAT32 = "float32"
    FLOAT64 = "float64"
    STRING = "string"
    BYTES = "bytes"


class Tag(BaseModel):
    """Represents a data point tag."""
    name: str
    address: str
    data_type: DataType
    description: Optional[str] = None
    units: Optional[str] = None
    read_only: bool = False
    scaling_factor: Optional[float] = None
    offset: Optional[float] = None

    def apply_scaling(self, raw_value: Union[int, float]) -> Union[int, float]:
        """Apply scaling to a raw value."""
        if self.scaling_factor is None and self.offset is None:
            return raw_value
        
        scaled = raw_value
        if self.scaling_factor is not None:
            scaled *= self.scaling_factor
        if self.offset is not None:
            scaled += self.offset
            
        # Return int for integer data types
        if self.data_type in [DataType.INT16, DataType.INT32, DataType.INT64, 
                             DataType.UINT16, DataType.UINT32, DataType.UINT64]:
            return int(scaled)
        return scaled

    def __str__(self) -> str:
        return f"Tag({self.name}, {self.address}, {self.data_type})"


class DeviceInfo(BaseModel):
    """Device information and connection details."""
    device_id: str
    protocol: ProtocolType
    host: str
    port: Optional[int] = None
    name: Optional[str] = None
    manufacturer: Optional[str] = None
    model: Optional[str] = None
    
    def model_post_init(self, __context: Any) -> None:
        """Set default name to device_id if not provided."""
        if self.name is None:
            self.name = self.device_id

    @property
    def connection_string(self) -> str:
        """Generate connection string."""
        protocol_map = {
            ProtocolType.MODBUS_TCP: "modbus_tcp",
            ProtocolType.MODBUS_RTU: "modbus_rtu", 
            ProtocolType.OPCUA: "opcua",
            ProtocolType.S7: "s7",
            ProtocolType.ETHERNET_IP: "ethernet_ip",
        }
        protocol_str = protocol_map.get(self.protocol, str(self.protocol))
        
        if self.port:
            return f"{protocol_str}://{self.host}:{self.port}"
        return f"{protocol_str}://{self.host}"


class ReadRequest(BaseModel):
    """Request to read data from tags."""
    tags: List[Tag]
    device_id: str
    timeout: float = 5.0
    priority: int = 0
    
    @property
    def tag_count(self) -> int:
        """Number of tags in this request."""
        return len(self.tags)


class WriteRequest(BaseModel):
    """Request to write data to a tag."""
    tag: Tag
    value: Any
    device_id: str
    timeout: float = 5.0
    priority: int = 0
    
    def model_post_init(self, __context: Any) -> None:
        """Validate that tag is not read-only."""
        if self.tag.read_only:
            raise ValueError(f"Cannot write to read-only tag: {self.tag.name}")


class PollingConfig(BaseModel):
    """Configuration for polling operations."""
    interval_ms: int = 1000
    max_batch_size: int = 100
    enabled: bool = True
    max_consecutive_errors: int = 5
    on_error_interval_ms: Optional[int] = None
    
    def model_post_init(self, __context: Any) -> None:
        """Set default error interval to 2x normal interval."""
        if self.on_error_interval_ms is None:
            self.on_error_interval_ms = self.interval_ms * 2
    
    @property
    def interval_seconds(self) -> float:
        """Interval in seconds."""
        return self.interval_ms / 1000.0
    
    @property
    def on_error_interval_seconds(self) -> float:
        """Error interval in seconds."""
        return (self.on_error_interval_ms or self.interval_ms) / 1000.0


def parse_address(address: str, protocol: ProtocolType) -> Dict[str, Any]:
    """Parse an address string based on protocol type."""
    if protocol == ProtocolType.MODBUS_TCP:
        if ":" in address:
            register_type, addr_str = address.split(":", 1)
            return {"register_type": register_type, "address": int(addr_str)}
        else:
            # Default to holding register for numeric addresses
            addr_num = int(address)
            if 40000 <= addr_num <= 49999:
                return {"register_type": "holding", "address": addr_num}
            elif 30000 <= addr_num <= 39999:
                return {"register_type": "input", "address": addr_num}
            elif 10000 <= addr_num <= 19999:
                return {"register_type": "discrete", "address": addr_num}
            elif 1 <= addr_num <= 9999:
                return {"register_type": "coil", "address": addr_num}
            return {"register_type": "holding", "address": addr_num}
    elif protocol == ProtocolType.OPCUA:
        return {"node_id": address}
    else:
        return {"address": address}


def validate_data_type_conversion(value: Any, data_type: DataType) -> bool:
    """Validate if a value can be converted to the specified data type."""
    if value is None:
        return False
    
    try:
        if data_type == DataType.BOOL:
            return True  # Most things can be converted to bool
        elif data_type in [DataType.INT16, DataType.INT32, DataType.INT64,
                          DataType.UINT16, DataType.UINT32, DataType.UINT64]:
            int(str(value))
            return True
        elif data_type in [DataType.FLOAT32, DataType.FLOAT64]:
            float(str(value))
            return True
        elif data_type == DataType.STRING:
            return True  # Everything can be converted to string
        elif data_type == DataType.BYTES:
            return isinstance(value, (bytes, bytearray)) or hasattr(value, 'encode')
        return False
    except (ValueError, TypeError):
        return False


def get_default_value(data_type: DataType) -> Any:
    """Get the default value for a data type."""
    defaults = {
        DataType.BOOL: False,
        DataType.INT16: 0,
        DataType.INT32: 0,
        DataType.INT64: 0,
        DataType.UINT16: 0,
        DataType.UINT32: 0,
        DataType.UINT64: 0,
        DataType.FLOAT32: 0.0,
        DataType.FLOAT64: 0.0,
        DataType.STRING: "",
        DataType.BYTES: b"",
    }
    return defaults.get(data_type)