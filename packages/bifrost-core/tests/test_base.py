"""Tests for bifrost-core base classes."""

import pytest
from bifrost_core.base import BaseConnection, ConnectionState
from bifrost_core.typing import DataType


class MockConnection(BaseConnection):
    """Mock connection for testing."""

    def __init__(self, host: str, port: int = 502, fail_connect: bool = False):
        self.host = host
        self.port = port
        self.fail_connect = fail_connect
        self._mock_data = {
            "40001": 123,
            "40002": 456.7,
            "coil:1": True,
        }
        self._state = ConnectionState.DISCONNECTED

    async def __aenter__(self) -> "MockConnection":
        if self.fail_connect:
            raise ConnectionError("Mock connection failure")
        self._state = ConnectionState.CONNECTED
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        self._state = ConnectionState.DISCONNECTED

    @property
    def is_connected(self) -> bool:
        return self._state == ConnectionState.CONNECTED

    async def read(self, tags):
        if not self.is_connected:
            raise ConnectionError("Not connected")
        return {tag: self._mock_data.get(tag, 0) for tag in tags}

    async def write(self, values):
        if not self.is_connected:
            raise ConnectionError("Not connected")
        self._mock_data.update(values)

    async def get_info(self):
        return {"host": self.host, "port": self.port}


class TestBaseConnection:
    """Test BaseConnection abstract class."""

    @pytest.mark.asyncio
    async def test_connection_lifecycle(self):
        conn = MockConnection("192.168.1.100", 502)

        assert conn.is_connected is False

        async with conn as c:
            assert c.is_connected is True

        assert conn.is_connected is False

    @pytest.mark.asyncio
    async def test_read_write(self):
        async with MockConnection("192.168.1.100", 502) as conn:
            readings = await conn.read(["40001"])
            assert readings["40001"] == 123

            await conn.write({"40001": 999})
            readings = await conn.read(["40001"])
            assert readings["40001"] == 999

    @pytest.mark.asyncio
    async def test_connection_failure(self):
        conn = MockConnection("192.168.1.100", 502, fail_connect=True)
        with pytest.raises(ConnectionError):
            async with conn:
                pass


class TestEnums:
    """Test enum classes."""

    def test_connection_state_enum(self):
        assert ConnectionState.DISCONNECTED.value == "disconnected"
        assert ConnectionState.CONNECTED.value == "connected"

    def test_data_type_enum(self):
        assert DataType.INT16.value == "int16"
        assert DataType.UINT16.value == "uint16"
        assert DataType.INT32.value == "int32"
        assert DataType.UINT32.value == "uint32"
        assert DataType.FLOAT32.value == "float32"
        assert DataType.FLOAT64.value == "float64"
        assert DataType.BOOLEAN.value == "boolean"
        assert DataType.STRING.value == "string"
        assert DataType.BYTE.value == "byte"
