"""Tests for Modbus implementation."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from bifrost.modbus import ModbusTCPConnection
from bifrost_core import ConnectionState, ProtocolError
from pymodbus.exceptions import ModbusException


class TestParseModbusAddress:
    """Test Modbus address parsing through ModbusConnection._parse_address."""

    @pytest.fixture
    def connection(self):
        """Create a ModbusTCPConnection for testing address parsing."""
        return ModbusTCPConnection("192.168.1.100", 502)

    def test_parse_explicit_coil(self, connection):
        """Test parsing explicit coil address."""
        reg_type, address = connection._parse_address("coil:100")
        assert reg_type == "coil"
        assert address == 100

    def test_parse_explicit_discrete_input(self, connection):
        """Test parsing explicit discrete input address."""
        reg_type, address = connection._parse_address("discrete:200")
        assert reg_type == "discrete"
        assert address == 200

    def test_parse_explicit_input_register(self, connection):
        """Test parsing explicit input register address."""
        reg_type, address = connection._parse_address("input:300")
        assert reg_type == "input"
        assert address == 300

    def test_parse_explicit_holding_register(self, connection):
        """Test parsing explicit holding register address."""
        reg_type, address = connection._parse_address("holding:400")
        assert reg_type == "holding"
        assert address == 400

    def test_parse_conventional_coil(self, connection):
        """Test parsing conventional coil address (0xxxx)."""
        reg_type, address = connection._parse_address("100")
        assert reg_type == "coil"
        assert address == 100

    def test_parse_conventional_discrete_input(self, connection):
        """Test parsing conventional discrete input address (1xxxx)."""
        reg_type, address = connection._parse_address("10200")
        assert reg_type == "discrete"
        assert address == 200  # 0-based

    def test_parse_conventional_input_register(self, connection):
        """Test parsing conventional input register address (3xxxx)."""
        reg_type, address = connection._parse_address("30300")
        assert reg_type == "input"
        assert address == 300  # 0-based

    def test_parse_conventional_holding_register(self, connection):
        """Test parsing conventional holding register address (4xxxx)."""
        reg_type, address = connection._parse_address("40400")
        assert reg_type == "holding"
        assert address == 400  # 0-based

    def test_parse_default_holding_register(self, connection):
        """Test parsing plain number defaults to holding register."""
        reg_type, address = connection._parse_address("50000")
        assert reg_type == "holding"
        assert address == 50000


class TestModbusTCPConnection:
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
    def connection(self, mock_client):
        """Create Modbus TCP connection with mock client."""
        conn = ModbusTCPConnection(
            host="192.168.1.100", port=502, unit_id=1, timeout=3.0
        )
        # Don't create actual client, use our mock
        return conn

    @pytest.mark.asyncio
    async def test_connect_success(self, connection, mock_client):
        """Test successful connection."""
        with patch("bifrost.modbus.AsyncModbusTcpClient", return_value=mock_client):
            await connection.connect()

        assert connection.is_connected
        mock_client.connect.assert_called_once()

    @pytest.mark.asyncio
    async def test_connect_failure(self, connection, mock_client):
        """Test connection failure."""
        mock_client.connect.return_value = False

        from bifrost_core import ConnectionError as BifrostConnectionError

        with pytest.raises(BifrostConnectionError, match="Failed to connect"):
            with patch("bifrost.modbus.AsyncModbusTcpClient", return_value=mock_client):
                await connection.connect()

    @pytest.mark.asyncio
    async def test_disconnect(self, connection, mock_client):
        """Test disconnection."""
        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client

        await connection.disconnect()

        assert not connection.is_connected
        mock_client.close.assert_called_once()

    @pytest.mark.asyncio
    async def test_read_coil(self, connection, mock_client):
        """Test reading a coil."""
        response = MagicMock()
        response.isError.return_value = False
        response.bits = [True]
        mock_client.read_coils = AsyncMock(return_value=response)

        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        connection._client = mock_client
        result = await connection.read_raw("coil:100")

        assert result == [True]
        mock_client.read_coils.assert_called_once_with(100, 1, slave=1)

    @pytest.mark.asyncio
    async def test_read_discrete_input(self, connection, mock_client):
        """Test reading a discrete input."""
        response = MagicMock()
        response.isError.return_value = False
        response.bits = [False]
        mock_client.read_discrete_inputs = AsyncMock(return_value=response)

        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        result = await connection.read_raw("discrete:200")

        assert result == [False]
        mock_client.read_discrete_inputs.assert_called_once_with(200, 1, slave=1)

    @pytest.mark.asyncio
    async def test_read_input_register(self, connection, mock_client):
        """Test reading an input register."""
        response = MagicMock()
        response.isError.return_value = False
        response.registers = [12345]
        mock_client.read_input_registers = AsyncMock(return_value=response)

        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        result = await connection.read_raw("input:300")

        assert result == [12345]
        mock_client.read_input_registers.assert_called_once_with(300, 1, slave=1)

    @pytest.mark.asyncio
    async def test_read_holding_register(self, connection, mock_client):
        """Test reading a holding register."""
        response = MagicMock()
        response.isError.return_value = False
        response.registers = [54321]
        mock_client.read_holding_registers = AsyncMock(return_value=response)

        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        result = await connection.read_raw("holding:400")

        assert result == [54321]
        mock_client.read_holding_registers.assert_called_once_with(400, 1, slave=1)

    @pytest.mark.asyncio
    async def test_read_error(self, connection, mock_client):
        """Test read error handling."""
        response = MagicMock()
        response.isError.return_value = True
        mock_client.read_holding_registers = AsyncMock(return_value=response)

        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client

        with pytest.raises(ProtocolError, match="Modbus read error"):
            await connection.read_raw("holding:400")

    @pytest.mark.asyncio
    async def test_write_coil(self, connection, mock_client):
        """Test writing a coil."""
        response = MagicMock()
        response.isError.return_value = False
        mock_client.write_coil = AsyncMock(return_value=response)

        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        await connection.write_raw("coil:100", [True])

        mock_client.write_coil.assert_called_once_with(100, True, slave=1)

    @pytest.mark.asyncio
    async def test_write_holding_register(self, connection, mock_client):
        """Test writing a holding register."""
        response = MagicMock()
        response.isError.return_value = False
        mock_client.write_register = AsyncMock(return_value=response)

        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        await connection.write_raw("holding:400", [12345])

        mock_client.write_register.assert_called_once_with(400, 12345, slave=1)

    @pytest.mark.asyncio
    async def test_write_read_only_register(self, connection, mock_client):
        """Test writing to read-only register types."""
        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client

        with pytest.raises(ProtocolError, match="Cannot write to discrete"):
            await connection.write_raw("discrete:200", [True])

        with pytest.raises(ProtocolError, match="Cannot write to input"):
            await connection.write_raw("input:300", [100])

    @pytest.mark.asyncio
    async def test_health_check_connected(self, connection, mock_client):
        """Test health check when connected."""
        response = MagicMock()
        response.isError.return_value = False
        response.bits = [True]
        mock_client.read_coils = AsyncMock(return_value=response)
        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client

        result = await connection.health_check()
        assert result is True

    @pytest.mark.asyncio
    async def test_health_check_disconnected(self, connection, mock_client):
        """Test health check when disconnected."""
        mock_client.is_socket_open.return_value = False
        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client

        result = await connection.health_check()
        assert result is False

    @pytest.mark.asyncio
    async def test_context_manager(self, mock_client):
        """Test using connection as context manager."""
        with patch("bifrost.modbus.AsyncModbusTcpClient", return_value=mock_client):
            async with ModbusTCPConnection("192.168.1.100", 502) as conn:
                assert conn.is_connected
                mock_client.connect.assert_called_once()

            mock_client.close.assert_called_once()

    @pytest.mark.asyncio
    async def test_connection_not_connected_error(self, connection):
        """Test operations when not connected."""
        connection._state = ConnectionState.DISCONNECTED

        from bifrost_core import ConnectionError as BifrostConnectionError

        with pytest.raises(BifrostConnectionError, match="Not connected"):
            await connection.read_raw("holding:100")

        with pytest.raises(BifrostConnectionError, match="Not connected"):
            await connection.write_raw("coil:100", [True])

    @pytest.mark.asyncio
    async def test_modbus_exception_handling(self, connection, mock_client):
        """Test handling of Modbus exceptions."""
        mock_client.read_holding_registers = AsyncMock(
            side_effect=ModbusException("Test exception")
        )

        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client

        with pytest.raises(ProtocolError, match="Modbus read error"):
            await connection.read_raw("holding:100")

    def test_connection_string_representation(self, connection):
        """Test string representation of connection."""
        expected = "ModbusTCPConnection(host='192.168.1.100', port=502, unit_id=1)"
        assert str(connection) == expected

    def test_connection_properties(self, connection):
        """Test connection properties."""
        assert connection.protocol == "modbus"
        assert connection.connection_string == "modbus://192.168.1.100:502/1"

    @pytest.mark.asyncio
    async def test_batch_read_success(self, connection, mock_client):
        """Test successful batch read operation."""
        # Mock responses for different register types
        mock_holding_response = MagicMock()
        mock_holding_response.isError.return_value = False
        mock_holding_response.registers = [100, 200, 300]
        
        mock_coil_response = MagicMock()
        mock_coil_response.isError.return_value = False
        mock_coil_response.bits = [True, False, True]
        
        mock_client.read_holding_registers = AsyncMock(return_value=mock_holding_response)
        mock_client.read_coils = AsyncMock(return_value=mock_coil_response)
        
        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        
        # Test batch read with mixed register types
        addresses = ["40001", "40002", "40003", "coil:1", "coil:2", "coil:3"]
        results = await connection.read_batch(addresses)
        
        assert len(results) == 6
        assert results["40001"] == [100]
        assert results["40002"] == [200]
        assert results["40003"] == [300]
        assert results["coil:1"] == [True]
        assert results["coil:2"] == [False]
        assert results["coil:3"] == [True]
        
        # Verify client calls
        mock_client.read_holding_registers.assert_called()
        mock_client.read_coils.assert_called()

    @pytest.mark.asyncio
    async def test_batch_read_range_optimization(self, connection, mock_client):
        """Test that batch read optimizes address ranges."""
        mock_response = MagicMock()
        mock_response.isError.return_value = False
        mock_response.registers = [10, 20, 30, 40, 50]  # 5 consecutive registers
        mock_client.read_holding_registers = AsyncMock(return_value=mock_response)
        
        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        
        # Read 5 consecutive addresses
        addresses = ["40001", "40002", "40003", "40004", "40005"]
        results = await connection.read_batch(addresses)
        
        assert len(results) == 5
        assert results["40001"] == [10]
        assert results["40005"] == [50]
        
        # Should have made only one read call for the range
        mock_client.read_holding_registers.assert_called_once_with(1, 5, slave=1)

    @pytest.mark.asyncio
    async def test_batch_read_not_connected(self, connection):
        """Test batch read when not connected."""
        connection._state = ConnectionState.DISCONNECTED
        
        from bifrost_core import ConnectionError as BifrostConnectionError
        
        with pytest.raises(BifrostConnectionError, match="Not connected"):
            await connection.read_batch(["40001", "40002"])

    @pytest.mark.asyncio
    async def test_batch_write_success(self, connection, mock_client):
        """Test successful batch write operation."""
        mock_response = MagicMock()
        mock_response.isError.return_value = False
        mock_client.write_registers = AsyncMock(return_value=mock_response)
        mock_client.write_coils = AsyncMock(return_value=mock_response)
        
        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        
        # Test batch write with mixed register types
        address_values = {
            "40001": 100,
            "40002": 200,
            "40003": 300,
            "coil:1": True,
            "coil:2": False
        }
        
        await connection.write_batch(address_values)
        
        # Verify client calls were made
        mock_client.write_registers.assert_called()
        mock_client.write_coils.assert_called()

    @pytest.mark.asyncio
    async def test_batch_write_range_optimization(self, connection, mock_client):
        """Test that batch write optimizes consecutive addresses."""
        mock_response = MagicMock()
        mock_response.isError.return_value = False
        mock_client.write_registers = AsyncMock(return_value=mock_response)
        
        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        
        # Write to consecutive addresses
        address_values = {
            "40001": 100,
            "40002": 200,
            "40003": 300
        }
        
        await connection.write_batch(address_values)
        
        # Should have made one write call for the consecutive range
        mock_client.write_registers.assert_called_once_with(1, [100, 200, 300], slave=1)

    @pytest.mark.asyncio
    async def test_batch_write_read_only_registers(self, connection, mock_client):
        """Test batch write to read-only registers raises error."""
        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        
        with pytest.raises(ProtocolError, match="Cannot write to discrete registers"):
            await connection.write_batch({"10001": 100})  # Discrete input

    @pytest.mark.asyncio
    async def test_batch_write_not_connected(self, connection):
        """Test batch write when not connected."""
        connection._state = ConnectionState.DISCONNECTED
        
        from bifrost_core import ConnectionError as BifrostConnectionError
        
        with pytest.raises(BifrostConnectionError, match="Not connected"):
            await connection.write_batch({"40001": 100})

    @pytest.mark.asyncio
    async def test_batch_operations_error_handling(self, connection, mock_client):
        """Test error handling in batch operations."""
        # Mock error response
        mock_error_response = MagicMock()
        mock_error_response.isError.return_value = True
        mock_client.read_holding_registers = AsyncMock(return_value=mock_error_response)
        
        connection._state = ConnectionState.CONNECTED
        connection._client = mock_client
        
        with pytest.raises(ProtocolError, match="Modbus read error"):
            await connection.read_batch(["40001", "40002"])

    def test_group_addresses_by_type(self, connection):
        """Test address grouping by register type."""
        addresses = ["40001", "40002", "coil:1", "10001", "30001"]
        groups = connection._group_addresses_by_type(addresses)
        
        assert len(groups["holding"]) == 2  # 40001, 40002
        assert len(groups["coil"]) == 1     # coil:1
        assert len(groups["discrete"]) == 1 # 10001
        assert len(groups["input"]) == 1    # 30001

    def test_optimize_address_ranges(self, connection):
        """Test address range optimization."""
        # Test consecutive addresses
        address_list = [("40001", 0), ("40002", 1), ("40003", 2)]
        ranges = connection._optimize_address_ranges(address_list)
        
        assert len(ranges) == 1
        start_addr, end_addr, addresses = ranges[0]
        assert start_addr == 0
        assert end_addr == 2
        assert len(addresses) == 3
        
        # Test non-consecutive addresses
        address_list = [("40001", 0), ("40010", 9)]  # Gap of 9
        ranges = connection._optimize_address_ranges(address_list)
        
        assert len(ranges) == 2  # Should split into separate ranges

    def test_optimize_write_ranges(self, connection):
        """Test write range optimization."""
        # Test consecutive writes
        writes = [(0, 100, "40001"), (1, 200, "40002"), (2, 300, "40003")]
        ranges = connection._optimize_write_ranges(writes)
        
        assert len(ranges) == 1
        start_addr, values, addresses = ranges[0]
        assert start_addr == 0
        assert values == [100, 200, 300]
        assert len(addresses) == 3
        
        # Test non-consecutive writes
        writes = [(0, 100, "40001"), (10, 200, "40011")]  # Gap
        ranges = connection._optimize_write_ranges(writes)
        
        assert len(ranges) == 2  # Should split into separate ranges
