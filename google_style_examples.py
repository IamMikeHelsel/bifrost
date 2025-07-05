"""Examples of Google-style docstring conversions for Bifrost codebase.

This file demonstrates the required changes to meet Google's docstring standards.
"""

from abc import ABC, abstractmethod
from collections.abc import Sequence
from typing import Any, Optional

from bifrost_core.typing import JsonDict, Tag, Value


# BEFORE: Current style in base.py
class ConnectionState_Before:
    """Represents the state of a connection."""
    pass


# AFTER: Google style
class ConnectionState_After:
    """Represents the state of a connection.
    
    This enum defines the various states that a connection can be in during
    its lifecycle, from initial disconnection through connection establishment
    and eventual disconnection.
    """
    pass


# BEFORE: Current style in base.py
class DeviceInfo_Before:
    """Represents information about a discovered device."""
    pass


# AFTER: Google style
class DeviceInfo_After:
    """Represents information about a discovered device.

    This class contains metadata about devices discovered on the network,
    including connection details and device identification information.

    Attributes:
        device_id: Unique identifier for the device.
        protocol: The communication protocol used (e.g., 'modbus.tcp').
        host: The IP address or hostname of the device.
        port: The port number of the device.
        name: Human-readable name of the device.
        manufacturer: Manufacturer of the device.
        model: Model of the device.
        description: A brief description of the device.
    """
    pass


# BEFORE: Current style in base.py
class BaseConnection_Before(ABC):
    """Abstract base class for a connection to a device or service."""
    
    @abstractmethod
    async def __aenter__(self) -> "BaseConnection":
        """Enter the async context manager."""
        raise NotImplementedError


# AFTER: Google style
class BaseConnection_After(ABC):
    """Abstract base class for a connection to a device or service.

    This class defines the interface for all connection types in Bifrost.
    Implementations should provide async context manager support and
    connection state management.
    """
    
    @abstractmethod
    async def __aenter__(self) -> "BaseConnection":
        """Enter the async context manager.
        
        Establishes the connection to the target device or service.
        
        Returns:
            The connection instance for use in the async context.
            
        Raises:
            ConnectionError: If the connection cannot be established.
        """
        raise NotImplementedError


# BEFORE: Current style in base.py
class BaseDevice_Before(ABC):
    """Abstract base class for a device."""
    
    @abstractmethod
    async def read(self, tags: Sequence[Tag]) -> dict[Tag, Any]:
        """Read one or more values from the device."""
        raise NotImplementedError


# AFTER: Google style
class BaseDevice_After(ABC):
    """Abstract base class for a device.

    This class defines the interface for all device types in Bifrost.
    Implementations should provide methods for reading from and writing to
    the device, as well as retrieving device information.

    Args:
        connection: The connection instance to use for device communication.
    """
    
    @abstractmethod
    async def read(self, tags: Sequence[Tag]) -> dict[Tag, Any]:
        """Read one or more values from the device.
        
        Reads the current values for the specified tags from the device.
        Failed reads are silently ignored and excluded from the result.
        
        Args:
            tags: Sequence of tags to read from the device.
            
        Returns:
            Dictionary mapping tags to their corresponding Reading objects.
            Tags that failed to read are excluded from the result.
            
        Raises:
            ConnectionError: If the device is not connected.
            ValueError: If any tag is invalid or unsupported.
        """
        raise NotImplementedError


# BEFORE: Current style in modbus.py
class ModbusConnection_Before:
    """Represents a connection to a Modbus device."""
    
    def __init__(self, host: str, port: int = 502):
        pass


# AFTER: Google style
class ModbusConnection_After:
    """Represents a connection to a Modbus device.

    This class provides an async context manager for Modbus TCP connections,
    handling connection lifecycle and providing a unified interface for
    Modbus operations.

    Args:
        host: The IP address or hostname of the Modbus device.
        port: The port number for the Modbus TCP connection. Defaults to 502.

    Attributes:
        host: The target host address.
        port: The target port number.
        client: The underlying pymodbus client instance.
    """
    
    def __init__(self, host: str, port: int = 502):
        """Initialize a new Modbus connection.
        
        Args:
            host: The IP address or hostname of the Modbus device.
            port: The port number for the Modbus TCP connection.
        """
        pass


# BEFORE: Current style in events.py
class EventBus_Before:
    """A simple event bus for dispatching events to listeners."""
    
    async def on(self, event_type, handler):
        """Register an event handler for a given event type."""
        pass


# AFTER: Google style
class EventBus_After:
    """A simple event bus for dispatching events to listeners.

    This class provides a thread-safe, async-first event system for
    decoupling components in Bifrost. Events are dispatched to all
    registered handlers asynchronously.

    The event bus supports type-safe event handling through generic
    type parameters and maintains handler registration state using
    internal locks for thread safety.
    """
    
    async def on(self, event_type, handler):
        """Register an event handler for a given event type.
        
        Registers a coroutine function to be called when events of the
        specified type are emitted. Multiple handlers can be registered
        for the same event type.
        
        Args:
            event_type: The type of event to listen for.
            handler: A coroutine function that will handle the event.
            
        Raises:
            TypeError: If the handler is not a coroutine function.
        """
        pass


# Function docstring examples
def discover_devices_before() -> None:
    """Discover devices on the network."""
    pass


def discover_devices_after() -> None:
    """Discover devices on the network.
    
    Scans the local network for industrial devices using various protocols
    and returns information about discovered devices. This is a blocking
    operation that may take several seconds to complete.
    
    Returns:
        None. Results are printed to the console using Rich formatting.
        
    Raises:
        NetworkError: If network scanning fails.
        TimeoutError: If the scan takes longer than expected.
    """
    pass