"""Tests for bifrost-core base classes."""

from collections.abc import Sequence

import pytest

from bifrost_core.base import (
    BaseConnection,
    BaseDevice,
    ConnectionState,
    Reading,
)
from bifrost_core.typing import DataType, JsonDict, Tag, Timestamp, Value


class MockConnection(BaseConnection):
    """Mock connection for testing."""

    def __init__(self, host: str, port: int = 502, fail_connect: bool = False):
        self.host = host
        self.port = port
        self.fail_connect = fail_connect
        self._mock_data: dict[Tag, Value] = {
            Tag(name="reg1", address="40001", data_type=DataType.INT16): 123,
            Tag(
                name="reg2", address="40002", data_type=DataType.FLOAT32
            ): 456.7,
            Tag(
                name="coil1", address="coil:1", data_type=DataType.BOOLEAN
            ): True,
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

    async def read(self, tags: Sequence[Tag]) -> dict[Tag, Reading[Value]]:
        if not self.is_connected:
            raise ConnectionError("Not connected")
        readings = {}
        for tag in tags:
            value = self._mock_data.get(tag)
            if value is not None:
                readings[tag] = Reading(
                    tag=tag, value=value, timestamp=Timestamp(0)
                )  # Mock timestamp
        return readings

    async def write(self, values: dict[Tag, Value]) -> None:
        if not self.is_connected:
            raise ConnectionError("Not connected")
        self._mock_data.update(values)

    async def get_info(self) -> JsonDict:
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
            tag1 = Tag(name="reg1", address="40001", data_type=DataType.INT16)
            tag2 = Tag(name="reg2", address="40002", data_type=DataType.FLOAT32)

            readings = await conn.read([tag1, tag2])
            assert readings[tag1].value == 123
            assert readings[tag2].value == 456.7

            await conn.write({tag1: 999})
            readings = await conn.read([tag1])
            assert readings[tag1].value == 999

    @pytest.mark.asyncio
    async def test_connection_failure(self):
        conn = MockConnection("192.168.1.100", 502, fail_connect=True)
        with pytest.raises(ConnectionError):
            async with conn:
                pass

    @pytest.mark.asyncio
    async def test_get_info(self):
        async with MockConnection("192.168.1.100", 502) as conn:
            info = await conn.get_info()
            assert info["host"] == "192.168.1.100"
            assert info["port"] == 502


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


class TestDeviceInfo:
    """Test DeviceInfo model."""

    def test_default_name(self):
        from bifrost_core.base import DeviceInfo

        device = DeviceInfo(
            device_id="test_id",
            protocol="test_protocol",
            host="localhost",
            port=1234,
            discovery_method="test_method",
        )
        assert device.name == "test_id"

    def test_explicit_name(self):
        from bifrost_core.base import DeviceInfo

        device = DeviceInfo(
            device_id="test_id",
            protocol="test_protocol",
            host="localhost",
            port=1234,
            discovery_method="test_method",
            name="My Device",
        )
        assert device.name == "My Device"


class MockDevice(BaseDevice[Value]):
    """Mock device for testing BaseDevice."""

    def __init__(self, connection: BaseConnection):
        super().__init__(connection)
        self._mock_readings: dict[Tag, Reading[Value]] = {}

    async def read(self, tags: Sequence[Tag]) -> dict[Tag, Reading[Value]]:
        return {
            tag: self._mock_readings.get(tag)
            for tag in tags
            if tag in self._mock_readings
        }

    async def write(self, values: dict[Tag, Value]) -> None:
        for tag, value in values.items():
            self._mock_readings[tag] = Reading(
                tag=tag, value=value, timestamp=Timestamp(0)
            )

    async def get_info(self) -> JsonDict:
        return {"device_type": "Mock Device"}


class TestBaseDevice:
    """Test BaseDevice abstract class."""

    @pytest.mark.asyncio
    async def test_read_write_info(self):
        conn = MockConnection("127.0.0.1")
        device = MockDevice(conn)

        tag1 = Tag(name="test_tag", address="1", data_type=DataType.INT16)
        tag2 = Tag(name="another_tag", address="2", data_type=DataType.STRING)

        # Test write
        await device.write({tag1: 100, tag2: "hello"})

        # Test read
        readings = await device.read([tag1, tag2])
        assert readings[tag1].value == 100
        assert readings[tag2].value == "hello"

        # Test get_info
        info = await device.get_info()
        assert info["device_type"] == "Mock Device"
