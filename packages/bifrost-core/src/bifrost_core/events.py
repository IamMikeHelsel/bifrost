"""Event system for Bifrost connection lifecycle and data events."""

import asyncio
from datetime import datetime
from enum import Enum
from typing import Any, Callable, Dict, List, Optional, Set
from pydantic import BaseModel

from .base import ConnectionState


class EventType(Enum):
    """Types of events that can be emitted."""
    CONNECTION_STATE_CHANGED = "connection_state_changed"
    DATA_RECEIVED = "data_received"
    ERROR_OCCURRED = "error_occurred"
    DEVICE_DISCOVERED = "device_discovered"
    HEALTH_CHECK_FAILED = "health_check_failed"


class Event(BaseModel):
    """Base event class for all Bifrost events."""
    
    event_type: EventType
    timestamp: datetime
    source: str
    data: Dict[str, Any]
    
    def __str__(self) -> str:
        return f"Event({self.event_type.value}, {self.source})"


class ConnectionStateEvent(Event):
    """Event emitted when connection state changes."""
    
    def __init__(self, source: str, old_state: ConnectionState, new_state: ConnectionState):
        super().__init__(
            event_type=EventType.CONNECTION_STATE_CHANGED,
            timestamp=datetime.now(),
            source=source,
            data={
                "old_state": old_state.value,
                "new_state": new_state.value
            }
        )
    
    @property
    def old_state(self) -> ConnectionState:
        return ConnectionState(self.data["old_state"])
    
    @property
    def new_state(self) -> ConnectionState:
        return ConnectionState(self.data["new_state"])


class DataReceivedEvent(Event):
    """Event emitted when data is received from a device."""
    
    def __init__(self, source: str, address: str, value: Any, data_type: str):
        super().__init__(
            event_type=EventType.DATA_RECEIVED,
            timestamp=datetime.now(),
            source=source,
            data={
                "address": address,
                "value": value,
                "data_type": data_type
            }
        )


class ErrorEvent(Event):
    """Event emitted when an error occurs."""
    
    def __init__(self, source: str, error: Exception, context: Optional[Dict[str, Any]] = None):
        super().__init__(
            event_type=EventType.ERROR_OCCURRED,
            timestamp=datetime.now(),
            source=source,
            data={
                "error_type": type(error).__name__,
                "error_message": str(error),
                "context": context or {}
            }
        )


EventHandler = Callable[[Event], None]
AsyncEventHandler = Callable[[Event], None]


class EventBus:
    """Centralized event bus for Bifrost components."""
    
    def __init__(self) -> None:
        self._handlers: Dict[EventType, Set[EventHandler]] = {}
        self._async_handlers: Dict[EventType, Set[AsyncEventHandler]] = {}
        self._global_handlers: Set[EventHandler] = set()
        self._global_async_handlers: Set[AsyncEventHandler] = set()
        self._event_history: List[Event] = []
        self._max_history_size = 1000
    
    def subscribe(
        self, 
        event_type: EventType, 
        handler: EventHandler
    ) -> None:
        """Subscribe to events of a specific type."""
        if event_type not in self._handlers:
            self._handlers[event_type] = set()
        self._handlers[event_type].add(handler)
    
    def subscribe_async(
        self, 
        event_type: EventType, 
        handler: AsyncEventHandler
    ) -> None:
        """Subscribe to events of a specific type with async handler."""
        if event_type not in self._async_handlers:
            self._async_handlers[event_type] = set()
        self._async_handlers[event_type].add(handler)
    
    def subscribe_all(self, handler: EventHandler) -> None:
        """Subscribe to all events."""
        self._global_handlers.add(handler)
    
    def subscribe_all_async(self, handler: AsyncEventHandler) -> None:
        """Subscribe to all events with async handler."""
        self._global_async_handlers.add(handler)
    
    def unsubscribe(
        self, 
        event_type: EventType, 
        handler: EventHandler
    ) -> None:
        """Unsubscribe from events of a specific type."""
        if event_type in self._handlers:
            self._handlers[event_type].discard(handler)
    
    def unsubscribe_async(
        self, 
        event_type: EventType, 
        handler: AsyncEventHandler
    ) -> None:
        """Unsubscribe from events of a specific type with async handler."""
        if event_type in self._async_handlers:
            self._async_handlers[event_type].discard(handler)
    
    def unsubscribe_all(self, handler: EventHandler) -> None:
        """Unsubscribe from all events."""
        self._global_handlers.discard(handler)
    
    def unsubscribe_all_async(self, handler: AsyncEventHandler) -> None:
        """Unsubscribe from all events with async handler."""
        self._global_async_handlers.discard(handler)
    
    def emit(self, event: Event) -> None:
        """Emit an event to all subscribers."""
        # Add to history
        self._event_history.append(event)
        if len(self._event_history) > self._max_history_size:
            self._event_history.pop(0)
        
        # Notify specific handlers
        handlers = self._handlers.get(event.event_type, set())
        for handler in handlers:
            try:
                handler(event)
            except Exception as e:
                # Avoid infinite recursion by not emitting error events for handler failures
                print(f"Event handler error: {e}")
        
        # Notify global handlers
        for handler in self._global_handlers:
            try:
                handler(event)
            except Exception as e:
                print(f"Global event handler error: {e}")
        
        # Schedule async handlers
        async_handlers = self._async_handlers.get(event.event_type, set())
        for handler in async_handlers:
            asyncio.create_task(self._safe_async_handler(handler, event))
        
        for handler in self._global_async_handlers:
            asyncio.create_task(self._safe_async_handler(handler, event))
    
    async def _safe_async_handler(self, handler: AsyncEventHandler, event: Event) -> None:
        """Safely execute async event handler."""
        try:
            await handler(event)
        except Exception as e:
            print(f"Async event handler error: {e}")
    
    def get_recent_events(
        self, 
        count: int = 10, 
        event_type: Optional[EventType] = None
    ) -> List[Event]:
        """Get recent events from history."""
        events = self._event_history
        if event_type:
            events = [e for e in events if e.event_type == event_type]
        return events[-count:] if count else events
    
    def clear_history(self) -> None:
        """Clear event history."""
        self._event_history.clear()


# Global event bus instance
_global_event_bus = EventBus()


def get_global_event_bus() -> EventBus:
    """Get the global event bus instance."""
    return _global_event_bus


def emit_event(event: Event) -> None:
    """Emit an event to the global event bus."""
    _global_event_bus.emit(event)


def subscribe_to_events(event_type: EventType, handler: EventHandler) -> None:
    """Subscribe to events on the global event bus."""
    _global_event_bus.subscribe(event_type, handler)


def subscribe_to_events_async(event_type: EventType, handler: AsyncEventHandler) -> None:
    """Subscribe to events on the global event bus with async handler."""
    _global_event_bus.subscribe_async(event_type, handler)