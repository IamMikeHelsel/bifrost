"""Core type definitions for Bifrost."""

from typing import Any, NewType, TypeVar

# Generic type for values read from or written to a device
Value = TypeVar("Value")

# Represents a unique identifier for a data point (e.g., a PLC tag or sensor reading)
# Note: Tag class is now defined in device_types.py

# Represents a timestamp in nanoseconds since the Unix epoch
Timestamp = NewType("Timestamp", int)

# Represents a specific feature or capability
Feature = NewType("Feature", str)

# A generic Bifrost object
T = TypeVar("T")

# A generic dictionary type
JsonDict = dict[str, Any]
