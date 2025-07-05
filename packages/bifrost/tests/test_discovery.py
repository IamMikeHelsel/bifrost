"""Tests for network discovery functionality."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from bifrost.discovery import DiscoveryConfig, discover_devices
from bifrost_core.base import DeviceInfo


class TestDiscoveryFunctions:
    """Test top-level discovery functions."""

    @pytest.mark.asyncio
    async def test_discover_devices_default(self):
        """Test discover_devices with default parameters."""
        # Create a mock DeviceInfo object
        mock_device = DeviceInfo(
            device_id="modbus_192.168.1.10",
            host="192.168.1.10",
            port=502,
            protocol="modbus.tcp",
            device_type="PLC",
            discovery_method="modbus",
            confidence=0.95,
        )
        
        # Mock the individual discovery functions
        with patch("bifrost.discovery.discover_modbus_devices") as mock_modbus:
            # Create an async generator that yields our mock device
            async def mock_generator():
                yield mock_device
            
            mock_modbus.return_value = mock_generator()
            
            # Test with modbus protocol only
            config = DiscoveryConfig(protocols=["modbus"])
            devices = []
            async for device in discover_devices(config):
                devices.append(device)

            assert len(devices) == 1
            assert devices[0].host == "192.168.1.10"
            assert devices[0].protocol == "modbus.tcp"
            assert devices[0].device_type == "PLC"
