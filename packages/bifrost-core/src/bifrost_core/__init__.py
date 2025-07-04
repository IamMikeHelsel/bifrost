"""Bifrost Core - Essential abstractions for industrial IoT.

This package provides the foundational classes and patterns used throughout
the Bifrost ecosystem for connecting to and communicating with industrial
equipment.
"""

__version__ = "0.1.0"

from .base import (
    BaseConnection,
    BaseProtocol,
    ConnectionError,
    ConnectionState,
    DataPoint,
    DataType,
    ProtocolError,
    ProtocolType,
    TimeoutError,
)
from .events import (
    ConnectionStateEvent,
    DataReceivedEvent,
    ErrorEvent,
    Event,
    EventBus,
    EventType,
    emit_event,
    get_global_event_bus,
    subscribe_to_events,
    subscribe_to_events_async,
)
from .pooling import (
    ConnectionPool,
    PooledConnection,
    get_global_pool,
    pooled_connection,
)
from .typing import (
    Address,
    ConnectionString,
    DeviceId,
    DeviceInfo,
    PollingConfig,
    ReadRequest,
    Tag,
    TagName,
    Value,
    WriteRequest,
    get_default_value,
    parse_address,
    validate_data_type_conversion,
)

__all__ = [
    # Base classes and exceptions
    "BaseConnection",
    "BaseProtocol",
    "DataPoint",
    "ConnectionError",
    "ProtocolError",
    "TimeoutError",
    # Enums
    "ConnectionState",
    "DataType",
    "ProtocolType",
    # Events
    "EventBus",
    "Event",
    "EventType",
    "ConnectionStateEvent",
    "DataReceivedEvent",
    "ErrorEvent",
    "get_global_event_bus",
    "emit_event",
    "subscribe_to_events",
    "subscribe_to_events_async",
    # Connection pooling
    "ConnectionPool",
    "PooledConnection",
    "get_global_pool",
    "pooled_connection",
    # Type definitions
    "Tag",
    "DeviceInfo",
    "ReadRequest",
    "WriteRequest",
    "PollingConfig",
    "Address",
    "Value",
    "TagName",
    "DeviceId",
    "ConnectionString",
    "parse_address",
    "validate_data_type_conversion",
    "get_default_value",
]
