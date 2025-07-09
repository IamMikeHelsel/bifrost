"""Performance benchmark tests for Bifrost."""

import asyncio
import time
from unittest.mock import AsyncMock, MagicMock

import pytest

from bifrost.modbus import ModbusConnection, ModbusDevice
from bifrost_core import DataType, Tag
from bifrost_core.base import BaseConnection, Reading
from bifrost_core.pooling import ConnectionPool
from bifrost_core.typing import Timestamp


class MockHighPerformanceConnection(BaseConnection):
    """Mock connection optimized for performance testing."""

    def __init__(self, host: str, port: int = 502):
        self.host = host
        self.port = port
        self._connected = False
        # Pre-generate mock data for performance
        self._mock_data = {
            f"tag_{i}": i * 100 for i in range(1000)
        }

    async def __aenter__(self):
        self._connected = True
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        self._connected = False

    @property
    def is_connected(self) -> bool:
        return self._connected

    async def read(self, tags):
        # Simulate minimal network delay
        await asyncio.sleep(0.0001)
        return {
            tag: Reading(
                tag=tag,
                value=self._mock_data.get(tag.name, 0),
                timestamp=Timestamp(time.time())
            )
            for tag in tags
        }

    async def write(self, values):
        # Simulate minimal network delay
        await asyncio.sleep(0.0001)
        for tag, value in values.items():
            self._mock_data[tag.name] = value

    async def get_info(self):
        return {"host": self.host, "port": self.port}


@pytest.mark.benchmark
class TestPerformanceBenchmarks:
    """Performance benchmark tests."""

    @pytest.mark.asyncio
    async def test_single_tag_read_performance(self):
        """Benchmark single tag read performance."""
        conn = MockHighPerformanceConnection("192.168.1.100")
        
        async with conn:
            tag = Tag(name="tag_1", address="40001", data_type=DataType.INT16)
            
            # Warm up
            for _ in range(10):
                await conn.read([tag])
            
            # Benchmark
            start_time = time.time()
            iterations = 1000
            
            for _ in range(iterations):
                result = await conn.read([tag])
                assert tag in result
            
            elapsed_time = time.time() - start_time
            ops_per_second = iterations / elapsed_time
            
            print(f"\nSingle tag read: {ops_per_second:.0f} ops/sec")
            assert ops_per_second > 5000  # Should achieve > 5000 ops/sec

    @pytest.mark.asyncio
    async def test_bulk_tag_read_performance(self):
        """Benchmark bulk tag read performance."""
        conn = MockHighPerformanceConnection("192.168.1.100")
        
        async with conn:
            # Create 100 tags
            tags = [
                Tag(name=f"tag_{i}", address=f"4000{i}", data_type=DataType.INT16)
                for i in range(100)
            ]
            
            # Warm up
            for _ in range(10):
                await conn.read(tags)
            
            # Benchmark
            start_time = time.time()
            iterations = 100
            
            for _ in range(iterations):
                result = await conn.read(tags)
                assert len(result) == 100
            
            elapsed_time = time.time() - start_time
            ops_per_second = iterations / elapsed_time
            tags_per_second = (iterations * 100) / elapsed_time
            
            print(f"\nBulk read (100 tags): {ops_per_second:.0f} ops/sec, {tags_per_second:.0f} tags/sec")
            assert tags_per_second > 50000  # Should achieve > 50k tags/sec

    @pytest.mark.asyncio
    async def test_concurrent_read_performance(self):
        """Benchmark concurrent read performance."""
        conn = MockHighPerformanceConnection("192.168.1.100")
        
        async with conn:
            # Create tags for different concurrent operations
            tag_groups = [
                [Tag(name=f"tag_{i}_{j}", address=f"400{i}{j}", data_type=DataType.INT16)
                 for j in range(10)]
                for i in range(10)
            ]
            
            # Benchmark concurrent reads
            start_time = time.time()
            iterations = 100
            
            for _ in range(iterations):
                tasks = [conn.read(tags) for tags in tag_groups]
                results = await asyncio.gather(*tasks)
                assert len(results) == 10
                assert all(len(r) == 10 for r in results)
            
            elapsed_time = time.time() - start_time
            total_tags = iterations * 10 * 10
            tags_per_second = total_tags / elapsed_time
            
            print(f"\nConcurrent read (10x10 tags): {tags_per_second:.0f} tags/sec")
            assert tags_per_second > 100000  # Should achieve > 100k tags/sec

    @pytest.mark.asyncio
    async def test_connection_pool_performance(self):
        """Benchmark connection pool performance."""
        async def factory():
            conn = MockHighPerformanceConnection("192.168.1.100")
            await conn.__aenter__()
            return conn
        
        pool = ConnectionPool(connection_factory=factory, max_size=5)
        
        tag = Tag(name="test_tag", address="40001", data_type=DataType.INT16)
        
        # Warm up
        for _ in range(10):
            conn = await pool.get()
            await conn.read([tag])
            await pool.put(conn)
        
        # Benchmark
        start_time = time.time()
        iterations = 1000
        
        for _ in range(iterations):
            conn = await pool.get()
            await conn.read([tag])
            await pool.put(conn)
        
        elapsed_time = time.time() - start_time
        ops_per_second = iterations / elapsed_time
        
        print(f"\nConnection pool operations: {ops_per_second:.0f} ops/sec")
        assert ops_per_second > 5000  # Should achieve > 5000 ops/sec
        
        await pool.close()

    @pytest.mark.asyncio
    async def test_write_performance(self):
        """Benchmark write performance."""
        conn = MockHighPerformanceConnection("192.168.1.100")
        
        async with conn:
            tag = Tag(name="write_tag", address="40001", data_type=DataType.INT16)
            
            # Warm up
            for i in range(10):
                await conn.write({tag: i})
            
            # Benchmark
            start_time = time.time()
            iterations = 1000
            
            for i in range(iterations):
                await conn.write({tag: i})
            
            elapsed_time = time.time() - start_time
            ops_per_second = iterations / elapsed_time
            
            print(f"\nSingle tag write: {ops_per_second:.0f} ops/sec")
            assert ops_per_second > 5000  # Should achieve > 5000 ops/sec

    @pytest.mark.asyncio
    async def test_mixed_read_write_performance(self):
        """Benchmark mixed read/write performance."""
        conn = MockHighPerformanceConnection("192.168.1.100")
        
        async with conn:
            read_tags = [
                Tag(name=f"read_tag_{i}", address=f"4000{i}", data_type=DataType.INT16)
                for i in range(10)
            ]
            write_tag = Tag(name="write_tag", address="40100", data_type=DataType.INT16)
            
            # Benchmark mixed operations
            start_time = time.time()
            iterations = 500
            
            for i in range(iterations):
                # Read 10 tags
                read_result = await conn.read(read_tags)
                assert len(read_result) == 10
                
                # Write 1 tag
                await conn.write({write_tag: i})
            
            elapsed_time = time.time() - start_time
            total_ops = iterations * 2  # Read + write operations
            ops_per_second = total_ops / elapsed_time
            
            print(f"\nMixed read/write: {ops_per_second:.0f} ops/sec")
            assert ops_per_second > 5000  # Should achieve > 5000 ops/sec

    @pytest.mark.asyncio
    async def test_tag_parsing_performance(self):
        """Benchmark tag address parsing performance."""
        # Test various address formats
        addresses = [
            "40001",
            "40001:10",
            "40001@5",
            "30001",
            "00001",
            "10001",
        ]
        
        def parse_modbus_address(address: str):
            """Simple Modbus address parser."""
            parts = address.split('@')
            addr_part = parts[0]
            slave_id = int(parts[1]) if len(parts) > 1 else 1
            
            if ':' in addr_part:
                base, count = addr_part.split(':')
                base_addr = int(base)
                count = int(count)
            else:
                base_addr = int(addr_part)
                count = 1
            
            # Determine function code from address range
            if 1 <= base_addr <= 9999:
                function_code = 1  # Coils
            elif 10001 <= base_addr <= 19999:
                function_code = 2  # Discrete inputs
            elif 30001 <= base_addr <= 39999:
                function_code = 4  # Input registers
            elif 40001 <= base_addr <= 49999:
                function_code = 3  # Holding registers
            else:
                raise ValueError(f"Invalid address: {address}")
            
            return function_code, base_addr % 10000 - 1, count, slave_id
        
        # Benchmark
        start_time = time.time()
        iterations = 100000
        
        for _ in range(iterations):
            for addr in addresses:
                result = parse_modbus_address(addr)
                assert result is not None
        
        elapsed_time = time.time() - start_time
        parses_per_second = (iterations * len(addresses)) / elapsed_time
        
        print(f"\nAddress parsing: {parses_per_second:.0f} parses/sec")
        assert parses_per_second > 1000000  # Should achieve > 1M parses/sec

    @pytest.mark.asyncio
    async def test_event_emission_performance(self):
        """Benchmark event emission performance."""
        from bifrost_core.events import EventBus, Event, EventType
        
        event_bus = EventBus()
        events_received = []
        
        async def handler(event):
            events_received.append(event)
        
        # Subscribe handler
        event_bus.subscribe(EventType.TAG_READ, handler)
        
        # Benchmark event emission
        start_time = time.time()
        iterations = 10000
        
        for i in range(iterations):
            event = Event(
                type=EventType.TAG_READ,
                data={"tag": f"tag_{i}", "value": i}
            )
            await event_bus.emit(event)
        
        # Wait for all events to be processed
        await asyncio.sleep(0.1)
        
        elapsed_time = time.time() - start_time
        events_per_second = iterations / elapsed_time
        
        print(f"\nEvent emission: {events_per_second:.0f} events/sec")
        assert events_per_second > 50000  # Should achieve > 50k events/sec
        assert len(events_received) == iterations

    @pytest.mark.asyncio
    async def test_memory_efficiency(self):
        """Test memory efficiency with large number of tags."""
        import sys
        
        # Create a large number of tags
        tags = []
        for i in range(10000):
            tag = Tag(
                name=f"tag_{i}",
                address=f"{40001 + (i % 1000)}",
                data_type=DataType.INT16
            )
            tags.append(tag)
        
        # Calculate memory usage
        total_size = sum(sys.getsizeof(tag) for tag in tags)
        avg_size_per_tag = total_size / len(tags)
        
        print(f"\nAverage memory per tag: {avg_size_per_tag:.0f} bytes")
        assert avg_size_per_tag < 500  # Should be less than 500 bytes per tag

    @pytest.mark.asyncio
    async def test_connection_establishment_performance(self):
        """Benchmark connection establishment performance."""
        # Mock fast connection establishment
        mock_client = MagicMock()
        mock_client.connect = AsyncMock(return_value=True)
        mock_client.close = AsyncMock()
        mock_client.is_socket_open = MagicMock(return_value=True)
        
        # Benchmark
        start_time = time.time()
        iterations = 100
        
        for _ in range(iterations):
            conn = ModbusConnection(host="192.168.1.100", port=502)
            conn.client = mock_client
            async with conn:
                pass
        
        elapsed_time = time.time() - start_time
        connections_per_second = iterations / elapsed_time
        
        print(f"\nConnection establishment: {connections_per_second:.0f} conn/sec")
        assert connections_per_second > 1000  # Should achieve > 1000 conn/sec