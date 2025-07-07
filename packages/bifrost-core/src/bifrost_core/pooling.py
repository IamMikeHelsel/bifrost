"""A generic connection pool for Bifrost."""

import asyncio
from collections import deque
from collections.abc import Callable, Coroutine
from types import TracebackType
from typing import Generic, TypeVar

from .base import BaseConnection

C = TypeVar("C", bound=BaseConnection)


class ConnectionPool(Generic[C]):
    """A generic connection pool for managing and reusing connections."""

    def __init__(
        self,
        connection_factory: Callable[[], Coroutine[None, None, C]],
        max_size: int = 10,
    ):
        """Initializes the ConnectionPool."""
        self._factory = connection_factory
        self._max_size = max_size
        self._pool: deque[C] = deque(maxlen=max_size)
        self._lock = asyncio.Lock()
        self._created_connections = 0
        self._used_connections: set[C] = set()

    async def __aenter__(self) -> "ConnectionPool[C]":
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: TracebackType | None,
    ) -> None:
        await self.close()

    async def get(self) -> C:
        """Get a connection from the pool."""
        async with self._lock:
            if self._pool:
                conn = self._pool.popleft()
                self._used_connections.add(conn)
                return conn

            if self._created_connections < self._max_size:
                self._created_connections += 1
                conn = await self._factory()
                self._used_connections.add(conn)
                return conn

            # Wait for a connection to be returned to the pool
            while not self._pool:
                await asyncio.sleep(0.01)  # Yield control to event loop
            conn = self._pool.popleft()
            self._used_connections.add(conn)
            return conn

    async def put(self, connection: C) -> None:
        """Return a connection to the pool."""
        async with self._lock:
            if connection in self._used_connections:
                self._used_connections.remove(connection)
            if connection.is_connected:
                self._pool.append(connection)

    async def release(self, connection: C) -> None:
        """Release a connection back to the pool."""
        await self.put(connection)

    async def close(self) -> None:
        """Close all connections in the pool and clear it."""
        async with self._lock:
            # Close connections in the pool
            while self._pool:
                conn = self._pool.popleft()
                if conn.is_connected:
                    await conn.__aexit__(
                        None, None, None
                    )  # Manually exit context

            # Close connections currently in use
            for conn in list(self._used_connections):  # Iterate over a copy
                if conn.is_connected:
                    await conn.__aexit__(None, None, None)
                self._used_connections.remove(conn)

            self._created_connections = 0
