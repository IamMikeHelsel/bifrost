"""Connection pooling for efficient resource management."""

import asyncio
from contextlib import asynccontextmanager
from datetime import datetime, timedelta
from typing import Any, AsyncIterator, Dict, List, Optional, Set
from weakref import WeakSet

from .base import BaseConnection, ConnectionState
from .events import ConnectionStateEvent, ErrorEvent, emit_event


class PooledConnection:
    """Wrapper for connections in the pool."""
    
    def __init__(self, connection: BaseConnection, pool: "ConnectionPool"):
        self.connection = connection
        self.pool = pool
        self.created_at = datetime.now()
        self.last_used = datetime.now()
        self.use_count = 0
        self.is_borrowed = False
        self._original_disconnect = connection.disconnect
        
        # Override disconnect to return to pool instead
        connection.disconnect = self._return_to_pool
    
    async def _return_to_pool(self) -> None:
        """Return connection to pool instead of disconnecting."""
        if self.is_borrowed:
            self.pool._return_connection(self)
    
    async def actual_disconnect(self) -> None:
        """Actually disconnect the underlying connection."""
        await self._original_disconnect()
    
    def touch(self) -> None:
        """Update last used timestamp and increment use count."""
        self.last_used = datetime.now()
        self.use_count += 1
    
    @property
    def age(self) -> timedelta:
        """Age of the connection since creation."""
        return datetime.now() - self.created_at
    
    @property
    def idle_time(self) -> timedelta:
        """Time since last use."""
        return datetime.now() - self.last_used


class ConnectionPool:
    """Pool for managing multiple connections efficiently."""
    
    def __init__(
        self,
        max_size: int = 10,
        min_size: int = 0,
        max_idle_time: int = 300,  # 5 minutes
        max_lifetime: int = 3600,  # 1 hour
        health_check_interval: int = 60,  # 1 minute
    ):
        self.max_size = max_size
        self.min_size = min_size
        self.max_idle_time = timedelta(seconds=max_idle_time)
        self.max_lifetime = timedelta(seconds=max_lifetime)
        self.health_check_interval = health_check_interval
        
        self._available: Set[PooledConnection] = set()
        self._borrowed: Set[PooledConnection] = set()
        self._creating: Set[str] = set()  # Track connections being created
        self._closed = False
        self._health_check_task: Optional[asyncio.Task] = None
        
        # Health checks will be started when first connection is requested
        self._health_checks_started = False
    
    @property
    def size(self) -> int:
        """Total number of connections in pool."""
        return len(self._available) + len(self._borrowed)
    
    @property
    def available_count(self) -> int:
        """Number of available connections."""
        return len(self._available)
    
    @property
    def borrowed_count(self) -> int:
        """Number of borrowed connections."""
        return len(self._borrowed)
    
    async def get_connection(
        self, 
        connection_factory,
        connection_key: str,
        **factory_kwargs
    ) -> BaseConnection:
        """Get a connection from the pool or create a new one."""
        if self._closed:
            raise RuntimeError("Connection pool is closed")
        
        # Start health checks on first use
        if not self._health_checks_started:
            self._start_health_checks()
            self._health_checks_started = True
        
        # Try to get an available connection first
        pooled_conn = self._get_available_connection()
        
        if pooled_conn is None and self.size < self.max_size:
            # Create new connection if pool not at capacity
            if connection_key not in self._creating:
                self._creating.add(connection_key)
                try:
                    connection = await connection_factory(**factory_kwargs)
                    await connection.connect()
                    pooled_conn = PooledConnection(connection, self)
                finally:
                    self._creating.discard(connection_key)
        
        if pooled_conn is None:
            raise RuntimeError("No available connections and pool at capacity")
        
        # Mark as borrowed and return
        pooled_conn.is_borrowed = True
        pooled_conn.touch()
        self._available.discard(pooled_conn)
        self._borrowed.add(pooled_conn)
        
        return pooled_conn.connection
    
    @asynccontextmanager
    async def connection(
        self, 
        connection_factory,
        connection_key: str,
        **factory_kwargs
    ) -> AsyncIterator[BaseConnection]:
        """Context manager for getting and returning connections."""
        conn = await self.get_connection(connection_factory, connection_key, **factory_kwargs)
        try:
            yield conn
        finally:
            # Connection will be returned to pool via overridden disconnect()
            await conn.disconnect()
    
    def _get_available_connection(self) -> Optional[PooledConnection]:
        """Get an available connection from the pool."""
        if not self._available:
            return None
        
        # Get the most recently used connection
        return max(self._available, key=lambda c: c.last_used)
    
    def _return_connection(self, pooled_conn: PooledConnection) -> None:
        """Return a connection to the available pool."""
        if pooled_conn in self._borrowed:
            pooled_conn.is_borrowed = False
            self._borrowed.remove(pooled_conn)
            
            # Check if connection is still healthy
            if (pooled_conn.connection.is_connected and 
                pooled_conn.age < self.max_lifetime and
                pooled_conn.idle_time < self.max_idle_time):
                self._available.add(pooled_conn)
            else:
                # Connection is stale, close it
                asyncio.create_task(pooled_conn.actual_disconnect())
    
    async def _cleanup_stale_connections(self) -> None:
        """Remove stale connections from the pool."""
        now = datetime.now()
        to_remove = set()
        
        for pooled_conn in self._available.copy():
            if (pooled_conn.age > self.max_lifetime or 
                pooled_conn.idle_time > self.max_idle_time or
                not pooled_conn.connection.is_connected):
                to_remove.add(pooled_conn)
        
        for pooled_conn in to_remove:
            self._available.discard(pooled_conn)
            await pooled_conn.actual_disconnect()
    
    async def _health_check_connections(self) -> None:
        """Perform health checks on all connections."""
        unhealthy = set()
        
        # Check available connections
        for pooled_conn in self._available.copy():
            try:
                is_healthy = await pooled_conn.connection.health_check()
                if not is_healthy:
                    unhealthy.add(pooled_conn)
            except Exception as e:
                unhealthy.add(pooled_conn)
                emit_event(ErrorEvent(
                    source=f"pool:{pooled_conn.connection.connection_id}",
                    error=e,
                    context={"operation": "health_check"}
                ))
        
        # Remove unhealthy connections
        for pooled_conn in unhealthy:
            self._available.discard(pooled_conn)
            await pooled_conn.actual_disconnect()
    
    def _start_health_checks(self) -> None:
        """Start background health check task."""
        async def health_check_loop():
            while not self._closed:
                try:
                    await self._cleanup_stale_connections()
                    await self._health_check_connections()
                except Exception as e:
                    emit_event(ErrorEvent(
                        source="connection_pool",
                        error=e,
                        context={"operation": "health_check_loop"}
                    ))
                
                await asyncio.sleep(self.health_check_interval)
        
        self._health_check_task = asyncio.create_task(health_check_loop())
    
    async def close(self) -> None:
        """Close all connections and shut down the pool."""
        self._closed = True
        
        # Cancel health check task
        if self._health_check_task:
            self._health_check_task.cancel()
            try:
                await self._health_check_task
            except asyncio.CancelledError:
                pass
        
        # Close all connections
        all_connections = list(self._available) + list(self._borrowed)
        for pooled_conn in all_connections:
            await pooled_conn.actual_disconnect()
        
        self._available.clear()
        self._borrowed.clear()
    
    def get_stats(self) -> Dict[str, Any]:
        """Get pool statistics."""
        return {
            "total_connections": self.size,
            "available_connections": self.available_count,
            "borrowed_connections": self.borrowed_count,
            "max_size": self.max_size,
            "is_closed": self._closed,
            "connections_being_created": len(self._creating)
        }


# Global connection pool
_global_pool = ConnectionPool()


def get_global_pool() -> ConnectionPool:
    """Get the global connection pool."""
    return _global_pool


@asynccontextmanager
async def pooled_connection(
    connection_factory,
    connection_key: str,
    **factory_kwargs
) -> AsyncIterator[BaseConnection]:
    """Get a pooled connection using the global pool."""
    async with _global_pool.connection(
        connection_factory, 
        connection_key, 
        **factory_kwargs
    ) as conn:
        yield conn