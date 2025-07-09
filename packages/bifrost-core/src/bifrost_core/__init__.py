"""Bifrost Core: Core abstractions for the Bifrost framework."""

__version__ = "0.1.0"

from .base import (
    BaseConnection,
    BaseDevice,
    ConnectionState,
    DeviceInfo,
    Reading,
)
from .device_registry import (
    DeviceRegistry,
    PerformanceMetrics,
    ProtocolSupport,
    RealDevice,
    VirtualDevice,
    VirtualDeviceConfiguration,
)
from .events import BaseEvent, EventBus
from .features import Feature, FeatureRegistry, HasFeatures
from .pooling import ConnectionPool
from .test_integration import (
    DeviceTestTracker,
    TestResult,
    TestSession,
    TestStatus,
)
from .typing import DataType, JsonDict, Tag, Timestamp, Value
