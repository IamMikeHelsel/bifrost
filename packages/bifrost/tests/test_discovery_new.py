"""Tests for device discovery functionality."""

import socket
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from bifrost.discovery import (
    DiscoveryConfig,
    discover_bootp_devices,
    discover_cip_devices,
    discover_modbus_devices,
)
from bifrost_core.base import DeviceInfo


@pytest.mark.asyncio
async def test_discovery_config():
    """Test DiscoveryConfig initialization."""
    config = DiscoveryConfig(
        network_range="192.168.1.0/24",
        timeout=5.0,
        max_concurrent=100,
        protocols=["modbus", "cip"],
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
        confidence=0.9,
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
        name="Custom Name",
    )

    assert device.name == "Custom Name"  # Should keep explicit name


@pytest.mark.asyncio
async def test_modbus_discovery_success():
    """Test successful Modbus discovery."""
    config = DiscoveryConfig(network_range="127.0.0.1/32", timeout=0.1)

    mock_reader = AsyncMock()
    mock_writer = AsyncMock()

    # Mock the response for Read Device Identification (simplified)
    # Transaction ID (2), Protocol ID (2), Length (6), Unit ID (1), Function (1), MEI Type (1), Read Code (1), Object ID (1)
    mock_reader.read.return_value = (
        b"\x00\x01\x00\x00\x00\x06\x01\x2b\x0e\x01\x00\x00"
    )

    with patch(
        "asyncio.open_connection", return_value=(mock_reader, mock_writer)
    ) as mock_open_connection:
        devices = []
        async for device in discover_modbus_devices(config):
            devices.append(device)

        mock_open_connection.assert_called_once_with("127.0.0.1", 502)
        assert len(devices) == 1
        assert devices[0].host == "127.0.0.1"
        assert devices[0].protocol == "modbus.tcp"
        assert devices[0].device_type == "Modbus Device"


@pytest.mark.asyncio
async def test_bootp_discovery_success():
    """Test successful BOOTP discovery."""
    config = DiscoveryConfig(network_range="192.168.1.0/24", timeout=0.1)

    mock_socket = MagicMock()
    mock_socket.recvfrom.side_effect = [
        (b"\x01" * 300, ("192.168.1.10", 67)),
        socket.timeout,
    ]  # Simplified BOOTP response, then timeout

    with patch("socket.socket", return_value=mock_socket) as mock_sock_init:
        devices = []
        async for device in discover_bootp_devices(config):
            devices.append(device)

        mock_sock_init.assert_called_once_with(
            socket.AF_INET, socket.SOCK_DGRAM
        )
        mock_socket.sendto.assert_called_once()
        assert len(devices) == 1
        assert devices[0].host == "192.168.1.10"
        assert devices[0].protocol == "bootp"


@pytest.mark.asyncio
async def test_cip_discovery_success():
    """Test successful CIP discovery."""
    config = DiscoveryConfig(network_range="192.168.1.0/24", timeout=0.1)

    mock_socket = MagicMock()
    mock_socket.recvfrom.side_effect = [
        (b"\x63\x00" + b"\x00" * 22, ("192.168.1.20", 44818)),
        socket.timeout,
    ]  # Simplified CIP response, then timeout

    with patch("socket.socket", return_value=mock_socket) as mock_sock_init:
        devices = []
        async for device in discover_cip_devices(config):
            devices.append(device)

        mock_sock_init.assert_called_once_with(
            socket.AF_INET, socket.SOCK_DGRAM
        )
        assert mock_socket.sendto.call_count == 2  # Multicast and broadcast
        assert len(devices) == 1
        assert devices[0].host == "192.168.1.20"
        assert devices[0].protocol == "ethernet_ip"
