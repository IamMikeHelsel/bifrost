"""Connection factory for Bifrost."""

from urllib.parse import urlparse

from bifrost_core.base import BaseConnection

from .modbus import ModbusConnection


class ConnectionFactory:
    """A factory for creating connections to different types of devices."""

    @staticmethod
    def create(url: str) -> BaseConnection:
        """Create a connection from a URL."""
        parsed_url = urlparse(url)
        scheme = parsed_url.scheme
        hostname = parsed_url.hostname
        port = parsed_url.port

        if not hostname:
            raise ValueError("Hostname is required")

        if scheme == "modbus.tcp":
            return ModbusConnection(hostname, port or 502)
        # In the future, we can add support for other protocols here.
        # elif scheme == "opcua.tcp":
        #     return OPCUAConnection(hostname, port or 4840)
        # elif scheme == "s7":
        #     return S7Connection(hostname, port or 102)
        else:
            raise ValueError(f"Unsupported protocol: {scheme}")
