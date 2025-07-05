"""A generic connection pool for Bifrost."""

import asyncio
from collections import deque
from collections.abc import Callable, Coroutine
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
        """Initialize the connection pool.

        Args:
            connection_factory: Factory function to create new connections.
            max_size: Maximum number of connections in the pool.
        """
        self._factory = connection_factory
        self._max_size = max_size
        self._pool: deque[C] = deque(maxlen=max_size)
        self._lock = asyncio.Lock()
        self._created_connections = 0

    async def get(self) -> C:
        """Get a connection from the pool."""
        async with self._lock:
            if self._pool:
                return self._pool.popleft()

            if self._created_connections < self._max_size:
                self._created_connections += 1
                return await self._factory()

            # Wait for a connection to be returned to the pool
            while not self._pool:
                await asyncio.sleep(0.01)
            return self._pool.popleft()

    async def put(self, connection: C) -> None:
        """Return a connection to the pool."""
        async with self._lock:
            if connection.is_connected:
                self._pool.append(connection)
