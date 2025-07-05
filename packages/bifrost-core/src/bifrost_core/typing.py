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
Tag = NewType("Tag", str)

# Represents a timestamp in nanoseconds since the Unix epoch
Timestamp = NewType("Timestamp", int)

# Represents a specific feature or capability
Feature = NewType("Feature", str)

# A generic Bifrost object
T = TypeVar("T")

# A generic dictionary type
JsonDict = dict[str, Any]
