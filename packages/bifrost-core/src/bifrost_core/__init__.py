"""Bifrost Core - Essential abstractions for industrial IoT.

This package provides the foundational classes and patterns used throughout
the Bifrost ecosystem for connecting to and communicating with industrial
equipment.
"""

__version__ = "0.1.0"

from .base import (
    BaseConnection,
    BaseProtocol, 
    DataPoint,
    ConnectionError,
    ProtocolError,
)
from .events import EventBus, Event
from .pooling import ConnectionPool
from .typing import ProtocolType, DataType, ConnectionState

__all__ = [
    "BaseConnection",
    "BaseProtocol", 
    "DataPoint",
    "ConnectionError",
    "ProtocolError",
    "EventBus",
    "Event", 
    "ConnectionPool",
    "ProtocolType",
    "DataType",
    "ConnectionState",
]