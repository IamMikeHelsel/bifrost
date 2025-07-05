"""Network device discovery for Bifrost."""

from typing import AsyncGenerator

from bifrost_core.base import JsonDict


async def discover_devices() -> AsyncGenerator[JsonDict, None]:
    """Discover devices on the network."""
    # This is a placeholder for network discovery. In a real implementation,
    # this function would use techniques like network scanning (e.g., with Nmap)
    # or custom discovery protocols to find devices.
    # For now, we'll just yield a few dummy devices.
    yield {
        "host": "192.168.1.10",
        "port": 502,
        "protocol": "modbus.tcp",
        "device_type": "PLC",
    }
    yield {
        "host": "192.168.1.20",
        "port": 4840,
        "protocol": "opcua.tcp",
        "device_type": "OPC-UA Server",
    }
