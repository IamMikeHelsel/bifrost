"""Tests for DeviceManager service."""

import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from bifrost_core.base import BaseConnection, BaseDevice, DeviceInfo
from bifrost_core.events import EventBus, EventType
from bifrost_core.pooling import ConnectionPool
from bifrost_core.services.device_manager import DeviceManager
from bifrost_core.typing import DataType, Tag


class MockConnection(BaseConnection):
    """Mock connection for testing."""

    def __init__(self, host: str, port: int = 502):
        self.host = host
        self.port = port
        self._connected = False

    async def __aenter__(self):
        self._connected = True
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        self._connected = False

    @property
    def is_connected(self) -> bool:
        return self._connected

    async def read(self, tags):
        return {}

    async def write(self, values):
        pass

    async def get_info(self):
        return {"host": self.host, "port": self.port}


class MockDevice(BaseDevice):
    """Mock device for testing."""

    def __init__(self, connection: BaseConnection):
        super().__init__(connection)

    async def read(self, tags):
        return {}

    async def write(self, values):
        pass

    async def get_info(self):
        return {"type": "mock"}


class TestDeviceManager:
    """Test DeviceManager functionality."""

    @pytest.fixture
    def event_bus(self):
        """Create event bus for testing."""
        return EventBus()

    @pytest.fixture
    def device_info(self):
        """Create test device info."""
        return DeviceInfo(
            device_id="test-device-1",
            protocol="mock",
            host="192.168.1.100",
            port=502,
            discovery_method="manual",
            name="Test Device"
        )

    @pytest.fixture
    async def device_manager(self, event_bus):
        """Create DeviceManager instance."""
        manager = DeviceManager(event_bus)
        yield manager
        await manager.stop()

    @pytest.mark.asyncio
    async def test_register_device(self, device_manager, device_info):
        """Test registering a new device."""
        # Create mock connection pool
        mock_pool = AsyncMock(spec=ConnectionPool)
        mock_pool.get = AsyncMock(return_value=MockConnection("192.168.1.100"))
        
        with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
            with patch.object(device_manager, '_create_device', return_value=MockDevice(MockConnection("192.168.1.100"))):
                # Register device
                await device_manager.register_device(device_info)
                
                # Verify device is registered
                assert device_info.device_id in device_manager._devices
                assert device_info.device_id in device_manager._connection_pools
                device_manager._create_connection_pool.assert_called_once_with(device_info)

    @pytest.mark.asyncio
    async def test_register_duplicate_device(self, device_manager, device_info):
        """Test registering a duplicate device."""
        # Register device once
        mock_pool = AsyncMock(spec=ConnectionPool)
        with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
            with patch.object(device_manager, '_create_device', return_value=MockDevice(MockConnection("192.168.1.100"))):
                await device_manager.register_device(device_info)
                
                # Try to register again - should raise error
                with pytest.raises(ValueError, match="already registered"):
                    await device_manager.register_device(device_info)

    @pytest.mark.asyncio
    async def test_unregister_device(self, device_manager, device_info):
        """Test unregistering a device."""
        # Register device first
        mock_pool = AsyncMock(spec=ConnectionPool)
        mock_pool.close = AsyncMock()
        
        with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
            with patch.object(device_manager, '_create_device', return_value=MockDevice(MockConnection("192.168.1.100"))):
                await device_manager.register_device(device_info)
                
                # Unregister device
                await device_manager.unregister_device(device_info.device_id)
                
                # Verify device is removed
                assert device_info.device_id not in device_manager._devices
                assert device_info.device_id not in device_manager._connection_pools
                mock_pool.close.assert_called_once()

    @pytest.mark.asyncio
    async def test_unregister_nonexistent_device(self, device_manager):
        """Test unregistering a device that doesn't exist."""
        # Should not raise error
        await device_manager.unregister_device("nonexistent-device")

    @pytest.mark.asyncio
    async def test_get_device(self, device_manager, device_info):
        """Test getting a device."""
        # Register device
        mock_pool = AsyncMock(spec=ConnectionPool)
        mock_device = MockDevice(MockConnection("192.168.1.100"))
        
        with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
            with patch.object(device_manager, '_create_device', return_value=mock_device):
                await device_manager.register_device(device_info)
                
                # Get device
                device = await device_manager.get_device(device_info.device_id)
                assert device == mock_device

    @pytest.mark.asyncio
    async def test_get_nonexistent_device(self, device_manager):
        """Test getting a device that doesn't exist."""
        with pytest.raises(KeyError):
            await device_manager.get_device("nonexistent-device")

    @pytest.mark.asyncio
    async def test_list_devices(self, device_manager):
        """Test listing all devices."""
        # Register multiple devices
        devices = []
        for i in range(3):
            device_info = DeviceInfo(
                device_id=f"test-device-{i}",
                protocol="mock",
                host=f"192.168.1.{100 + i}",
                port=502,
                discovery_method="manual"
            )
            devices.append(device_info)
            
            mock_pool = AsyncMock(spec=ConnectionPool)
            with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
                with patch.object(device_manager, '_create_device', return_value=MockDevice(MockConnection(device_info.host))):
                    await device_manager.register_device(device_info)
        
        # List devices
        device_list = device_manager.list_devices()
        assert len(device_list) == 3
        assert all(d.device_id in [dev.device_id for dev in devices] for d in device_list)

    @pytest.mark.asyncio
    async def test_read_from_device(self, device_manager, device_info):
        """Test reading from a device."""
        # Create mock device with read capability
        mock_device = MockDevice(MockConnection("192.168.1.100"))
        mock_device.read = AsyncMock(return_value={"tag1": 123})
        
        mock_pool = AsyncMock(spec=ConnectionPool)
        with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
            with patch.object(device_manager, '_create_device', return_value=mock_device):
                await device_manager.register_device(device_info)
                
                # Read from device
                tags = [Tag(name="tag1", address="40001", data_type=DataType.INT16)]
                result = await device_manager.read_from_device(device_info.device_id, tags)
                
                assert result == {"tag1": 123}
                mock_device.read.assert_called_once_with(tags)

    @pytest.mark.asyncio
    async def test_write_to_device(self, device_manager, device_info):
        """Test writing to a device."""
        # Create mock device with write capability
        mock_device = MockDevice(MockConnection("192.168.1.100"))
        mock_device.write = AsyncMock()
        
        mock_pool = AsyncMock(spec=ConnectionPool)
        with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
            with patch.object(device_manager, '_create_device', return_value=mock_device):
                await device_manager.register_device(device_info)
                
                # Write to device
                tag = Tag(name="tag1", address="40001", data_type=DataType.INT16)
                values = {tag: 456}
                await device_manager.write_to_device(device_info.device_id, values)
                
                mock_device.write.assert_called_once_with(values)

    @pytest.mark.asyncio
    async def test_get_device_info(self, device_manager, device_info):
        """Test getting device info."""
        # Create mock device with info
        mock_device = MockDevice(MockConnection("192.168.1.100"))
        mock_device.get_info = AsyncMock(return_value={"status": "online"})
        
        mock_pool = AsyncMock(spec=ConnectionPool)
        with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
            with patch.object(device_manager, '_create_device', return_value=mock_device):
                await device_manager.register_device(device_info)
                
                # Get device info
                info = await device_manager.get_device_info(device_info.device_id)
                
                assert info == {"status": "online"}
                mock_device.get_info.assert_called_once()

    @pytest.mark.asyncio
    async def test_event_emission_on_register(self, device_manager, device_info, event_bus):
        """Test that events are emitted when registering devices."""
        # Set up event listener
        events_received = []
        
        async def event_handler(event):
            events_received.append(event)
        
        event_bus.subscribe(EventType.DEVICE_DISCOVERED, event_handler)
        
        # Register device
        mock_pool = AsyncMock(spec=ConnectionPool)
        with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
            with patch.object(device_manager, '_create_device', return_value=MockDevice(MockConnection("192.168.1.100"))):
                await device_manager.register_device(device_info)
                
                # Give event bus time to process
                await asyncio.sleep(0.1)
                
                # Verify event was emitted
                assert len(events_received) == 1
                assert events_received[0].type == EventType.DEVICE_DISCOVERED
                assert events_received[0].data == device_info

    @pytest.mark.asyncio
    async def test_stop_closes_all_pools(self, device_manager):
        """Test that stopping the manager closes all connection pools."""
        # Register multiple devices
        pools = []
        for i in range(3):
            device_info = DeviceInfo(
                device_id=f"test-device-{i}",
                protocol="mock",
                host=f"192.168.1.{100 + i}",
                port=502,
                discovery_method="manual"
            )
            
            mock_pool = AsyncMock(spec=ConnectionPool)
            mock_pool.close = AsyncMock()
            pools.append(mock_pool)
            
            with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
                with patch.object(device_manager, '_create_device', return_value=MockDevice(MockConnection(device_info.host))):
                    await device_manager.register_device(device_info)
        
        # Stop the manager
        await device_manager.stop()
        
        # Verify all pools were closed
        for pool in pools:
            pool.close.assert_called_once()
        
        # Verify all devices were removed
        assert len(device_manager._devices) == 0
        assert len(device_manager._connection_pools) == 0

    @pytest.mark.asyncio
    async def test_concurrent_device_operations(self, device_manager):
        """Test concurrent operations on multiple devices."""
        # Register multiple devices
        device_infos = []
        for i in range(5):
            device_info = DeviceInfo(
                device_id=f"test-device-{i}",
                protocol="mock",
                host=f"192.168.1.{100 + i}",
                port=502,
                discovery_method="manual"
            )
            device_infos.append(device_info)
            
            mock_pool = AsyncMock(spec=ConnectionPool)
            mock_device = MockDevice(MockConnection(device_info.host))
            mock_device.read = AsyncMock(return_value={f"tag{i}": i * 100})
            
            with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
                with patch.object(device_manager, '_create_device', return_value=mock_device):
                    await device_manager.register_device(device_info)
        
        # Perform concurrent reads
        tags = [Tag(name=f"tag{i}", address=f"4000{i}", data_type=DataType.INT16) for i in range(5)]
        
        read_tasks = [
            device_manager.read_from_device(info.device_id, [tags[i]])
            for i, info in enumerate(device_infos)
        ]
        
        results = await asyncio.gather(*read_tasks)
        
        # Verify results
        for i, result in enumerate(results):
            assert result == {f"tag{i}": i * 100}

    @pytest.mark.asyncio
    async def test_connection_pool_configuration(self, device_manager, device_info):
        """Test that connection pools are created with correct configuration."""
        # Spy on connection pool creation
        mock_pool = AsyncMock(spec=ConnectionPool)
        
        with patch('bifrost_core.services.device_manager.ConnectionPool', return_value=mock_pool) as pool_class:
            with patch.object(device_manager, '_create_device', return_value=MockDevice(MockConnection("192.168.1.100"))):
                await device_manager.register_device(device_info)
                
                # Verify pool was created with correct parameters
                pool_class.assert_called_once()
                call_kwargs = pool_class.call_args.kwargs
                assert 'connection_factory' in call_kwargs
                assert call_kwargs['max_size'] == 10  # Default max size

    @pytest.mark.asyncio
    async def test_device_creation_error_handling(self, device_manager, device_info):
        """Test error handling when device creation fails."""
        mock_pool = AsyncMock(spec=ConnectionPool)
        
        with patch.object(device_manager, '_create_connection_pool', return_value=mock_pool):
            with patch.object(device_manager, '_create_device', side_effect=Exception("Device creation failed")):
                with pytest.raises(Exception, match="Device creation failed"):
                    await device_manager.register_device(device_info)
                
                # Verify device was not registered
                assert device_info.device_id not in device_manager._devices