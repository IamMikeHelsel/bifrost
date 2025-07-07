"""Unified data model for industrial data points."""

from datetime import datetime
from enum import Enum
from typing import Any

from pydantic import BaseModel, ConfigDict, Field


class DataQuality(str, Enum):
    """Data quality indicators following OPC UA standards."""

    GOOD = "good"
    BAD = "bad"
    UNCERTAIN = "uncertain"
    BAD_COMMUNICATION_FAILURE = "bad_communication_failure"
    BAD_DEVICE_FAILURE = "bad_device_failure"
    BAD_SENSOR_FAILURE = "bad_sensor_failure"
    BAD_LAST_KNOWN_VALUE = "bad_last_known_value"
    BAD_NOT_CONNECTED = "bad_not_connected"
    UNCERTAIN_LAST_USABLE_VALUE = "uncertain_last_usable_value"
    UNCERTAIN_SENSOR_NOT_ACCURATE = "uncertain_sensor_not_accurate"


class DataType(str, Enum):
    """Supported data types across protocols."""

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
    DATETIME = "datetime"


class DataPoint(BaseModel):
    """Unified data point representation across all protocols."""

    model_config = ConfigDict(arbitrary_types_allowed=True)

    # Core fields
    name: str = Field(..., description="Human-readable name for the data point")
    address: str = Field(
        ...,
        description="Protocol-specific address (e.g., 'HR1000', 'ns=2;i=1001')",
    )
    value: Any = Field(None, description="Current value of the data point")
    timestamp: datetime = Field(
        default_factory=datetime.utcnow,
        description="UTC timestamp of the reading",
    )

    # Type and quality
    data_type: DataType = Field(..., description="Data type of the value")
    quality: DataQuality = Field(
        DataQuality.GOOD, description="Quality indicator for the data"
    )

    # Protocol information
    protocol: str = Field(
        ..., description="Protocol used (modbus, opcua, s7, etc.)"
    )
    source_device: str | None = Field(
        None, description="Source device identifier"
    )

    # Metadata
    unit: str | None = Field(
        None, description="Engineering unit (e.g., '°C', 'bar', 'rpm')"
    )
    description: str | None = Field(None, description="Additional description")
    metadata: dict[str, Any] = Field(
        default_factory=dict, description="Protocol-specific metadata"
    )

    # Value bounds (optional)
    min_value: int | float | None = Field(
        None, description="Minimum expected value"
    )
    max_value: int | float | None = Field(
        None, description="Maximum expected value"
    )

    def is_valid(self) -> bool:
        """Check if the data point has good quality."""
        return self.quality == DataQuality.GOOD

    def to_dict(self) -> dict[str, Any]:
        """Convert to dictionary for serialization."""
        data = self.model_dump()
        data["timestamp"] = self.timestamp.isoformat()
        return data

    @classmethod
    def from_modbus(
        cls,
        name: str,
        address: int,
        value: Any,
        function_code: int,
        unit: str | None = None,
    ) -> "DataPoint":
        """Create DataPoint from Modbus data."""
        # Map Modbus function codes to data types
        type_map = {
            1: DataType.BOOL,  # Coils
            2: DataType.BOOL,  # Discrete inputs
            3: DataType.UINT16,  # Holding registers
            4: DataType.UINT16,  # Input registers
        }

        return cls(
            name=name,
            address=f"HR{address}" if function_code == 3 else f"IR{address}",
            value=value,
            data_type=type_map.get(function_code, DataType.UINT16),
            protocol="modbus",
            unit=unit,
            metadata={"function_code": function_code},
        )

    @classmethod
    def from_opcua(
        cls,
        name: str,
        node_id: str,
        value: Any,
        data_type: str,
        unit: str | None = None,
    ) -> "DataPoint":
        """Create DataPoint from OPC UA data."""
        return cls(
            name=name,
            address=node_id,
            value=value,
            data_type=DataType(data_type.lower()),
            protocol="opcua",
            unit=unit,
            metadata={"node_id": node_id},
        )
