"""Bifrost Core: Core abstractions for the Bifrost framework."""

__version__ = "0.1.0"

from .base import BaseConnection, BaseDevice, Reading
from .device_types import (
    DataType,
    DeviceInfo,
    PollingConfig,
    ProtocolType,
    ReadRequest,
    Tag,
    WriteRequest,
    get_default_value,
    parse_address,
    validate_data_type_conversion,
)
from .events import BaseEvent, EventBus
from .features import Feature, FeatureRegistry, HasFeatures
from .pooling import ConnectionPool
from .typing import JsonDict, Timestamp, Value
