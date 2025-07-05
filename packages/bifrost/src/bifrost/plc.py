"""Base classes for Programmable Logic Controllers (PLCs)."""

from types import TracebackType

from bifrost_core.base import BaseConnection, BaseDevice
from bifrost_core.typing import Value


class PLCConnection(BaseConnection):
    """Represents a connection to a PLC."""

    def __init__(self, host: str, port: int):
        self.host = host
        self.port = port
        self._is_connected = False

    async def __aenter__(self) -> "PLCConnection":
        # In a real implementation, this would establish a network connection.
        self._is_connected = True
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: TracebackType | None,
    ) -> None:
        # In a real implementation, this would close the network connection.
        self._is_connected = False

    @property
    def is_connected(self) -> bool:
        return self._is_connected


class PLC(BaseDevice[Value]):
    """Represents a generic PLC."""

    def __init__(self, connection: PLCConnection):
        super().__init__(connection)
