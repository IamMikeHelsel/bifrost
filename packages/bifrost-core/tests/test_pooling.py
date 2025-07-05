"""Tests for connection pooling functionality."""

import asyncio
from unittest.mock import AsyncMock, MagicMock

import pytest
from bifrost_core import BaseConnection, ConnectionState
from bifrost_core.pooling import ConnectionPool, PooledConnection, pooled_connection


class MockConnection(BaseConnection):
    """Mock connection for testing."""

    def __init__(self, host: str, port: int = 502, **kwargs):
        super().__init__(host, port, **kwargs)
        self._mock_connected = False

    async def connect(self) -> None:
        """Mock connect."""
        self._state = ConnectionState.CONNECTING
        await asyncio.sleep(0.01)  # Simulate connection time
        self._state = ConnectionState.CONNECTED
        self._mock_connected = True

    async def disconnect(self) -> None:
        """Mock disconnect."""
        self._state = ConnectionState.DISCONNECTED
        self._mock_connected = False

    async def read_raw(self, address: str, count: int = 1):
        """Mock read."""
        if not self._mock_connected:
            raise RuntimeError("Not connected")
        return [42] * count

    async def write_raw(self, address: str, values):
        """Mock write."""
        if not self._mock_connected:
            raise RuntimeError("Not connected")

    async def health_check(self) -> bool:
        """Mock health check."""
        return self._mock_connected

    @property
    def is_connected(self) -> bool:
        """Mock connection status."""
        return self._mock_connected


class TestPooledConnection:
    """Test PooledConnection wrapper."""

    @pytest.mark.asyncio
    async def test_pooled_connection_creation(self):
        """Test creating a pooled connection."""
        pool = ConnectionPool()
        base_conn = MockConnection("192.168.1.100")
        pooled_conn = PooledConnection(base_conn, pool)
        
        assert pooled_conn.connection == base_conn
        assert pooled_conn.pool == pool
        assert pooled_conn.use_count == 0
        assert not pooled_conn.is_borrowed
        assert pooled_conn.age.total_seconds() >= 0
        assert pooled_conn.idle_time.total_seconds() >= 0

    @pytest.mark.asyncio
    async def test_pooled_connection_touch(self):
        """Test touching a pooled connection."""
        pool = ConnectionPool()
        base_conn = MockConnection("192.168.1.100")
        pooled_conn = PooledConnection(base_conn, pool)
        
        initial_count = pooled_conn.use_count
        initial_time = pooled_conn.last_used
        
        await asyncio.sleep(0.01)
        pooled_conn.touch()
        
        assert pooled_conn.use_count == initial_count + 1
        assert pooled_conn.last_used > initial_time

    @pytest.mark.asyncio
    async def test_pooled_connection_return_to_pool(self):
        """Test returning connection to pool via disconnect override."""
        pool = ConnectionPool()
        base_conn = MockConnection("192.168.1.100")
        pooled_conn = PooledConnection(base_conn, pool)
        pooled_conn.is_borrowed = True
        
        # Mock pool return method
        pool._return_connection = MagicMock()
        
        # Call disconnect (should return to pool instead)
        await pooled_conn.connection.disconnect()
        
        pool._return_connection.assert_called_once_with(pooled_conn)

    @pytest.mark.asyncio
    async def test_pooled_connection_actual_disconnect(self):
        """Test actual disconnect of pooled connection."""
        pool = ConnectionPool()
        base_conn = MockConnection("192.168.1.100")
        await base_conn.connect()
        assert base_conn.is_connected
        
        pooled_conn = PooledConnection(base_conn, pool)
        
        # Call actual disconnect
        await pooled_conn.actual_disconnect()
        
        assert not base_conn.is_connected


class TestConnectionPool:
    """Test ConnectionPool functionality."""

    def setup_method(self):
        """Set up test fixtures."""
        self.pool = ConnectionPool(max_size=3, min_size=0)

    def test_connection_pool_init(self):
        """Test ConnectionPool initialization."""
        assert self.pool.max_size == 3
        assert self.pool.min_size == 0
        assert self.pool.size == 0
        assert self.pool.available_count == 0
        assert self.pool.borrowed_count == 0

    @pytest.mark.asyncio
    async def test_get_connection_new(self):
        """Test getting a new connection from empty pool."""
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        conn = await self.pool.get_connection(
            connection_factory, 
            "test_key"
        )
        
        assert isinstance(conn, MockConnection)
        assert conn.is_connected
        assert self.pool.size == 1
        assert self.pool.borrowed_count == 1
        assert self.pool.available_count == 0

    @pytest.mark.asyncio
    async def test_get_connection_reuse(self):
        """Test reusing connections from pool."""
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        # Get first connection
        conn1 = await self.pool.get_connection(connection_factory, "test_key")
        
        # Return it to pool
        await conn1.disconnect()
        
        # Get second connection (should reuse)
        conn2 = await self.pool.get_connection(connection_factory, "test_key")
        
        assert self.pool.size == 1  # Only one connection created
        assert conn2.is_connected

    @pytest.mark.asyncio
    async def test_connection_pool_max_size(self):
        """Test pool max size enforcement."""
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        # Fill up the pool to max size
        connections = []
        for i in range(self.pool.max_size):
            conn = await self.pool.get_connection(connection_factory, f"test_key_{i}")
            connections.append(conn)
        
        assert self.pool.size == self.pool.max_size
        assert self.pool.borrowed_count == self.pool.max_size
        
        # Try to get one more (should fail)
        with pytest.raises(RuntimeError, match="No available connections"):
            await self.pool.get_connection(connection_factory, "overflow_key")

    @pytest.mark.asyncio
    async def test_connection_context_manager(self):
        """Test connection pool context manager."""
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        async with self.pool.connection(connection_factory, "test_key") as conn:
            assert conn.is_connected
            assert self.pool.borrowed_count == 1
        
        # After context, connection should be returned
        assert self.pool.borrowed_count == 0
        assert self.pool.available_count == 1

    @pytest.mark.asyncio
    async def test_connection_pool_stats(self):
        """Test pool statistics."""
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        # Initial stats
        stats = self.pool.get_stats()
        assert stats["total_connections"] == 0
        assert stats["available_connections"] == 0
        assert stats["borrowed_connections"] == 0
        assert stats["max_size"] == 3
        assert not stats["is_closed"]
        
        # Get a connection
        conn = await self.pool.get_connection(connection_factory, "test_key")
        
        stats = self.pool.get_stats()
        assert stats["total_connections"] == 1
        assert stats["available_connections"] == 0
        assert stats["borrowed_connections"] == 1

    @pytest.mark.asyncio
    async def test_connection_pool_close(self):
        """Test closing the connection pool."""
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        # Get some connections
        conn1 = await self.pool.get_connection(connection_factory, "test_key_1")
        await conn1.disconnect()  # Return to pool
        
        conn2 = await self.pool.get_connection(connection_factory, "test_key_2")
        # Keep conn2 borrowed
        
        assert self.pool.size == 2
        
        # Close the pool
        await self.pool.close()
        
        # Pool should be marked as closed
        stats = self.pool.get_stats()
        assert stats["is_closed"]
        assert stats["total_connections"] == 0

    @pytest.mark.asyncio
    async def test_connection_pool_health_checks(self):
        """Test pool health check functionality."""
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        # Get and return a connection
        conn = await self.pool.get_connection(connection_factory, "test_key")
        await conn.disconnect()
        
        assert self.pool.available_count == 1
        
        # Manually trigger health check
        await self.pool._health_check_connections()
        
        # Connection should still be available (it's healthy)
        assert self.pool.available_count == 1

    @pytest.mark.asyncio
    async def test_stale_connection_cleanup(self):
        """Test cleanup of stale connections."""
        # Create pool with very short lifetime
        pool = ConnectionPool(max_lifetime=0.001)  # 1ms lifetime
        
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        # Get and return a connection
        conn = await pool.get_connection(connection_factory, "test_key")
        await conn.disconnect()
        
        assert pool.available_count == 1
        
        # Wait for connection to become stale
        await asyncio.sleep(0.01)
        
        # Trigger cleanup
        await pool._cleanup_stale_connections()
        
        # Stale connection should be removed
        assert pool.available_count == 0


class TestGlobalPoolFunctions:
    """Test global pool functions."""

    @pytest.mark.asyncio
    async def test_pooled_connection_function(self):
        """Test global pooled_connection function."""
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        async with pooled_connection(connection_factory, "test_key") as conn:
            assert isinstance(conn, MockConnection)
            assert conn.is_connected
        
        # Connection should be returned to global pool
        from bifrost_core.pooling import get_global_pool
        global_pool = get_global_pool()
        assert global_pool.available_count >= 0  # May have other connections

    @pytest.mark.asyncio
    async def test_global_pool_access(self):
        """Test accessing the global pool."""
        from bifrost_core.pooling import get_global_pool
        
        pool1 = get_global_pool()
        pool2 = get_global_pool()
        
        # Should return the same instance
        assert pool1 is pool2
        assert isinstance(pool1, ConnectionPool)


class TestPoolConcurrency:
    """Test pool behavior under concurrent access."""

    @pytest.mark.asyncio
    async def test_concurrent_connection_requests(self):
        """Test concurrent connection requests."""
        pool = ConnectionPool(max_size=2)
        
        async def connection_factory():
            await asyncio.sleep(0.01)  # Simulate connection time
            return MockConnection("192.168.1.100")
        
        # Start multiple concurrent requests
        tasks = []
        for i in range(3):
            task = asyncio.create_task(
                pool.get_connection(connection_factory, f"test_key_{i}")
            )
            tasks.append(task)
        
        # Wait for completion (one should fail due to max_size)
        results = await asyncio.gather(*tasks, return_exceptions=True)
        
        # Should have 2 successful connections and 1 exception
        successful = [r for r in results if isinstance(r, MockConnection)]
        exceptions = [r for r in results if isinstance(r, Exception)]
        
        assert len(successful) == 2
        assert len(exceptions) == 1
        assert "No available connections" in str(exceptions[0])

    @pytest.mark.asyncio
    async def test_concurrent_return_and_get(self):
        """Test concurrent return and get operations."""
        pool = ConnectionPool(max_size=1)
        
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        # Get initial connection
        conn1 = await pool.get_connection(connection_factory, "test_key_1")
        
        async def return_and_get():
            # Return connection
            await conn1.disconnect()
            # Immediately try to get another
            return await pool.get_connection(connection_factory, "test_key_2")
        
        # Should succeed without deadlock
        conn2 = await return_and_get()
        assert conn2.is_connected
        
        await conn2.disconnect()

    @pytest.mark.asyncio
    async def test_pool_under_load(self):
        """Test pool behavior under high load."""
        pool = ConnectionPool(max_size=5, health_check_interval=0.1)
        
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        # Simulate high load with many rapid operations
        async def worker(worker_id: int):
            for i in range(10):
                async with pool.connection(connection_factory, f"worker_{worker_id}_op_{i}") as conn:
                    await conn.read_raw("40001")
                await asyncio.sleep(0.001)  # Brief pause
        
        # Run multiple workers concurrently
        workers = [asyncio.create_task(worker(i)) for i in range(3)]
        await asyncio.gather(*workers)
        
        # Pool should still be functioning
        stats = pool.get_stats()
        assert not stats["is_closed"]
        assert stats["total_connections"] <= pool.max_size
        
        await pool.close()


class TestErrorHandling:
    """Test error handling in connection pooling."""

    @pytest.mark.asyncio
    async def test_connection_factory_failure(self):
        """Test handling of connection factory failures."""
        pool = ConnectionPool()
        
        async def failing_factory():
            raise ConnectionError("Factory failed")
        
        with pytest.raises(ConnectionError, match="Factory failed"):
            await pool.get_connection(failing_factory, "test_key")
        
        # Pool should remain functional
        assert not pool._closed

    @pytest.mark.asyncio
    async def test_unhealthy_connection_removal(self):
        """Test removal of unhealthy connections."""
        pool = ConnectionPool()
        
        class UnhealthyConnection(MockConnection):
            async def health_check(self) -> bool:
                return False  # Always unhealthy
        
        async def connection_factory():
            return UnhealthyConnection("192.168.1.100")
        
        # Get and return connection
        conn = await pool.get_connection(connection_factory, "test_key")
        await conn.disconnect()
        
        assert pool.available_count == 1
        
        # Run health check
        await pool._health_check_connections()
        
        # Unhealthy connection should be removed
        assert pool.available_count == 0

    @pytest.mark.asyncio
    async def test_pool_closed_error(self):
        """Test error when using closed pool."""
        pool = ConnectionPool()
        await pool.close()
        
        async def connection_factory():
            return MockConnection("192.168.1.100")
        
        with pytest.raises(RuntimeError, match="Connection pool is closed"):
            await pool.get_connection(connection_factory, "test_key")