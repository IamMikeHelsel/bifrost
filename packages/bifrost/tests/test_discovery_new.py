"""Tests for device discovery functionality."""

import pytest
from unittest.mock import AsyncMock, patch
from bifrost.discovery import DiscoveryConfig, discover_modbus_devices
from bifrost_core.base import DeviceInfo


@pytest.mark.asyncio
async def test_discovery_config():
    """Test DiscoveryConfig initialization."""
    config = DiscoveryConfig(
        network_range="192.168.1.0/24",
        timeout=5.0,
        max_concurrent=100,
        protocols=["modbus", "cip"]
    )
    
    assert config.network_range == "192.168.1.0/24"
    assert config.timeout == 5.0
    assert config.max_concurrent == 100
    assert config.protocols == ["modbus", "cip"]


@pytest.mark.asyncio
async def test_discovery_config_defaults():
    """Test DiscoveryConfig default values."""
    config = DiscoveryConfig()
    
    assert config.network_range == "192.168.1.0/24"
    assert config.timeout == 2.0
    assert config.max_concurrent == 50
    assert config.protocols == ("modbus", "cip", "bootp")


@pytest.mark.asyncio
async def test_modbus_discovery_no_devices():
    """Test Modbus discovery with no devices found."""
    config = DiscoveryConfig(network_range="127.0.0.1/32", timeout=0.1)
    
    devices = []
    async for device in discover_modbus_devices(config):
        devices.append(device)
    
    # Should find no devices on localhost
    assert len(devices) == 0


@pytest.mark.asyncio
async def test_device_info_model():
    """Test DeviceInfo model creation and validation."""
    device = DeviceInfo(
        device_id="test_device",
        protocol="modbus.tcp",
        host="192.168.1.100",
        port=502,
        discovery_method="modbus",
        device_type="PLC",
        confidence=0.9
    )
    
    assert device.device_id == "test_device"
    assert device.protocol == "modbus.tcp"
    assert device.host == "192.168.1.100"
    assert device.port == 502
    assert device.discovery_method == "modbus"
    assert device.device_type == "PLC"
    assert device.confidence == 0.9
    assert device.name == "test_device"  # Should be set by validator


@pytest.mark.asyncio
async def test_device_info_model_with_name():
    """Test DeviceInfo model with explicit name."""
    device = DeviceInfo(
        device_id="test_device",
        protocol="modbus.tcp",
        host="192.168.1.100",
        port=502,
        discovery_method="modbus",
        name="Custom Name"
    )
    
    assert device.name == "Custom Name"  # Should keep explicit name