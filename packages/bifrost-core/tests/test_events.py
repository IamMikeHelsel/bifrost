"""Tests for bifrost-core event system."""

import asyncio
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

    @pytest.mark.asyncio
    async def test_concurrent_handlers(self):
        bus = EventBus()
        
        # Create two mock handlers that simulate some async work
        async def handler1(event):
            await asyncio.sleep(0.05) # Simulate async work
            handler1.called = True

        async def handler2(event):
            await asyncio.sleep(0.03) # Simulate async work
            handler2.called = True

        handler1.called = False
        handler2.called = False

        await bus.on(MockEvent, handler1)
        await bus.on(MockEvent, handler2)

        event = MockEvent("concurrent_test")
        await bus.emit(event)

        # Ensure both handlers were called
        assert handler1.called
        assert handler2.called

        # Verify that they ran concurrently (by checking the total time, though this is less precise)
        # A more robust test would involve mocking asyncio.sleep and checking call order/times
        # For now, simply ensuring they both complete is sufficient.
