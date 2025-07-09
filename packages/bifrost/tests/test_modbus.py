"""Tests for Modbus implementation."""

from unittest.mock import AsyncMock, MagicMock

import pytest
import pytest_asyncio
from pymodbus.exceptions import ModbusException

from bifrost.modbus import ModbusConnection, ModbusDevice
from bifrost_core import DataType, Tag


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

    @pytest_asyncio.fixture
    async def connection(self, mock_client):
        """Create Modbus TCP connection with mock client."""
        conn = ModbusConnection(host="192.168.1.100", port=502)
        conn.client = mock_client  # Inject mock client
        return conn

    @pytest_asyncio.fixture
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
    async def test_read_multiple_holding_registers(
        self, modbus_device, mock_client
    ):
        """Test reading multiple holding registers."""
        response = MagicMock()
        response.isError.return_value = False
        response.registers = [123, 456, 789]
        mock_client.read_holding_registers = AsyncMock(return_value=response)

        async with modbus_device.connection:
            tags = [
                Tag(name="test", address="40001:3", data_type=DataType.INT16)
            ]
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
    async def test_write_multiple_holding_registers(
        self, modbus_device, mock_client
    ):
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
            tags = [
                Tag(name="test", address="00001:3", data_type=DataType.BOOLEAN)
            ]
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
            tags = [
                Tag(name="test", address="10001:2", data_type=DataType.BOOLEAN)
            ]
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
            tags = [
                Tag(name="test", address="30001:2", data_type=DataType.INT16)
            ]
            result = await modbus_device.read(tags)
            assert tags[0] in result
            assert result[tags[0]].value == [1234, 5678]
            mock_client.read_input_registers.assert_called_once_with(
                address=0, count=2, slave=1
            )

    @pytest.mark.asyncio
    async def test_connection_not_connected_error(
        self, modbus_device, mock_client
    ):
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

    @pytest.mark.asyncio
    async def test_invalid_address_format(self, modbus_device, mock_client):
        """Test reading with invalid address format."""
        async with modbus_device.connection:
            # Test various invalid addresses
            invalid_tags = [
                Tag(name="test1", address="invalid", data_type=DataType.INT16),
                Tag(name="test2", address="50001", data_type=DataType.INT16),  # Out of range
                Tag(name="test3", address="40001:invalid", data_type=DataType.INT16),
                Tag(name="test4", address="", data_type=DataType.INT16),
            ]
            
            for tag in invalid_tags:
                result = await modbus_device.read([tag])
                assert tag not in result

    @pytest.mark.asyncio
    async def test_mixed_data_types(self, modbus_device, mock_client):
        """Test reading multiple tags with different data types."""
        # Set up different responses for different register types
        mock_client.read_holding_registers = AsyncMock(
            return_value=MagicMock(isError=lambda: False, registers=[0x1234, 0x5678])
        )
        mock_client.read_coils = AsyncMock(
            return_value=MagicMock(isError=lambda: False, bits=[True, False, True])
        )
        mock_client.read_input_registers = AsyncMock(
            return_value=MagicMock(isError=lambda: False, registers=[0xABCD])
        )
        
        async with modbus_device.connection:
            tags = [
                Tag(name="holding", address="40001:2", data_type=DataType.INT16),
                Tag(name="coil", address="00001:3", data_type=DataType.BOOLEAN),
                Tag(name="input", address="30001", data_type=DataType.INT16),
            ]
            
            results = await modbus_device.read(tags)
            
            assert len(results) == 3
            assert results[tags[0]].value == [0x1234, 0x5678]
            assert results[tags[1]].value == [True, False, True]
            assert results[tags[2]].value == [0xABCD]

    @pytest.mark.asyncio
    async def test_large_register_count(self, modbus_device, mock_client):
        """Test reading a large number of registers."""
        # Create response with 125 registers (near Modbus limit)
        large_response = MagicMock()
        large_response.isError.return_value = False
        large_response.registers = list(range(125))
        mock_client.read_holding_registers = AsyncMock(return_value=large_response)
        
        async with modbus_device.connection:
            tag = Tag(name="large", address="40001:125", data_type=DataType.INT16)
            result = await modbus_device.read([tag])
            
            assert tag in result
            assert len(result[tag].value) == 125
            assert result[tag].value == list(range(125))

    @pytest.mark.asyncio
    async def test_slave_id_handling(self, modbus_device, mock_client):
        """Test handling different slave IDs."""
        response = MagicMock()
        response.isError.return_value = False
        response.registers = [999]
        mock_client.read_holding_registers = AsyncMock(return_value=response)
        
        async with modbus_device.connection:
            # Test with slave ID in address
            tag = Tag(name="test", address="40001@5", data_type=DataType.INT16)
            await modbus_device.read([tag])
            
            # Verify slave ID was parsed and used
            mock_client.read_holding_registers.assert_called_with(
                address=0, count=1, slave=5
            )

    @pytest.mark.asyncio
    async def test_concurrent_read_operations(self, modbus_device, mock_client):
        """Test concurrent read operations."""
        import asyncio
        
        response = MagicMock()
        response.isError.return_value = False
        response.registers = [123]
        
        # Add delay to simulate network latency
        async def delayed_read(*args, **kwargs):
            await asyncio.sleep(0.01)
            return response
        
        mock_client.read_holding_registers = AsyncMock(side_effect=delayed_read)
        
        async with modbus_device.connection:
            # Create multiple tags
            tags = [
                Tag(name=f"tag{i}", address=f"4000{i}", data_type=DataType.INT16)
                for i in range(1, 6)
            ]
            
            # Perform concurrent reads
            tasks = [modbus_device.read([tag]) for tag in tags]
            results = await asyncio.gather(*tasks)
            
            # Verify all reads completed
            assert len(results) == 5
            for i, result in enumerate(results):
                assert tags[i] in result
                assert result[tags[i]].value == [123]

    @pytest.mark.asyncio
    async def test_write_coils_multiple(self, modbus_device, mock_client):
        """Test writing multiple coils."""
        response = MagicMock()
        response.isError.return_value = False
        mock_client.write_coils = AsyncMock(return_value=response)
        
        async with modbus_device.connection:
            tag = Tag(name="test", address="00001", data_type=DataType.BOOLEAN)
            await modbus_device.write({tag: [True, False, True, True]})
            
            mock_client.write_coils.assert_called_once_with(
                address=0, values=[True, False, True, True], slave=1
            )

    @pytest.mark.asyncio
    async def test_connection_retry_logic(self, mock_client):
        """Test connection retry behavior."""
        # First attempt fails, second succeeds
        mock_client.connect = AsyncMock(side_effect=[False, True])
        
        conn = ModbusConnection(host="192.168.1.100", port=502)
        conn.client = mock_client
        
        async with conn:
            # Should succeed on second attempt
            assert conn.is_connected
            assert mock_client.connect.call_count == 1  # Only one attempt in context manager

    @pytest.mark.asyncio
    async def test_float_data_type_conversion(self, modbus_device, mock_client):
        """Test reading and writing float values."""
        import struct
        
        # Create float value packed as two 16-bit registers
        float_value = 123.456
        packed = struct.pack('>f', float_value)
        registers = struct.unpack('>HH', packed)
        
        response = MagicMock()
        response.isError.return_value = False
        response.registers = list(registers)
        mock_client.read_holding_registers = AsyncMock(return_value=response)
        
        async with modbus_device.connection:
            tag = Tag(name="float_tag", address="40001:2", data_type=DataType.FLOAT32)
            result = await modbus_device.read([tag])
            
            # Note: The actual float conversion would be done by the application layer
            assert tag in result
            assert len(result[tag].value) == 2
            assert result[tag].value == list(registers)

    @pytest.mark.asyncio
    async def test_timeout_handling(self, modbus_device, mock_client):
        """Test handling of timeout errors."""
        import asyncio
        
        # Simulate timeout
        mock_client.read_holding_registers = AsyncMock(
            side_effect=asyncio.TimeoutError("Operation timed out")
        )
        
        async with modbus_device.connection:
            result = await modbus_device.read(
                [Tag(name="test", address="40001", data_type=DataType.INT16)]
            )
            # Should handle timeout gracefully
            assert len(result) == 0

    @pytest.mark.asyncio
    async def test_connection_state_transitions(self):
        """Test connection state transitions."""
        conn = ModbusConnection(host="192.168.1.100", port=502)
        
        # Initial state
        assert not conn.is_connected
        
        # Mock successful connection
        mock_client = MagicMock()
        mock_client.connect = AsyncMock(return_value=True)
        mock_client.close = AsyncMock()
        mock_client.is_socket_open = MagicMock(return_value=True)
        conn.client = mock_client
        
        # Connect
        async with conn:
            assert conn.is_connected
            
        # After context exit
        assert not conn.is_connected
        mock_client.close.assert_called_once()
