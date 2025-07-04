"""Connection utilities and factory functions."""

from typing import Any, Dict, List
from bifrost_core import BaseConnection, DeviceInfo, ProtocolType


async def connect(connection_string: str, **kwargs: Any) -> BaseConnection:
    """Connect to a device using a connection string.
    
    Args:
        connection_string: Protocol connection string (e.g., "modbus://192.168.1.100")
        **kwargs: Additional connection parameters
        
    Returns:
        Connected device instance
        
    Raises:
        ValueError: If protocol is not supported
        ConnectionError: If connection fails
    """
    # Parse protocol from connection string
    if "://" not in connection_string:
        raise ValueError(f"Invalid connection string: {connection_string}")
    
    protocol_name, _ = connection_string.split("://", 1)
    
    # Map protocol names to implementations
    if protocol_name.lower() in ("modbus", "modbus_tcp"):
        from .modbus import ModbusTCPConnection
        return ModbusTCPConnection.from_url(connection_string, **kwargs)
    
    elif protocol_name.lower() == "opcua":
        try:
            from bifrost_opcua import OPCUAConnection
            return OPCUAConnection.from_url(connection_string, **kwargs)
        except ImportError:
            raise ImportError(
                "OPC UA support requires: pip install bifrost[opcua]\n"
                "Or: pip install bifrost-opcua"
            )
    
    else:
        raise ValueError(f"Unsupported protocol: {protocol_name}")


async def discover_devices(
    network: str = "192.168.1.0/24",
    protocols: List[str] = None,
    timeout: float = 5.0
) -> List[DeviceInfo]:
    """Discover devices on the network.
    
    Args:
        network: Network to scan (CIDR notation)
        protocols: List of protocols to scan for (default: all)
        timeout: Timeout per device in seconds
        
    Returns:
        List of discovered devices
    """
    if protocols is None:
        protocols = ["modbus"]  # Start with just Modbus for now
    
    discovered = []
    
    # For now, return empty list - will implement actual discovery later
    # TODO: Implement network scanning for different protocols
    
    return discovered