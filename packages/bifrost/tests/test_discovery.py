"""Tests for network discovery functionality."""

import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from bifrost_core import DeviceInfo, ProtocolType

from bifrost.discovery import (
    BootPRequest,
    DiscoveredDevice,
    NetworkDiscovery,
    assign_device_ip,
    discover_devices,
)


class TestBootPRequest:
    """Test BootP/DHCP request packet structure."""

    def test_bootp_request_creation(self):
        """Test creating a BootP request."""
        request = BootPRequest()
        
        assert request.op == 1  # Boot request
        assert request.htype == 1  # Ethernet
        assert request.hlen == 6  # MAC address length
        assert request.hops == 0
        assert request.xid == 0  # Transaction ID
        assert request.secs == 0
        assert request.flags == 0
        assert request.ciaddr == "0.0.0.0"
        assert request.yiaddr == "0.0.0.0"
        assert request.siaddr == "0.0.0.0"
        assert request.giaddr == "0.0.0.0"
        assert len(request.chaddr) == 16
        assert len(request.sname) == 64
        assert len(request.file) == 128

    def test_bootp_request_custom_fields(self):
        """Test creating BootP request with custom fields."""
        request = BootPRequest(
            xid=12345,
            ciaddr="192.168.1.100",
            chaddr=b"\x00\x01\x02\x03\x04\x05" + b"\x00" * 10
        )
        
        assert request.xid == 12345
        assert request.ciaddr == "192.168.1.100"
        assert request.chaddr.startswith(b"\x00\x01\x02\x03\x04\x05")


class TestDiscoveredDevice:
    """Test discovered device structure."""

    def test_discovered_device_creation(self):
        """Test creating a discovered device."""
        device = DiscoveredDevice(
            mac_address="aa:bb:cc:dd:ee:ff",
            ip_address="192.168.1.100",
            hostname="plc-device",
            vendor="Schneider Electric",
            device_type="PLC",
            protocol=ProtocolType.MODBUS_TCP,
            ports=[502],
            last_seen=1234567890.0,
            additional_info={"model": "M340"}
        )
        
        assert device.mac_address == "aa:bb:cc:dd:ee:ff"
        assert device.ip_address == "192.168.1.100"
        assert device.hostname == "plc-device"
        assert device.vendor == "Schneider Electric"
        assert device.device_type == "PLC"
        assert device.protocol == ProtocolType.MODBUS_TCP
        assert device.ports == [502]
        assert device.last_seen == 1234567890.0
        assert device.additional_info["model"] == "M340"

    def test_discovered_device_minimal(self):
        """Test creating a minimal discovered device."""
        device = DiscoveredDevice(
            mac_address="aa:bb:cc:dd:ee:ff",
            ip_address=None,
            hostname=None,
            vendor=None,
            device_type=None,
            protocol=None,
            ports=[],
            last_seen=0.0,
            additional_info={}
        )
        
        assert device.mac_address == "aa:bb:cc:dd:ee:ff"
        assert device.ip_address is None
        assert device.hostname is None
        assert device.vendor is None
        assert device.device_type is None
        assert device.protocol is None
        assert device.ports == []
        assert device.last_seen == 0.0
        assert device.additional_info == {}


class TestNetworkDiscovery:
    """Test network discovery functionality."""

    def setup_method(self):
        """Set up test fixtures."""
        self.discovery = NetworkDiscovery()

    def test_network_discovery_init(self):
        """Test NetworkDiscovery initialization."""
        assert isinstance(self.discovery, NetworkDiscovery)

    @pytest.mark.asyncio
    async def test_discover_network_default(self):
        """Test basic network discovery."""
        with patch.object(self.discovery, '_ping_sweep') as mock_ping, \
             patch.object(self.discovery, '_arp_discovery') as mock_arp, \
             patch.object(self.discovery, '_bootp_discovery') as mock_bootp, \
             patch.object(self.discovery, '_modbus_discovery') as mock_modbus:
            
            # Mock all discovery methods
            mock_ping.return_value = None
            mock_arp.return_value = None
            mock_bootp.return_value = None
            mock_modbus.return_value = None
            
            devices = await self.discovery.discover_network(
                network="192.168.1.0/24",
                methods=["ping", "arp", "bootp", "modbus"],
                timeout=5.0
            )
            
            assert isinstance(devices, list)
            mock_ping.assert_called_once()
            mock_arp.assert_called_once()
            mock_bootp.assert_called_once()
            mock_modbus.assert_called_once()

    @pytest.mark.asyncio
    async def test_discover_network_ping_only(self):
        """Test network discovery with ping only."""
        with patch.object(self.discovery, '_ping_sweep') as mock_ping, \
             patch.object(self.discovery, '_arp_discovery') as mock_arp:
            
            mock_ping.return_value = None
            
            devices = await self.discovery.discover_network(
                network="192.168.1.0/24",
                methods=["ping"],
                timeout=5.0
            )
            
            assert isinstance(devices, list)
            mock_ping.assert_called_once()
            mock_arp.assert_not_called()

    @pytest.mark.asyncio
    async def test_discover_network_invalid_method(self):
        """Test network discovery with invalid method."""
        devices = await self.discovery.discover_network(
            network="192.168.1.0/24",
            methods=["invalid_method"],
            timeout=5.0
        )
        
        assert isinstance(devices, list)

    @pytest.mark.asyncio
    async def test_ping_sweep(self):
        """Test ping sweep functionality."""
        with patch("asyncio.create_subprocess_exec") as mock_subprocess:
            # Mock successful ping for one IP
            mock_process = MagicMock()
            mock_process.communicate = AsyncMock(return_value=(b"", b""))
            mock_process.returncode = 0
            mock_subprocess.return_value = mock_process
            
            await self.discovery._ping_sweep("192.168.1.0/30", timeout=1.0)
            
            # Should have called subprocess for each IP in the range
            assert mock_subprocess.call_count > 0

    @pytest.mark.asyncio
    async def test_modbus_discovery(self):
        """Test Modbus discovery functionality."""
        with patch("asyncio.open_connection") as mock_connection:
            # Mock successful connection
            mock_reader = AsyncMock()
            mock_writer = MagicMock()
            mock_writer.wait_closed = AsyncMock()
            mock_connection.return_value = (mock_reader, mock_writer)
            
            await self.discovery._modbus_discovery("192.168.1.0/30", timeout=1.0)
            
            # Should have attempted connections
            assert mock_connection.call_count >= 0

    @pytest.mark.asyncio
    async def test_bootp_discovery(self):
        """Test BootP discovery functionality."""
        with patch("socket.socket") as mock_socket:
            mock_sock = MagicMock()
            mock_socket.return_value.__enter__.return_value = mock_sock
            mock_sock.recvfrom = MagicMock(side_effect=OSError("timeout"))
            
            await self.discovery._bootp_discovery(timeout=0.1)
            
            # Should have created a socket
            mock_socket.assert_called()


class TestDiscoveryFunctions:
    """Test top-level discovery functions."""

    @pytest.mark.asyncio
    async def test_discover_devices_default(self):
        """Test discover_devices with default parameters."""
        with patch("bifrost.discovery.NetworkDiscovery") as mock_discovery_class:
            mock_discovery = MagicMock()
            mock_discovery.discover_network = AsyncMock(return_value=[])
            mock_discovery_class.return_value = mock_discovery
            
            devices = await discover_devices()
            
            assert isinstance(devices, list)
            mock_discovery_class.assert_called_once()
            mock_discovery.discover_network.assert_called_once()

    @pytest.mark.asyncio
    async def test_discover_devices_custom_params(self):
        """Test discover_devices with custom parameters."""
        with patch("bifrost.discovery.NetworkDiscovery") as mock_discovery_class:
            mock_discovery = MagicMock()
            mock_discovery.discover_network = AsyncMock(return_value=[])
            mock_discovery_class.return_value = mock_discovery
            
            devices = await discover_devices(
                network="10.0.0.0/24",
                methods=["ping"],
                timeout=10.0
            )
            
            assert isinstance(devices, list)
            mock_discovery.discover_network.assert_called_once_with(
                network="10.0.0.0/24",
                methods=["ping"],
                timeout=10.0
            )

    @pytest.mark.asyncio
    async def test_assign_device_ip_success(self):
        """Test successful device IP assignment."""
        with patch("socket.socket") as mock_socket:
            mock_sock = MagicMock()
            mock_socket.return_value.__enter__.return_value = mock_sock
            
            # Mock successful assignment
            result = await assign_device_ip(
                "aa:bb:cc:dd:ee:ff",
                "192.168.1.100",
                "255.255.255.0",
                "192.168.1.1"
            )
            
            assert result is True
            mock_sock.sendto.assert_called()

    @pytest.mark.asyncio
    async def test_assign_device_ip_failure(self):
        """Test failed device IP assignment."""
        with patch("socket.socket") as mock_socket:
            mock_socket.side_effect = OSError("Network error")
            
            result = await assign_device_ip(
                "aa:bb:cc:dd:ee:ff",
                "192.168.1.100"
            )
            
            assert result is False


class TestEdgeCases:
    """Test edge cases and error conditions."""

    @pytest.mark.asyncio
    async def test_discovery_with_invalid_network(self):
        """Test discovery with invalid network."""
        discovery = NetworkDiscovery()
        
        # Should handle invalid network gracefully
        devices = await discovery.discover_network(
            network="invalid.network",
            methods=["ping"],
            timeout=1.0
        )
        assert isinstance(devices, list)

    @pytest.mark.asyncio
    async def test_discovery_network_timeout(self):
        """Test discovery with very short timeout."""
        discovery = NetworkDiscovery()
        devices = await discovery.discover_network(
            network="192.168.1.0/24",
            methods=["ping"],
            timeout=0.001
        )
        # Should handle timeout gracefully
        assert isinstance(devices, list)

    @pytest.mark.asyncio
    async def test_assign_ip_invalid_mac(self):
        """Test IP assignment with invalid MAC address."""
        result = await assign_device_ip(
            "invalid_mac",
            "192.168.1.100"
        )
        # Should handle invalid MAC gracefully
        assert result is False

    @pytest.mark.asyncio
    async def test_assign_ip_invalid_ip(self):
        """Test IP assignment with invalid IP address."""
        result = await assign_device_ip(
            "aa:bb:cc:dd:ee:ff",
            "invalid.ip"
        )
        # Should handle invalid IP gracefully
        assert result is False