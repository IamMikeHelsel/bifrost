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

        raise ValueError(f"Unsupported protocol: {scheme}")
