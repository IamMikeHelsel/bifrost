"""Tests for connection pooling functionality."""

import asyncio

import pytest

from bifrost_core.base import BaseConnection
from bifrost_core.pooling import ConnectionPool


class MockConnection(BaseConnection):
    """Mock connection for testing."""

    def __init__(self, host: str, port: int = 502):
        self.host = host
        self.port = port
        self._is_connected_flag = False

    async def __aenter__(self) -> "MockConnection":
        self._is_connected_flag = True
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        self._is_connected_flag = False

    @property
    def is_connected(self) -> bool:
        return self._is_connected_flag

    async def read(self, tags):
        return dict.fromkeys(tags, 1)

    async def write(self, values):
        pass

    async def get_info(self):
        return {"host": self.host, "port": self.port}


class TestConnectionPool:
    """Test ConnectionPool functionality."""

    @pytest.mark.asyncio
    async def test_get_new_connection(self):
        async def factory():
            return MockConnection("localhost")

        pool = ConnectionPool(connection_factory=factory, max_size=1)
        conn = await pool.get()
        assert isinstance(conn, MockConnection)
        assert conn.is_connected

    @pytest.mark.asyncio
    async def test_put_and_get_reused_connection(self):
        async def factory():
            return MockConnection("localhost")

        pool = ConnectionPool(connection_factory=factory, max_size=1)
        conn1 = await pool.get()
        await pool.put(conn1)

        conn2 = await pool.get()
        assert conn1 is conn2  # Should be the same instance

    @pytest.mark.asyncio
    async def test_max_size_enforcement(self):
        async def factory():
            return MockConnection("localhost")

        pool = ConnectionPool(connection_factory=factory, max_size=1)
        conn1 = await pool.get()

        # Try to get another connection, should wait as max_size is 1
        get_task = asyncio.create_task(pool.get())
        await asyncio.sleep(0.01)  # Give it a moment to try and get
        assert not get_task.done()

        await pool.put(conn1)
        conn2 = await get_task
        assert conn1 is conn2

    @pytest.mark.asyncio
    async def test_put_disconnected_connection(self):
        async def factory():
            return MockConnection("localhost")

        pool = ConnectionPool(connection_factory=factory, max_size=1)
        conn = MockConnection("localhost")
        # Simulate a disconnected connection
        conn._is_connected_flag = False

        await pool.put(conn)

        # Pool should not contain the disconnected connection
        assert len(pool._pool) == 0

    @pytest.mark.asyncio
    async def test_connection_factory_async(self):
        async def async_factory():
            await asyncio.sleep(0.01)
            return MockConnection("async_host")

        pool = ConnectionPool(connection_factory=async_factory, max_size=1)
        conn = await pool.get()
        assert isinstance(conn, MockConnection)
        assert conn.host == "async_host"

    @pytest.mark.asyncio
    async def test_release_connection(self):
        async def factory():
            return MockConnection("localhost")

        pool = ConnectionPool(connection_factory=factory, max_size=1)
        conn = await pool.get()
        assert len(pool._used_connections) == 1
        await pool.release(conn)
        assert len(pool._used_connections) == 0
        assert len(pool._pool) == 1

    @pytest.mark.asyncio
    async def test_close_pool(self):
        async def factory():
            return MockConnection("localhost")

        pool = ConnectionPool(connection_factory=factory, max_size=2)
        conn1 = await pool.get()
        conn2 = await pool.get()

        await pool.close()

        assert not conn1.is_connected
        assert not conn2.is_connected
        assert len(pool._pool) == 0
        assert len(pool._used_connections) == 0

    @pytest.mark.asyncio
    async def test_get_with_timeout(self):
        async def factory():
            # This factory will never return a connection, simulating a timeout
            await asyncio.sleep(100)
            return MockConnection("localhost")

        pool = ConnectionPool(connection_factory=factory, max_size=1)
        with pytest.raises(asyncio.TimeoutError):
            await asyncio.wait_for(pool.get(), timeout=0.01)

    @pytest.mark.asyncio
    async def test_context_manager(self):
        async def factory():
            return MockConnection("localhost")

        pool = ConnectionPool(connection_factory=factory, max_size=1)
        async with pool as conn:
            assert isinstance(conn, MockConnection)
            assert conn.is_connected
        assert not conn.is_connected  # Should be closed after exiting context

    @pytest.mark.asyncio
    async def test_release_connection(self):
        pool = ConnectionPool(
            connection_factory=lambda: MockConnection("localhost"), max_size=1
        )
        conn = await pool.get()
        assert len(pool._used_connections) == 1
        await pool.release(conn)
        assert len(pool._used_connections) == 0
        assert len(pool._pool) == 1

    @pytest.mark.asyncio
    async def test_close_pool(self):
        pool = ConnectionPool(
            connection_factory=lambda: MockConnection("localhost"), max_size=2
        )
        conn1 = await pool.get()
        conn2 = await pool.get()

        await pool.close()

        assert not conn1.is_connected
        assert not conn2.is_connected
        assert len(pool._pool) == 0
        assert len(pool._used_connections) == 0

    @pytest.mark.asyncio
    async def test_get_with_timeout(self):
        pool = ConnectionPool(
            connection_factory=lambda: MockConnection("localhost"),
            max_size=0,  # No connections available
        )
        with pytest.raises(asyncio.TimeoutError):
            await asyncio.wait_for(pool.get(), timeout=0.01)

    @pytest.mark.asyncio
    async def test_context_manager(self):
        async def factory():
            return MockConnection("localhost")

        pool = ConnectionPool(connection_factory=factory, max_size=1)
        async with pool as conn:
            assert isinstance(conn, MockConnection)
            assert conn.is_connected
        assert not conn.is_connected  # Should be closed after exiting context
