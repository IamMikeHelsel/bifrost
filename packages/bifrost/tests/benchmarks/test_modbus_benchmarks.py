import pytest
from unittest.mock import AsyncMock, MagicMock

from bifrost.modbus import ModbusConnection, ModbusDevice
from bifrost_core import DataType, Tag


@pytest.fixture
def mock_client():
    """Create mock Modbus client."""
    client = MagicMock()
    client.connect = AsyncMock(return_value=True)
    client.close = AsyncMock()
    client.is_socket_open = MagicMock(return_value=True)
    return client


@pytest.fixture
async def modbus_device(mock_client):
    """Create Modbus device with mock connection."""
    conn = ModbusConnection(host="192.168.1.100", port=502)
    conn.client = mock_client  # Inject mock client
    async with conn:
        yield ModbusDevice(conn)


@pytest.mark.asyncio
@pytest.mark.benchmark(group="modbus_read")
async def test_read_single_register_benchmark(modbus_device, benchmark):
    """Benchmark reading a single holding register."""
    response = MagicMock()
    response.isError.return_value = False
    response.registers = [54321]
    modbus_device.connection.client.read_holding_registers = AsyncMock(return_value=response)

    tag = Tag(name="test", address="40001", data_type=DataType.INT16)

    @benchmark
    async def _():
        await modbus_device.read([tag])


@pytest.mark.asyncio
@pytest.mark.benchmark(group="modbus_read")
async def test_read_multiple_registers_benchmark(modbus_device, benchmark):
    """Benchmark reading multiple holding registers."""
    response = MagicMock()
    response.isError.return_value = False
    response.registers = [i for i in range(100)]  # Simulate 100 registers
    modbus_device.connection.client.read_holding_registers = AsyncMock(return_value=response)

    tag = Tag(name="test", address="40001:100", data_type=DataType.INT16)

    @benchmark
    async def _():
        await modbus_device.read([tag])