"""Bifrost: The Industrial IoT Framework."""

__version__ = "0.1.0"

from typing import Any

from .connections import ConnectionFactory
from .discovery import discover_devices
from .modbus import ModbusDevice
from .plc import PLC

# Smart imports for optional dependencies
try:
    from bifrost_opcua import OPCUAClient  # type: ignore
except ImportError:

    def OPCUAClient(*args: Any, **kwargs: Any) -> Any:
        raise ImportError(
            "OPC UA support requires: pip install bifrost-opcua\n"
            "Or for everything: pip install bifrost-all"
        )


try:
    from bifrost_analytics import AnalyticsEngine  # type: ignore
except ImportError:

    def AnalyticsEngine(*args: Any, **kwargs: Any) -> Any:
        raise ImportError(
            "Analytics support requires: pip install bifrost-analytics\n"
            "Or for everything: pip install bifrost-all"
        )


try:
    from bifrost_cloud import CloudBridge  # type: ignore
except ImportError:

    def CloudBridge(*args: Any, **kwargs: Any) -> Any:
        raise ImportError(
            "Cloud support requires: pip install bifrost-cloud\n"
            "Or for everything: pip install bifrost-all"
        )
