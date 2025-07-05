"""Tests for connection pooling functionality."""

import asyncio
from unittest.mock import AsyncMock

import pytest
from bifrost_core.base import BaseConnection
from bifrost_core.pooling import ConnectionPool


class MockConnection(BaseConnection):
    """Mock connection for testing."""

    def __init__(self, host: str, port: int = 502):
        super().__init__(host, port)
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
        return {tag: 1 for tag in tags}

    async def write(self, values):
        pass

    async def get_info(self):
        return {"host": self.host, "port": self.port}


class TestConnectionPool:
    """Test ConnectionPool functionality."""

    @pytest.mark.asyncio
    async def test_get_new_connection(self):
        pool = ConnectionPool(connection_factory=lambda: MockConnection("localhost"), max_size=1)
        conn = await pool.get()
        assert isinstance(conn, MockConnection)
        assert conn.is_connected

    @pytest.mark.asyncio
    async def test_put_and_get_reused_connection(self):
        pool = ConnectionPool(connection_factory=lambda: MockConnection("localhost"), max_size=1)
        conn1 = await pool.get()
        await pool.put(conn1)

        conn2 = await pool.get()
        assert conn1 is conn2  # Should be the same instance

    @pytest.mark.asyncio
    async def test_max_size_enforcement(self):
        pool = ConnectionPool(connection_factory=lambda: MockConnection("localhost"), max_size=1)
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
        pool = ConnectionPool(connection_factory=lambda: MockConnection("localhost"), max_size=1)
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