"""Core type definitions for Bifrost."""

from enum import Enum
from typing import Any, NewType, TypeVar


class DataType(Enum):
    """Represents common data types for industrial protocols."""

    INT16 = "int16"
    UINT16 = "uint16"
    INT32 = "int32"
    UINT32 = "uint32"
    FLOAT32 = "float32"
    FLOAT64 = "float64"
    BOOLEAN = "boolean"
    STRING = "string"
    BYTE = "byte"
    # Add more as needed


# Generic type for values read from or written to a device
Value = TypeVar("Value")

# Represents a unique identifier for a data point (e.g., a PLC tag or sensor reading)
from pydantic import BaseModel, Field
from typing import Optional

class Tag(BaseModel):
    """Represents a unique identifier for a data point (e.g., a PLC tag or sensor reading)."""
    name: str = Field(..., description="Human-readable name of the tag.")
    address: str = Field(..., description="The device-specific address or identifier for the tag.")
    data_type: DataType = Field(..., description="The expected data type of the tag.")
    description: Optional[str] = Field(None, description="A brief description of the tag.")
    units: Optional[str] = Field(None, description="Units of measurement for the tag's value.")
    read_only: bool = Field(False, description="True if the tag is read-only, False otherwise.")
    scaling_factor: Optional[float] = Field(None, description="Factor to multiply the raw value by.")
    offset: Optional[float] = Field(None, description="Value to add to the scaled value.")

    def apply_scaling(self, raw_value: Any) -> Any:
        """Applies scaling and offset to a raw value."""
        if self.scaling_factor is not None:
            scaled_value = raw_value * self.scaling_factor
        else:
            scaled_value = raw_value

        if self.offset is not None:
            scaled_value += self.offset

        # Attempt to convert to the target data type if scaling was applied
        if self.scaling_factor is not None or self.offset is not None:
            if self.data_type == DataType.INT16 or self.data_type == DataType.INT32 or self.data_type == DataType.UINT16 or self.data_type == DataType.UINT32:
                return int(scaled_value)
            elif self.data_type == DataType.FLOAT32 or self.data_type == DataType.FLOAT64:
                return float(scaled_value)
        return scaled_value

    def __str__(self) -> str:
        return f"Tag({self.name}, {self.address}, {self.data_type.value})"

# Represents a timestamp in nanoseconds since the Unix epoch
Timestamp = NewType("Timestamp", int)

# Represents a specific feature or capability
Feature = NewType("Feature", str)

# A generic Bifrost object
T = TypeVar("T")

# A generic dictionary type
JsonDict = dict[str, Any]
