"""Tests for Modbus implementation."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from bifrost.modbus import ModbusConnection
from bifrost_core import ConnectionState
from pymodbus.exceptions import ModbusException


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

        with pytest.raises(ConnectionError, match="Failed to connect"):
            async with connection:
                pass

    @pytest.mark.asyncio
    async def test_disconnect(self, connection, mock_client):
        """Test disconnection."""
        async with connection:
            pass
        assert not connection.is_connected
        mock_client.close.assert_called_once()

    @pytest.mark.asyncio
    async def test_read_holding_register(self, connection, mock_client):
        """Test reading a holding register."""
        response = MagicMock()
        response.isError.return_value = False
        response.registers = [54321]
        mock_client.read_holding_registers = AsyncMock(return_value=response)

        async with connection:
            result = await connection.read(["40001"])
            assert result["40001"].value == 54321
            mock_client.read_holding_registers.assert_called_once_with(address=40001, count=1)

    @pytest.mark.asyncio
    async def test_read_error(self, connection, mock_client):
        """Test read error handling."""
        response = MagicMock()
        response.isError.return_value = True
        mock_client.read_holding_registers = AsyncMock(return_value=response)

        async with connection:
            result = await connection.read(["40001"])
            assert "40001" not in result # Error should result in no reading

    @pytest.mark.asyncio
    async def test_write_holding_register(self, connection, mock_client):
        """Test writing a holding register."""
        response = MagicMock()
        response.isError.return_value = False
        mock_client.write_register = AsyncMock(return_value=response)

        async with connection:
            await connection.write({"40001": 12345})
            mock_client.write_register.assert_called_once_with(address=40001, value=12345)

    @pytest.mark.asyncio
    async def test_write_error(self, connection, mock_client):
        """Test write error handling."""
        response = MagicMock()
        response.isError.return_value = True
        mock_client.write_register = AsyncMock(return_value=response)

        async with connection:
            await connection.write({"40001": 12345})
            # No exception should be raised, but the write should not have occurred
            mock_client.write_register.assert_called_once()

    @pytest.mark.asyncio
    async def test_connection_not_connected_error(self, connection):
        """Test operations when not connected."""
        with pytest.raises(ConnectionError, match="Failed to connect"):
            await connection.read(["40001"])

        with pytest.raises(ConnectionError, match="Failed to connect"):
            await connection.write({"40001": 123})

    @pytest.mark.asyncio
    async def test_modbus_exception_handling(self, connection, mock_client):
        """Test handling of Modbus exceptions."""
        mock_client.read_holding_registers = AsyncMock(
            side_effect=ModbusException("Test exception")
        )

        async with connection:
            result = await connection.read(["40001"])
            assert "40001" not in result