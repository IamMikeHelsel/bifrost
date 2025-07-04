"""Tests for bifrost-core base classes."""

import pytest
from datetime import datetime
from bifrost_core import (
    BaseConnection, DataPoint, DataType, ConnectionState, 
    ConnectionError, ProtocolError
)


class MockConnection(BaseConnection):
    """Mock connection for testing."""
    
    def __init__(self, host: str, port: int = 502, fail_connect: bool = False):
        super().__init__(host, port)
        self.fail_connect = fail_connect
        self._mock_data = {
            "40001": 123,
            "40002": 456.7,
            "coil:1": True,
        }
    
    async def connect(self) -> None:
        if self.fail_connect:
            raise ConnectionError("Mock connection failure")
        self._state = ConnectionState.CONNECTED
    
    async def disconnect(self) -> None:
        self._state = ConnectionState.DISCONNECTED
    
    async def read_raw(self, address: str, count: int = 1):
        if not self.is_connected:
            raise ConnectionError("Not connected")
        
        value = self._mock_data.get(address, 0)
        return [value] * count
    
    async def write_raw(self, address: str, values):
        if not self.is_connected:
            raise ConnectionError("Not connected")
        self._mock_data[address] = values[0]


class TestDataPoint:
    """Test DataPoint class."""
    
    def test_datapoint_creation(self):
        dp = DataPoint(
            address="40001",
            value=123,
            data_type=DataType.INT32,
            timestamp=datetime.now()
        )
        assert dp.address == "40001"
        assert dp.value == 123
        assert dp.data_type == DataType.INT32
    
    def test_datapoint_string_representation(self):
        dp = DataPoint(
            address="40001", 
            value=123,
            data_type=DataType.INT32,
            timestamp=datetime.now()
        )
        assert "DataPoint(40001=123" in str(dp)


class TestBaseConnection:
    """Test BaseConnection abstract class."""
    
    @pytest.mark.asyncio
    async def test_connection_lifecycle(self):
        conn = MockConnection("192.168.1.100", 502)
        
        assert conn.state == ConnectionState.DISCONNECTED
        assert not conn.is_connected
        
        await conn.connect()
        assert conn.state == ConnectionState.CONNECTED
        assert conn.is_connected
        
        await conn.disconnect()
        assert conn.state == ConnectionState.DISCONNECTED
        assert not conn.is_connected
    
    @pytest.mark.asyncio
    async def test_connection_context_manager(self):
        async with MockConnection("192.168.1.100", 502) as conn:
            assert conn.is_connected
        # Connection should be automatically closed
    
    @pytest.mark.asyncio
    async def test_read_single(self):
        async with MockConnection("192.168.1.100", 502) as conn:
            dp = await conn.read_single("40001", DataType.INT32)
            assert dp.address == "40001"
            assert dp.value == 123
            assert dp.data_type == DataType.INT32
    
    @pytest.mark.asyncio
    async def test_read_multiple(self):
        async with MockConnection("192.168.1.100", 502) as conn:
            addresses = ["40001", "40002"]
            data_types = [DataType.INT32, DataType.FLOAT32]
            
            results = await conn.read_multiple(addresses, data_types)
            assert len(results) == 2
            assert results[0].value == 123
            assert results[1].value == 456.7
    
    @pytest.mark.asyncio
    async def test_write_single(self):
        async with MockConnection("192.168.1.100", 502) as conn:
            await conn.write_single("40003", 999, DataType.INT32)
            
            # Verify the write
            dp = await conn.read_single("40003", DataType.INT32)
            assert dp.value == 999
    
    @pytest.mark.asyncio
    async def test_health_check(self):
        async with MockConnection("192.168.1.100", 502) as conn:
            is_healthy = await conn.health_check()
            assert is_healthy
    
    @pytest.mark.asyncio
    async def test_connection_failure(self):
        conn = MockConnection("192.168.1.100", 502, fail_connect=True)
        
        with pytest.raises(ConnectionError):
            await conn.connect()
    
    def test_connection_properties(self):
        conn = MockConnection("192.168.1.100", 502)
        assert conn.host == "192.168.1.100"
        assert conn.port == 502
        assert conn.connection_id == "192.168.1.100:502"
    
    def test_value_conversion(self):
        conn = MockConnection("192.168.1.100", 502)
        
        # Test boolean conversion
        assert conn._convert_value(1, DataType.BOOL) is True
        assert conn._convert_value(0, DataType.BOOL) is False
        
        # Test integer conversion
        assert conn._convert_value(123.7, DataType.INT32) == 123
        
        # Test float conversion
        assert conn._convert_value(123, DataType.FLOAT32) == 123.0
        
        # Test string conversion
        assert conn._convert_value(123, DataType.STRING) == "123"


class TestEnums:
    """Test enum classes."""
    
    def test_connection_state_enum(self):
        assert ConnectionState.DISCONNECTED.value == "disconnected"
        assert ConnectionState.CONNECTED.value == "connected"
    
    def test_data_type_enum(self):
        assert DataType.BOOL.value == "bool"
        assert DataType.INT32.value == "int32"
        assert DataType.FLOAT32.value == "float32"


class TestExceptions:
    """Test custom exceptions."""
    
    def test_connection_error(self):
        error = ConnectionError("Test error")
        assert str(error) == "Test error"
        assert isinstance(error, Exception)
    
    def test_protocol_error(self):
        error = ProtocolError("Protocol error")
        assert str(error) == "Protocol error"
        assert isinstance(error, Exception)