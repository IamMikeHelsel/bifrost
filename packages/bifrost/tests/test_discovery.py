"""Tests for network discovery functionality."""

from unittest.mock import AsyncMock, patch

import pytest

from bifrost.discovery import discover_devices


class TestDiscoveryFunctions:
    """Test top-level discovery functions."""

    @pytest.mark.asyncio
    async def test_discover_devices_default(self):
        """Test discover_devices with default parameters."""
        with patch(
            "bifrost.discovery.discover_devices"
        ) as mock_discover_devices:
            mock_discover_devices.return_value = AsyncMock(
                return_value=[
                    {
                        "host": "192.168.1.10",
                        "port": 502,
                        "protocol": "modbus.tcp",
                        "device_type": "PLC",
                    }
                ]
            )
            devices = []
            async for device in discover_devices():
                devices.append(device)

            assert len(devices) > 0
            assert devices[0]["host"] == "192.168.1.10"
            assert devices[0]["protocol"] == "modbus.tcp"
