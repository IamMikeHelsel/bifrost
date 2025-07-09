from unittest.mock import AsyncMock, MagicMock

import pytest

from bifrost.modbus import ModbusConnection, ModbusDevice
from bifrost_core import DataType, Tag

# Skip all benchmark tests if pytest-benchmark is not available
pytestmark = pytest.mark.skip(
    reason="pytest-benchmark not properly configured"
)


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
@pytest.mark.slow
async def test_read_single_register_benchmark(modbus_device, benchmark):
    """Benchmark reading a single holding register."""
    response = MagicMock()
    response.isError.return_value = False
    response.registers = [54321]
    modbus_device.connection.client.read_holding_registers = AsyncMock(
        return_value=response
    )

    tag = Tag(name="test", address="40001", data_type=DataType.INT16)

    @benchmark
    async def _():
        await modbus_device.read([tag])


@pytest.mark.asyncio
@pytest.mark.benchmark(group="modbus_read")
@pytest.mark.slow
async def test_read_multiple_registers_benchmark(modbus_device, benchmark):
    """Benchmark reading multiple holding registers."""
    response = MagicMock()
    response.isError.return_value = False
    response.registers = [i for i in range(100)]  # Simulate 100 registers
    modbus_device.connection.client.read_holding_registers = AsyncMock(
        return_value=response
    )

    tag = Tag(name="test", address="40001:100", data_type=DataType.INT16)

    @benchmark
    async def _():
        await modbus_device.read([tag])


@pytest.mark.asyncio
@pytest.mark.benchmark(group="modbus_read")
async def test_read_coils_benchmark(modbus_device, benchmark):
    """Benchmark reading multiple coils."""
    response = MagicMock()
    response.isError.return_value = False
    response.bits = [True for _ in range(100)]  # Simulate 100 coils
    modbus_device.connection.client.read_coils = AsyncMock(
        return_value=response
    )

    tag = Tag(name="test", address="00001:100", data_type=DataType.BOOLEAN)

    @benchmark
    async def _():
        await modbus_device.read([tag])


@pytest.mark.asyncio
@pytest.mark.benchmark(group="modbus_read")
async def test_read_discrete_inputs_benchmark(modbus_device, benchmark):
    """Benchmark reading multiple discrete inputs."""
    response = MagicMock()
    response.isError.return_value = False
    response.bits = [False for _ in range(100)]  # Simulate 100 discrete inputs
    modbus_device.connection.client.read_discrete_inputs = AsyncMock(
        return_value=response
    )

    tag = Tag(name="test", address="10001:100", data_type=DataType.BOOLEAN)

    @benchmark
    async def _():
        await modbus_device.read([tag])


@pytest.mark.asyncio
@pytest.mark.benchmark(group="modbus_read")
async def test_read_input_registers_benchmark(modbus_device, benchmark):
    """Benchmark reading multiple input registers."""
    response = MagicMock()
    response.isError.return_value = False
    response.registers = [i for i in range(100)]  # Simulate 100 input registers
    modbus_device.connection.client.read_input_registers = AsyncMock(
        return_value=response
    )

    tag = Tag(name="test", address="30001:100", data_type=DataType.INT16)

    @benchmark
    async def _():
        await modbus_device.read([tag])


@pytest.mark.asyncio
@pytest.mark.benchmark(group="modbus_write")
async def test_write_single_register_benchmark(modbus_device, benchmark):
    """Benchmark writing a single holding register."""
    response = MagicMock()
    response.isError.return_value = False
    modbus_device.connection.client.write_register = AsyncMock(
        return_value=response
    )

    tag = Tag(name="test", address="40001", data_type=DataType.INT16)

    @benchmark
    async def _():
        await modbus_device.write({tag: 12345})


@pytest.mark.asyncio
@pytest.mark.benchmark(group="modbus_write")
async def test_write_multiple_registers_benchmark(modbus_device, benchmark):
    """Benchmark writing multiple holding registers."""
    response = MagicMock()
    response.isError.return_value = False
    modbus_device.connection.client.write_registers = AsyncMock(
        return_value=response
    )

    tag = Tag(name="test", address="40001", data_type=DataType.INT16)
    values = [i for i in range(100)]

    @benchmark
    async def _():
        await modbus_device.write({tag: values})


@pytest.mark.asyncio
@pytest.mark.benchmark(group="modbus_write")
async def test_write_single_coil_benchmark(modbus_device, benchmark):
    """Benchmark writing a single coil."""
    response = MagicMock()
    response.isError.return_value = False
    modbus_device.connection.client.write_coil = AsyncMock(
        return_value=response
    )

    tag = Tag(name="test", address="00001", data_type=DataType.BOOLEAN)

    @benchmark
    async def _():
        await modbus_device.write({tag: True})


@pytest.mark.asyncio
@pytest.mark.benchmark(group="modbus_write")
async def test_write_multiple_coils_benchmark(modbus_device, benchmark):
    """Benchmark writing multiple coils."""
    response = MagicMock()
    response.isError.return_value = False
    modbus_device.connection.client.write_coils = AsyncMock(
        return_value=response
    )

    tag = Tag(name="test", address="00001", data_type=DataType.BOOLEAN)
    values = [True for _ in range(100)]

    @benchmark
    async def _():
        await modbus_device.write({tag: values})
