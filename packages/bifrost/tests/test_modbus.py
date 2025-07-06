"""Tests for Modbus implementation."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from pymodbus.exceptions import ModbusException

from bifrost.modbus import ModbusConnection, ModbusDevice
from bifrost_core import ConnectionState, DataType, Tag


class TestModbusConnection:
    """Test Modbus TCP connection."""

    @pytest.fixture
    def mock_client(self):
        """Create mock Modbus client."""
        client = MagicMock()
        client.connect = AsyncMock(return_value=True)
        client.close = AsyncMock()
        client.is_socket_open = MagicMock(return_value=True)
        return client

    @pytest.fixture
    async def connection(self, mock_client):
        """Create Modbus TCP connection with mock client."""
        conn = ModbusConnection(host="192.168.1.100", port=502)
        conn.client = mock_client  # Inject mock client
        return conn
    
    @pytest.fixture
    async def modbus_device(self, connection):
        """Create Modbus device with mock connection."""
        return ModbusDevice(connection)

    @pytest.mark.asyncio
    async def test_connect_success(self, connection, mock_client):
        """Test successful connection."""
        async with connection:
            assert connection.is_connected
            mock_client.connect.assert_called_once()

    @pytest.mark.asyncio
    async def test_connect_failure(self, connection, mock_client):
        """Test connection failure."""
        mock_client.connect.return_value = False

        async with connection:
            assert not connection.is_connected

    @pytest.mark.asyncio
    async def test_disconnect(self, connection, mock_client):
        """Test disconnection."""
        async with connection:
            pass
        assert not connection.is_connected
        mock_client.close.assert_called_once()

    @pytest.mark.asyncio
    async def test_read_holding_register(self, modbus_device, mock_client):
        """Test reading a holding register."""
        response = MagicMock()
        response.isError.return_value = False
        response.registers = [54321]
        mock_client.read_holding_registers = AsyncMock(return_value=response)

        async with modbus_device.connection:
            tags = [Tag(name="test", address="40001", data_type=DataType.INT16)]
            result = await modbus_device.read(tags)
            assert tags[0] in result
            assert result[tags[0]].value == [54321]
            mock_client.read_holding_registers.assert_called_once_with(
                address=0, count=1, slave=1
            )

    @pytest.mark.asyncio
    async def test_read_multiple_holding_registers(self, modbus_device, mock_client):
        """Test reading multiple holding registers."""
        response = MagicMock()
        response.isError.return_value = False
        response.registers = [123, 456, 789]
        mock_client.read_holding_registers = AsyncMock(return_value=response)

        async with modbus_device.connection:
            tags = [Tag(name="test", address="40001:3", data_type=DataType.INT16)]
            result = await modbus_device.read(tags)
            assert tags[0] in result
            assert result[tags[0]].value == [123, 456, 789]
            mock_client.read_holding_registers.assert_called_once_with(
                address=0, count=3, slave=1
            )

    @pytest.mark.asyncio
    async def test_read_error(self, modbus_device, mock_client):
        """Test read error handling."""
        response = MagicMock()
        response.isError.return_value = True
        mock_client.read_holding_registers = AsyncMock(return_value=response)

        async with modbus_device.connection:
            tags = [Tag(name="test", address="40001", data_type=DataType.INT16)]
            result = await modbus_device.read(tags)
            assert len(result) == 0  # Error should result in no reading

    @pytest.mark.asyncio
    async def test_write_holding_register(self, modbus_device, mock_client):
        """Test writing a holding register."""
        response = MagicMock()
        response.isError.return_value = False
        mock_client.write_register = AsyncMock(return_value=response)

        async with modbus_device.connection:
            tag = Tag(name="test", address="40001", data_type=DataType.INT16)
            await modbus_device.write({tag: 12345})
            mock_client.write_register.assert_called_once_with(
                address=0, value=12345, slave=1
            )

    @pytest.mark.asyncio
    async def test_write_multiple_holding_registers(self, modbus_device, mock_client):
        """Test writing multiple holding registers."""
        response = MagicMock()
        response.isError.return_value = False
        mock_client.write_registers = AsyncMock(return_value=response)

        async with modbus_device.connection:
            tag = Tag(name="test", address="40001", data_type=DataType.INT16)
            await modbus_device.write({tag: [123, 456, 789]})
            mock_client.write_registers.assert_called_once_with(
                address=0, values=[123, 456, 789], slave=1
            )

    @pytest.mark.asyncio
    async def test_write_error(self, modbus_device, mock_client):
        """Test write error handling."""
        response = MagicMock()
        response.isError.return_value = True
        mock_client.write_register = AsyncMock(return_value=response)

        async with modbus_device.connection:
            tag = Tag(name="test", address="40001", data_type=DataType.INT16)
            await modbus_device.write({tag: 12345})
            # No exception should be raised, but the write should not have occurred
            mock_client.write_register.assert_called_once()

    @pytest.mark.asyncio
    async def test_read_coils(self, modbus_device, mock_client):
        """Test reading coils."""
        response = MagicMock()
        response.isError.return_value = False
        response.bits = [True, False, True]
        mock_client.read_coils = AsyncMock(return_value=response)

        async with modbus_device.connection:
            tags = [Tag(name="test", address="00001:3", data_type=DataType.BOOLEAN)]
            result = await modbus_device.read(tags)
            assert tags[0] in result
            assert result[tags[0]].value == [True, False, True]
            mock_client.read_coils.assert_called_once_with(
                address=0, count=3, slave=1
            )

    @pytest.mark.asyncio
    async def test_write_coil(self, modbus_device, mock_client):
        """Test writing a coil."""
        response = MagicMock()
        response.isError.return_value = False
        mock_client.write_coil = AsyncMock(return_value=response)

        async with modbus_device.connection:
            tag = Tag(name="test", address="00001", data_type=DataType.BOOLEAN)
            await modbus_device.write({tag: True})
            mock_client.write_coil.assert_called_once_with(
                address=0, value=True, slave=1
            )

    @pytest.mark.asyncio
    async def test_read_discrete_inputs(self, modbus_device, mock_client):
        """Test reading discrete inputs."""
        response = MagicMock()
        response.isError.return_value = False
        response.bits = [True, False]
        mock_client.read_discrete_inputs = AsyncMock(return_value=response)

        async with modbus_device.connection:
            tags = [Tag(name="test", address="10001:2", data_type=DataType.BOOLEAN)]
            result = await modbus_device.read(tags)
            assert tags[0] in result
            assert result[tags[0]].value == [True, False]
            mock_client.read_discrete_inputs.assert_called_once_with(
                address=0, count=2, slave=1
            )

    @pytest.mark.asyncio
    async def test_read_input_registers(self, modbus_device, mock_client):
        """Test reading input registers."""
        response = MagicMock()
        response.isError.return_value = False
        response.registers = [1234, 5678]
        mock_client.read_input_registers = AsyncMock(return_value=response)

        async with modbus_device.connection:
            tags = [Tag(name="test", address="30001:2", data_type=DataType.INT16)]
            result = await modbus_device.read(tags)
            assert tags[0] in result
            assert result[tags[0]].value == [1234, 5678]
            mock_client.read_input_registers.assert_called_once_with(
                address=0, count=2, slave=1
            )

    @pytest.mark.asyncio
    async def test_connection_not_connected_error(self, modbus_device, mock_client):
        """Test operations when not connected."""
        # Without context manager, operations should handle gracefully
        tags = [Tag(name="test", address="40001", data_type=DataType.INT16)]
        
        # Set up mock to simulate not being connected
        mock_client.read_holding_registers = AsyncMock(
            side_effect=Exception("Not connected")
        )
        
        result = await modbus_device.read(tags)
        assert len(result) == 0  # Should return empty result on error
        
        # Test write as well
        mock_client.write_register = AsyncMock(
            side_effect=Exception("Not connected")
        )
        
        tag = Tag(name="test", address="40001", data_type=DataType.INT16)
        await modbus_device.write({tag: 123})  # Should not raise

    @pytest.mark.asyncio
    async def test_modbus_exception_handling(self, modbus_device, mock_client):
        """Test handling of Modbus exceptions."""
        mock_client.read_holding_registers = AsyncMock(
            side_effect=ModbusException("Test exception")
        )

        async with modbus_device.connection:
            result = await modbus_device.read(
                [Tag(name="test", address="40001", data_type=DataType.INT16)]
            )
            assert (
                Tag(name="test", address="40001", data_type=DataType.INT16)
                not in result
            )
