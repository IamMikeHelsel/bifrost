"""Comprehensive tests for device discovery functionality."""

import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from bifrost.discovery import DeviceDiscovery, DiscoveryMethod
from bifrost_core.base import DeviceInfo
from bifrost_core.events import EventBus, EventType


class MockProtocolHandler:
    """Mock protocol handler for testing."""
    
    def __init__(self, protocol_name: str):
        self.protocol_name = protocol_name
        self.discover_devices = AsyncMock()
        self.get_device_info = AsyncMock()


class TestDeviceDiscovery:
    """Test DeviceDiscovery functionality."""

    @pytest.fixture
    def event_bus(self):
        """Create event bus for testing."""
        return EventBus()

    @pytest.fixture
    async def discovery(self, event_bus):
        """Create DeviceDiscovery instance."""
        discovery = DeviceDiscovery(event_bus)
        yield discovery
        await discovery.stop()

    @pytest.fixture
    def mock_handlers(self):
        """Create mock protocol handlers."""
        return {
            "modbus": MockProtocolHandler("modbus"),
            "ethernetip": MockProtocolHandler("ethernetip"),
            "opcua": MockProtocolHandler("opcua"),
        }

    @pytest.mark.asyncio
    async def test_register_protocol_handler(self, discovery):
        """Test registering protocol handlers."""
        handler = MockProtocolHandler("test-protocol")
        
        discovery.register_protocol_handler("test-protocol", handler)
        
        assert "test-protocol" in discovery._protocol_handlers
        assert discovery._protocol_handlers["test-protocol"] == handler

    @pytest.mark.asyncio
    async def test_register_duplicate_handler(self, discovery):
        """Test registering duplicate protocol handler."""
        handler1 = MockProtocolHandler("test-protocol")
        handler2 = MockProtocolHandler("test-protocol")
        
        discovery.register_protocol_handler("test-protocol", handler1)
        
        # Should overwrite existing handler
        discovery.register_protocol_handler("test-protocol", handler2)
        
        assert discovery._protocol_handlers["test-protocol"] == handler2

    @pytest.mark.asyncio
    async def test_discover_single_protocol(self, discovery, mock_handlers):
        """Test discovery with single protocol."""
        # Register handlers
        for protocol, handler in mock_handlers.items():
            discovery.register_protocol_handler(protocol, handler)
        
        # Set up mock responses
        mock_handlers["modbus"].discover_devices.return_value = [
            DeviceInfo(
                device_id="modbus-device-1",
                protocol="modbus",
                host="192.168.1.100",
                port=502,
                discovery_method="network_scan"
            )
        ]
        
        # Perform discovery
        devices = await discovery.discover_devices(
            protocols=["modbus"],
            network_range="192.168.1.0/24"
        )
        
        assert len(devices) == 1
        assert devices[0].protocol == "modbus"
        assert devices[0].device_id == "modbus-device-1"
        
        # Verify only modbus handler was called
        mock_handlers["modbus"].discover_devices.assert_called_once()
        mock_handlers["ethernetip"].discover_devices.assert_not_called()
        mock_handlers["opcua"].discover_devices.assert_not_called()

    @pytest.mark.asyncio
    async def test_discover_multiple_protocols(self, discovery, mock_handlers):
        """Test discovery with multiple protocols."""
        # Register handlers
        for protocol, handler in mock_handlers.items():
            discovery.register_protocol_handler(protocol, handler)
        
        # Set up mock responses
        mock_handlers["modbus"].discover_devices.return_value = [
            DeviceInfo(
                device_id="modbus-device-1",
                protocol="modbus",
                host="192.168.1.100",
                port=502,
                discovery_method="network_scan"
            )
        ]
        mock_handlers["ethernetip"].discover_devices.return_value = [
            DeviceInfo(
                device_id="eip-device-1",
                protocol="ethernetip",
                host="192.168.1.101",
                port=44818,
                discovery_method="network_scan"
            ),
            DeviceInfo(
                device_id="eip-device-2",
                protocol="ethernetip",
                host="192.168.1.102",
                port=44818,
                discovery_method="network_scan"
            )
        ]
        
        # Perform discovery
        devices = await discovery.discover_devices(
            protocols=["modbus", "ethernetip"],
            network_range="192.168.1.0/24"
        )
        
        assert len(devices) == 3
        assert sum(1 for d in devices if d.protocol == "modbus") == 1
        assert sum(1 for d in devices if d.protocol == "ethernetip") == 2

    @pytest.mark.asyncio
    async def test_discover_all_protocols(self, discovery, mock_handlers):
        """Test discovery with all registered protocols."""
        # Register handlers
        for protocol, handler in mock_handlers.items():
            discovery.register_protocol_handler(protocol, handler)
            handler.discover_devices.return_value = [
                DeviceInfo(
                    device_id=f"{protocol}-device",
                    protocol=protocol,
                    host="192.168.1.100",
                    port=502,
                    discovery_method="network_scan"
                )
            ]
        
        # Perform discovery without specifying protocols
        devices = await discovery.discover_devices(network_range="192.168.1.0/24")
        
        assert len(devices) == 3
        assert set(d.protocol for d in devices) == {"modbus", "ethernetip", "opcua"}

    @pytest.mark.asyncio
    async def test_discovery_with_timeout(self, discovery, mock_handlers):
        """Test discovery with timeout."""
        # Create a handler that takes too long
        slow_handler = MockProtocolHandler("slow")
        
        async def slow_discover(*args, **kwargs):
            await asyncio.sleep(5)  # Longer than timeout
            return []
        
        slow_handler.discover_devices = slow_discover
        discovery.register_protocol_handler("slow", slow_handler)
        
        # Perform discovery with short timeout
        devices = await discovery.discover_devices(
            protocols=["slow"],
            network_range="192.168.1.0/24",
            timeout=0.1
        )
        
        # Should return empty due to timeout
        assert len(devices) == 0

    @pytest.mark.asyncio
    async def test_discovery_error_handling(self, discovery, mock_handlers):
        """Test error handling during discovery."""
        # Register handlers with one that fails
        discovery.register_protocol_handler("modbus", mock_handlers["modbus"])
        discovery.register_protocol_handler("ethernetip", mock_handlers["ethernetip"])
        
        # Set up one to succeed and one to fail
        mock_handlers["modbus"].discover_devices.return_value = [
            DeviceInfo(
                device_id="modbus-device",
                protocol="modbus",
                host="192.168.1.100",
                port=502,
                discovery_method="network_scan"
            )
        ]
        mock_handlers["ethernetip"].discover_devices.side_effect = Exception("Discovery failed")
        
        # Perform discovery
        devices = await discovery.discover_devices(
            protocols=["modbus", "ethernetip"],
            network_range="192.168.1.0/24"
        )
        
        # Should still return successful discoveries
        assert len(devices) == 1
        assert devices[0].protocol == "modbus"

    @pytest.mark.asyncio
    async def test_event_emission(self, discovery, mock_handlers, event_bus):
        """Test that discovery emits events."""
        events_received = []
        
        async def event_handler(event):
            events_received.append(event)
        
        event_bus.subscribe(EventType.DEVICE_DISCOVERED, event_handler)
        
        # Register handler
        discovery.register_protocol_handler("modbus", mock_handlers["modbus"])
        mock_handlers["modbus"].discover_devices.return_value = [
            DeviceInfo(
                device_id="modbus-device",
                protocol="modbus",
                host="192.168.1.100",
                port=502,
                discovery_method="network_scan"
            )
        ]
        
        # Perform discovery
        await discovery.discover_devices(
            protocols=["modbus"],
            network_range="192.168.1.0/24"
        )
        
        # Give event bus time to process
        await asyncio.sleep(0.1)
        
        # Verify event was emitted
        assert len(events_received) == 1
        assert events_received[0].type == EventType.DEVICE_DISCOVERED
        assert events_received[0].data.device_id == "modbus-device"

    @pytest.mark.asyncio
    async def test_discovery_methods(self, discovery, mock_handlers):
        """Test different discovery methods."""
        discovery.register_protocol_handler("modbus", mock_handlers["modbus"])
        
        # Test network scan
        await discovery.discover_devices(
            protocols=["modbus"],
            network_range="192.168.1.0/24",
            method=DiscoveryMethod.NETWORK_SCAN
        )
        
        mock_handlers["modbus"].discover_devices.assert_called_with(
            network_range="192.168.1.0/24",
            method=DiscoveryMethod.NETWORK_SCAN,
            options={}
        )
        
        # Reset mock
        mock_handlers["modbus"].discover_devices.reset_mock()
        
        # Test broadcast
        await discovery.discover_devices(
            protocols=["modbus"],
            method=DiscoveryMethod.BROADCAST
        )
        
        mock_handlers["modbus"].discover_devices.assert_called_with(
            network_range=None,
            method=DiscoveryMethod.BROADCAST,
            options={}
        )

    @pytest.mark.asyncio
    async def test_discovery_with_options(self, discovery, mock_handlers):
        """Test discovery with custom options."""
        discovery.register_protocol_handler("modbus", mock_handlers["modbus"])
        
        custom_options = {
            "port_range": [502, 503, 504],
            "timeout": 2,
            "max_devices": 10
        }
        
        await discovery.discover_devices(
            protocols=["modbus"],
            network_range="192.168.1.0/24",
            options=custom_options
        )
        
        mock_handlers["modbus"].discover_devices.assert_called_with(
            network_range="192.168.1.0/24",
            method=DiscoveryMethod.NETWORK_SCAN,
            options=custom_options
        )

    @pytest.mark.asyncio
    async def test_concurrent_protocol_discovery(self, discovery, mock_handlers):
        """Test that multiple protocols are discovered concurrently."""
        # Register handlers with delays
        for protocol, handler in mock_handlers.items():
            discovery.register_protocol_handler(protocol, handler)
            
            async def delayed_discover(proto=protocol, *args, **kwargs):
                await asyncio.sleep(0.1)  # Simulate network delay
                return [
                    DeviceInfo(
                        device_id=f"{proto}-device",
                        protocol=proto,
                        host="192.168.1.100",
                        port=502,
                        discovery_method="network_scan"
                    )
                ]
            
            handler.discover_devices = AsyncMock(side_effect=delayed_discover)
        
        # Measure time for concurrent discovery
        import time
        start_time = time.time()
        
        devices = await discovery.discover_devices(
            protocols=["modbus", "ethernetip", "opcua"],
            network_range="192.168.1.0/24"
        )
        
        elapsed_time = time.time() - start_time
        
        # Should complete in ~0.1s (concurrent) not ~0.3s (sequential)
        assert elapsed_time < 0.2
        assert len(devices) == 3

    @pytest.mark.asyncio
    async def test_duplicate_device_handling(self, discovery, mock_handlers):
        """Test handling of duplicate devices from different protocols."""
        # Register handlers that return same device
        for protocol in ["modbus", "ethernetip"]:
            discovery.register_protocol_handler(protocol, mock_handlers[protocol])
            mock_handlers[protocol].discover_devices.return_value = [
                DeviceInfo(
                    device_id="same-device",  # Same ID
                    protocol=protocol,
                    host="192.168.1.100",
                    port=502,
                    discovery_method="network_scan"
                )
            ]
        
        devices = await discovery.discover_devices(
            protocols=["modbus", "ethernetip"],
            network_range="192.168.1.0/24"
        )
        
        # Should return both devices even with same ID (different protocols)
        assert len(devices) == 2

    @pytest.mark.asyncio
    async def test_stop_discovery(self, discovery, mock_handlers):
        """Test stopping discovery service."""
        # Register handler
        discovery.register_protocol_handler("modbus", mock_handlers["modbus"])
        
        # Start a long-running discovery
        async def long_discover(*args, **kwargs):
            await asyncio.sleep(5)
            return []
        
        mock_handlers["modbus"].discover_devices = long_discover
        
        # Start discovery in background
        discovery_task = asyncio.create_task(
            discovery.discover_devices(
                protocols=["modbus"],
                network_range="192.168.1.0/24"
            )
        )
        
        # Stop discovery service
        await asyncio.sleep(0.1)  # Let discovery start
        await discovery.stop()
        
        # Task should be cancelled
        with pytest.raises(asyncio.CancelledError):
            await discovery_task

    @pytest.mark.asyncio
    async def test_discovery_result_validation(self, discovery, mock_handlers):
        """Test validation of discovery results."""
        discovery.register_protocol_handler("modbus", mock_handlers["modbus"])
        
        # Return invalid device info (missing required fields)
        mock_handlers["modbus"].discover_devices.return_value = [
            {"invalid": "data"},  # Not a DeviceInfo object
            None,  # None value
            DeviceInfo(
                device_id="",  # Empty ID
                protocol="modbus",
                host="192.168.1.100",
                port=502,
                discovery_method="network_scan"
            )
        ]
        
        devices = await discovery.discover_devices(
            protocols=["modbus"],
            network_range="192.168.1.0/24"
        )
        
        # Should filter out invalid devices
        assert len(devices) == 0

    @pytest.mark.asyncio
    async def test_discovery_caching(self, discovery, mock_handlers):
        """Test caching of discovery results."""
        discovery.register_protocol_handler("modbus", mock_handlers["modbus"])
        mock_handlers["modbus"].discover_devices.return_value = [
            DeviceInfo(
                device_id="modbus-device",
                protocol="modbus",
                host="192.168.1.100",
                port=502,
                discovery_method="network_scan"
            )
        ]
        
        # Enable caching
        discovery.enable_caching(ttl=60)
        
        # First discovery
        devices1 = await discovery.discover_devices(
            protocols=["modbus"],
            network_range="192.168.1.0/24"
        )
        
        # Second discovery (should use cache)
        devices2 = await discovery.discover_devices(
            protocols=["modbus"],
            network_range="192.168.1.0/24"
        )
        
        # Handler should only be called once
        assert mock_handlers["modbus"].discover_devices.call_count == 1
        assert devices1 == devices2

    @pytest.mark.asyncio
    async def test_manual_device_addition(self, discovery, event_bus):
        """Test manually adding devices."""
        events_received = []
        
        async def event_handler(event):
            events_received.append(event)
        
        event_bus.subscribe(EventType.DEVICE_DISCOVERED, event_handler)
        
        # Manually add device
        device_info = DeviceInfo(
            device_id="manual-device",
            protocol="modbus",
            host="192.168.1.200",
            port=502,
            discovery_method="manual"
        )
        
        await discovery.add_device(device_info)
        
        # Give event bus time to process
        await asyncio.sleep(0.1)
        
        # Verify event was emitted
        assert len(events_received) == 1
        assert events_received[0].data.device_id == "manual-device"
        assert events_received[0].data.discovery_method == "manual"