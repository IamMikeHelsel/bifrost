"""Tests for bifrost-core event system."""

import asyncio

import pytest
from bifrost_core import (
    ConnectionState,
    ConnectionStateEvent,
    DataReceivedEvent,
    ErrorEvent,
    EventBus,
    EventType,
)


class TestEventBus:
    """Test EventBus class."""

    def test_subscribe_and_emit(self):
        bus = EventBus()
        events_received = []

        def handler(event):
            events_received.append(event)

        bus.subscribe(EventType.CONNECTION_STATE_CHANGED, handler)

        event = ConnectionStateEvent(
            "test_device", ConnectionState.DISCONNECTED, ConnectionState.CONNECTED
        )
        bus.emit(event)

        assert len(events_received) == 1
        assert events_received[0] == event

    @pytest.mark.asyncio
    async def test_async_subscribe_and_emit(self):
        bus = EventBus()
        events_received = []

        async def async_handler(event):
            events_received.append(event)

        bus.subscribe_async(EventType.DATA_RECEIVED, async_handler)

        event = DataReceivedEvent("test_device", "40001", 123, "int32")
        bus.emit(event)

        # Give async handler time to execute
        await asyncio.sleep(0.1)

        assert len(events_received) == 1
        assert events_received[0] == event

    def test_global_handler(self):
        bus = EventBus()
        events_received = []

        def global_handler(event):
            events_received.append(event)

        bus.subscribe_all(global_handler)

        # Emit different types of events
        event1 = ConnectionStateEvent(
            "device1", ConnectionState.DISCONNECTED, ConnectionState.CONNECTED
        )
        event2 = DataReceivedEvent("device2", "40001", 456, "int32")

        bus.emit(event1)
        bus.emit(event2)

        assert len(events_received) == 2

    def test_unsubscribe(self):
        bus = EventBus()
        events_received = []

        def handler(event):
            events_received.append(event)

        bus.subscribe(EventType.CONNECTION_STATE_CHANGED, handler)
        bus.unsubscribe(EventType.CONNECTION_STATE_CHANGED, handler)

        event = ConnectionStateEvent(
            "test_device", ConnectionState.DISCONNECTED, ConnectionState.CONNECTED
        )
        bus.emit(event)

        assert len(events_received) == 0

    def test_event_history(self):
        bus = EventBus()

        event1 = ConnectionStateEvent(
            "device1", ConnectionState.DISCONNECTED, ConnectionState.CONNECTED
        )
        event2 = DataReceivedEvent("device2", "40001", 789, "int32")

        bus.emit(event1)
        bus.emit(event2)

        recent = bus.get_recent_events(count=5)
        assert len(recent) == 2
        assert recent[0] == event1
        assert recent[1] == event2

    def test_filtered_event_history(self):
        bus = EventBus()

        event1 = ConnectionStateEvent(
            "device1", ConnectionState.DISCONNECTED, ConnectionState.CONNECTED
        )
        event2 = DataReceivedEvent("device2", "40001", 789, "int32")

        bus.emit(event1)
        bus.emit(event2)

        # Get only connection events
        conn_events = bus.get_recent_events(
            count=10, event_type=EventType.CONNECTION_STATE_CHANGED
        )
        assert len(conn_events) == 1
        assert conn_events[0] == event1


class TestEventTypes:
    """Test specific event types."""

    def test_connection_state_event(self):
        event = ConnectionStateEvent(
            "test_device", ConnectionState.DISCONNECTED, ConnectionState.CONNECTED
        )

        assert event.event_type == EventType.CONNECTION_STATE_CHANGED
        assert event.source == "test_device"
        assert event.old_state == ConnectionState.DISCONNECTED
        assert event.new_state == ConnectionState.CONNECTED

    def test_data_received_event(self):
        event = DataReceivedEvent("test_device", "40001", 123, "int32")

        assert event.event_type == EventType.DATA_RECEIVED
        assert event.source == "test_device"
        assert event.data["address"] == "40001"
        assert event.data["value"] == 123
        assert event.data["data_type"] == "int32"

    def test_error_event(self):
        error = ValueError("Test error")
        event = ErrorEvent("test_device", error, {"context": "test"})

        assert event.event_type == EventType.ERROR_OCCURRED
        assert event.source == "test_device"
        assert event.data["error_type"] == "ValueError"
        assert event.data["error_message"] == "Test error"
        assert event.data["context"]["context"] == "test"

    def test_event_string_representation(self):
        event = ConnectionStateEvent(
            "test_device", ConnectionState.DISCONNECTED, ConnectionState.CONNECTED
        )

        assert "Event(connection_state_changed" in str(event)
        assert "test_device" in str(event)
