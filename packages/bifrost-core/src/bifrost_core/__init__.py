"""Bifrost Core: Core abstractions for the Bifrost framework."""

__version__ = "0.1.0"

from .base import (
    BaseConnection,
    BaseDevice,
    ConnectionState,
    DeviceInfo,
    Reading,
)
from .events import BaseEvent, EventBus
from .features import Feature, FeatureRegistry, HasFeatures
from .patterns import (
    DevicePattern,
    PatternDatabase,
    PatternMatchResult,
    PatternStatus,
    ProtocolSpec,
    VersionRange,
)
from .pattern_storage import PatternManager, PatternStorage
from .pooling import ConnectionPool
from .typing import DataType, JsonDict, Tag, Timestamp, Value
