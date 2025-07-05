"""Tests for bifrost-core event system."""

from unittest.mock import AsyncMock

import pytest

from bifrost_core.events import BaseEvent, EventBus


class MockEvent(BaseEvent):
    """Mock event for testing."""

    def __init__(self, value: str):
        self.value = value


class TestEventBus:
    """Test EventBus class."""

    @pytest.mark.asyncio
    async def test_on_and_emit(self):
        bus = EventBus()
        mock_handler = AsyncMock()

        await bus.on(MockEvent, mock_handler)

        event = MockEvent("test_value")
        await bus.emit(event)

        mock_handler.assert_called_once_with(event)

    @pytest.mark.asyncio
    async def test_off(self):
        bus = EventBus()
        mock_handler = AsyncMock()

        await bus.on(MockEvent, mock_handler)
        await bus.off(MockEvent, mock_handler)

        event = MockEvent("test_value")
        await bus.emit(event)

        mock_handler.assert_not_called()

    @pytest.mark.asyncio
    async def test_multiple_handlers(self):
        bus = EventBus()
        mock_handler1 = AsyncMock()
        mock_handler2 = AsyncMock()

        await bus.on(MockEvent, mock_handler1)
        await bus.on(MockEvent, mock_handler2)

        event = MockEvent("test_value")
        await bus.emit(event)

        mock_handler1.assert_called_once_with(event)
        mock_handler2.assert_called_once_with(event)

    @pytest.mark.asyncio
    async def test_no_handlers(self):
        bus = EventBus()
        event = MockEvent("test_value")
        await bus.emit(event)
        # Should not raise an error

    @pytest.mark.asyncio
    async def test_different_event_types(self):
        class AnotherMockEvent(BaseEvent):
            pass

        bus = EventBus()
        mock_handler1 = AsyncMock()
        mock_handler2 = AsyncMock()

        await bus.on(MockEvent, mock_handler1)
        await bus.on(AnotherMockEvent, mock_handler2)

        event1 = MockEvent("test_value")
        event2 = AnotherMockEvent()

        await bus.emit(event1)
        await bus.emit(event2)

        mock_handler1.assert_called_once_with(event1)
        mock_handler2.assert_called_once_with(event2)
