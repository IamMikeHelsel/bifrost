"""Tests for network discovery functionality."""

from unittest.mock import patch

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

    @pytest.mark.asyncio
    async def test_discover_devices_unsupported_protocol(self):
        """Test discover_devices with an unsupported protocol."""
        config = DiscoveryConfig(protocols=["unsupported_protocol"])
        devices = []
        async for device in discover_devices(config):
            devices.append(device)
        assert len(devices) == 0

    @pytest.mark.asyncio
    async def test_discover_devices_no_results(self):
        """Test discover_devices when no devices are found."""
        with patch("bifrost.discovery.discover_modbus_devices") as mock_modbus:

            async def empty_generator():
                if False:
                    yield  # This makes it an async generator

            mock_modbus.return_value = empty_generator()

            config = DiscoveryConfig(protocols=["modbus"])
            devices = []
            async for device in discover_devices(config):
                devices.append(device)
            assert len(devices) == 0
