"""A simple, thread-safe, async-first event system for Bifrost."""

import asyncio
from collections import defaultdict
from collections.abc import Callable, Coroutine
from typing import TypeVar, cast


class BaseEvent:
    """Base class for all events."""


Event = TypeVar("Event", bound=BaseEvent)

# A coroutine function that handles an event
EventHandler = Callable[[Event], Coroutine[None, None, None]]


class EventBus:
    """A simple event bus for dispatching events to listeners."""

    def __init__(self) -> None:
        self._listeners: dict[type[BaseEvent], list[EventHandler[BaseEvent]]] = (
            defaultdict(list)
        )
        self._lock = asyncio.Lock()

    async def on(self, event_type: type[Event], handler: EventHandler[Event]) -> None:
        """Register an event handler for a given event type."""
        async with self._lock:
            self._listeners[event_type].append(cast(EventHandler[BaseEvent], handler))

    async def off(self, event_type: type[Event], handler: EventHandler[Event]) -> None:
        """Unregister an event handler."""
        async with self._lock:
            if cast(EventHandler[BaseEvent], handler) in self._listeners[event_type]:
                self._listeners[event_type].remove(
                    cast(EventHandler[BaseEvent], handler)
                )

    async def emit(self, event: BaseEvent) -> None:
        """Dispatch an event to all registered listeners."""
        handlers = self._listeners.get(type(event), [])
        if handlers:
            await asyncio.gather(*(handler(event) for handler in handlers))
