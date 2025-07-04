"""Bifrost - Bridge your industrial equipment to modern IT infrastructure.

Bifrost makes it easy to connect to PLCs, SCADA systems, and other industrial
equipment using modern Python patterns. Get data from your factory floor to
your cloud analytics platform with minimal code.

Quick Start:
    >>> import bifrost
    >>> async with bifrost.connect("modbus://192.168.1.100") as plc:
    ...     data = await plc.read_tags(["temperature", "pressure"])
    ...     print(data)
"""

__version__ = "0.1.0"

# Core exports (always available)
from bifrost_core import (
    BaseConnection,
    BaseProtocol,
    ConnectionError,
    ConnectionPool,
    ConnectionState,
    DataPoint,
    DataType,
    DeviceInfo,
    EventBus,
    ProtocolError,
    ProtocolType,
    Tag,
)

# CLI exports
from .cli import main as cli_main

# Main package exports
from .connections import connect, discover_devices
from .modbus import ModbusConnection, ModbusRTUConnection, ModbusTCPConnection
from .plc import PLCConnection


# Smart imports with helpful error messages
def __getattr__(name: str):
    """Provide helpful error messages for missing optional dependencies."""

    # OPC UA imports
    if name in ("OPCUAClient", "OPCUAServer", "OPCUAConnection"):
        try:
            from bifrost_opcua import OPCUAClient, OPCUAConnection, OPCUAServer

            return locals()[name]
        except ImportError as err:
            raise ImportError(
                f"'{name}' requires OPC UA support. Install with:\n"
                "  uv add bifrost[opcua]\n"
                "Or for everything:\n"
                "  uv add bifrost[all]"
            ) from err

    # Analytics imports
    if name in ("TimeSeriesEngine", "StreamProcessor", "Pipeline", "AnomalyDetector"):
        try:
            from bifrost_analytics import (
                AnomalyDetector,
                Pipeline,
                StreamProcessor,
                TimeSeriesEngine,
            )

            return locals()[name]
        except ImportError as err:
            raise ImportError(
                f"'{name}' requires analytics support. Install with:\n"
                "  uv add bifrost[analytics]\n"
                "Or for everything:\n"
                "  uv add bifrost[all]"
            ) from err

    # Cloud imports
    if name in ("CloudBridge", "AWSConnector", "AzureConnector", "MQTTBridge"):
        try:
            from bifrost_cloud import (
                AWSConnector,
                AzureConnector,
                CloudBridge,
                MQTTBridge,
            )

            return locals()[name]
        except ImportError as err:
            raise ImportError(
                f"'{name}' requires cloud support. Install with:\n"
                "  uv add bifrost[cloud]\n"
                "Or for everything:\n"
                "  uv add bifrost[all]"
            ) from err

    # Protocol imports
    if name in ("EthernetIPConnection", "S7Connection", "DNP3Connection"):
        try:
            from bifrost_protocols import (
                DNP3Connection,
                EthernetIPConnection,
                S7Connection,
            )

            return locals()[name]
        except ImportError as err:
            raise ImportError(
                f"'{name}' requires additional protocols. Install with:\n"
                "  uv add bifrost[protocols]\n"
                "Or for everything:\n"
                "  uv add bifrost[all]"
            ) from err

    raise AttributeError(f"module '{__name__}' has no attribute '{name}'")


__all__ = [
    # Core
    "BaseConnection",
    "BaseProtocol",
    "DataPoint",
    "ConnectionError",
    "ProtocolError",
    "EventBus",
    "ConnectionPool",
    "ProtocolType",
    "DataType",
    "ConnectionState",
    "Tag",
    "DeviceInfo",
    # Main
    "connect",
    "discover_devices",
    "ModbusConnection",
    "ModbusTCPConnection",
    "ModbusRTUConnection",
    "PLCConnection",
    "cli_main",
    # Optional (via __getattr__)
    "OPCUAClient",
    "OPCUAServer",
    "OPCUAConnection",
    "TimeSeriesEngine",
    "StreamProcessor",
    "Pipeline",
    "AnomalyDetector",
    "CloudBridge",
    "AWSConnector",
    "AzureConnector",
    "MQTTBridge",
    "EthernetIPConnection",
    "S7Connection",
    "DNP3Connection",
]
