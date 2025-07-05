"""A generic connection pool for Bifrost."""

import asyncio
from collections import deque
from typing import Callable, Generic

from .base import BaseConnection
from .typing import T


class ConnectionPool(Generic[T]):
    """A generic connection pool for managing and reusing connections."""

    def __init__(self, connection_factory: Callable[[], T], max_size: int = 10):
        self._factory = connection_factory
        self._max_size = max_size
        self._pool: deque[T] = deque(maxlen=max_size)
        self._lock = asyncio.Lock()
        self._created_connections = 0

    async def get(self) -> T:
        """Get a connection from the pool."""
        async with self._lock:
            if self._pool:
                return self._pool.popleft()

            if self._created_connections < self._max_size:
                self._created_connections += 1
                return self._factory()

            # Wait for a connection to be returned to the pool
            while not self._pool:
                await asyncio.sleep(0.01)
            return self._pool.popleft()

    async def put(self, connection: T) -> None:
        """Return a connection to the pool."""
        async with self._lock:
            if isinstance(connection, BaseConnection) and connection.is_connected:
                self._pool.append(connection)
