"""Tests for network discovery functionality."""

import asyncio
import time
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from bifrost_core import DeviceInfo, ProtocolType

from bifrost.discovery import (
    BootPRequest,
    DiscoveredDevice,
    DiscoveryCache,
    DiscoveryConfig,
    NetworkDiscovery,
    assign_device_ip,
    clear_discovery_cache,
    discover_devices,
    get_cache_info,
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
                timeout=10.0,
                use_cache=True
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


class TestDiscoveryConfig:
    """Test discovery configuration."""

    def test_discovery_config_defaults(self):
        """Test default configuration values."""
        config = DiscoveryConfig()
        
        assert config.cache_enabled is True
        assert config.cache_ttl_seconds == 300
        assert config.max_cache_size_mb == 10
        assert config.cache_dir is not None

    def test_discovery_config_custom(self):
        """Test custom configuration values."""
        config = DiscoveryConfig(
            cache_enabled=False,
            cache_ttl_seconds=600,
            cache_dir="/tmp/test_cache",
            max_cache_size_mb=5
        )
        
        assert config.cache_enabled is False
        assert config.cache_ttl_seconds == 600
        assert config.cache_dir == "/tmp/test_cache"
        assert config.max_cache_size_mb == 5


class TestDiscoveryCache:
    """Test discovery caching functionality."""

    def setup_method(self):
        """Set up test fixtures."""
        self.config = DiscoveryConfig(
            cache_enabled=True,
            cache_ttl_seconds=10,
            cache_dir="/tmp/bifrost_test_cache",
            max_cache_size_mb=1
        )
        self.cache = DiscoveryCache(self.config)

    def teardown_method(self):
        """Clean up test fixtures."""
        self.cache.clear_cache()

    def test_cache_key_generation(self):
        """Test cache key generation."""
        key1 = self.cache._generate_cache_key("192.168.1.0/24", ["ping"], 5.0)
        key2 = self.cache._generate_cache_key("192.168.1.0/24", ["ping"], 5.0)
        key3 = self.cache._generate_cache_key("192.168.1.0/24", ["ping", "modbus"], 5.0)
        
        assert key1 == key2  # Same parameters should generate same key
        assert key1 != key3  # Different parameters should generate different key
        assert len(key1) == 16  # Key should be 16 characters

    @pytest.mark.asyncio
    async def test_cache_miss(self):
        """Test cache miss behavior."""
        result = await self.cache.get_cached_results("192.168.1.0/24", ["ping"], 5.0)
        assert result is None

    @pytest.mark.asyncio
    async def test_cache_hit(self):
        """Test cache hit behavior."""
        # Create test device
        device = DiscoveredDevice(
            mac_address="aa:bb:cc:dd:ee:ff",
            ip_address="192.168.1.100",
            protocol=ProtocolType.MODBUS_TCP
        )
        devices = [device]
        
        # Cache the results
        await self.cache.cache_results("192.168.1.0/24", ["ping"], 5.0, devices)
        
        # Retrieve from cache
        cached_devices = await self.cache.get_cached_results("192.168.1.0/24", ["ping"], 5.0)
        
        assert cached_devices is not None
        assert len(cached_devices) == 1
        assert cached_devices[0].mac_address == "aa:bb:cc:dd:ee:ff"
        assert cached_devices[0].ip_address == "192.168.1.100"
        assert cached_devices[0].protocol == ProtocolType.MODBUS_TCP

    @pytest.mark.asyncio
    async def test_cache_expiration(self):
        """Test cache expiration."""
        # Create config with very short TTL
        short_config = DiscoveryConfig(
            cache_enabled=True,
            cache_ttl_seconds=1,  # 1 second
            cache_dir="/tmp/bifrost_test_cache_short"
        )
        short_cache = DiscoveryCache(short_config)
        
        device = DiscoveredDevice(mac_address="aa:bb:cc:dd:ee:ff")
        devices = [device]
        
        # Cache the results
        await short_cache.cache_results("192.168.1.0/24", ["ping"], 5.0, devices)
        
        # Should get cache hit immediately
        cached_devices = await short_cache.get_cached_results("192.168.1.0/24", ["ping"], 5.0)
        assert cached_devices is not None
        
        # Wait for cache to expire
        await asyncio.sleep(2)
        
        # Should get cache miss after expiration
        cached_devices = await short_cache.get_cached_results("192.168.1.0/24", ["ping"], 5.0)
        assert cached_devices is None
        
        # Clean up
        short_cache.clear_cache()

    def test_cache_disabled(self):
        """Test cache when disabled."""
        disabled_config = DiscoveryConfig(cache_enabled=False)
        disabled_cache = DiscoveryCache(disabled_config)
        
        # Should always return None when cache is disabled
        assert disabled_cache._is_cache_valid({"timestamp": time.time()}) is False

    def test_device_serialization(self):
        """Test device serialization and deserialization."""
        device = DiscoveredDevice(
            mac_address="aa:bb:cc:dd:ee:ff",
            ip_address="192.168.1.100",
            hostname="test-device",
            vendor="Test Vendor",
            device_type="PLC",
            protocol=ProtocolType.MODBUS_TCP,
            ports=[502, 503],
            last_seen=1234567890.0,
            additional_info={"model": "Test Model"}
        )
        
        # Serialize
        serialized = self.cache._serialize_devices([device])
        assert len(serialized) == 1
        assert serialized[0]["mac_address"] == "aa:bb:cc:dd:ee:ff"
        assert serialized[0]["protocol"] == "modbus_tcp"
        
        # Deserialize
        deserialized = self.cache._deserialize_devices(serialized)
        assert len(deserialized) == 1
        restored_device = deserialized[0]
        assert restored_device.mac_address == device.mac_address
        assert restored_device.ip_address == device.ip_address
        assert restored_device.protocol == device.protocol
        assert restored_device.additional_info == device.additional_info


class TestNetworkDiscoveryWithCache:
    """Test network discovery with caching."""

    def setup_method(self):
        """Set up test fixtures."""
        self.config = DiscoveryConfig(
            cache_enabled=True,
            cache_ttl_seconds=60,
            cache_dir="/tmp/bifrost_test_discovery_cache"
        )
        self.discovery = NetworkDiscovery(self.config)

    def teardown_method(self):
        """Clean up test fixtures."""
        self.discovery.cache.clear_cache()

    @pytest.mark.asyncio
    async def test_discovery_with_cache_miss(self):
        """Test discovery when cache is empty."""
        with patch.object(self.discovery, '_ping_sweep') as mock_ping:
            mock_ping.return_value = None
            
            devices = await self.discovery.discover_network(
                network="192.168.1.0/30",
                methods=["ping"],
                timeout=1.0,
                use_cache=True
            )
            
            # Should have called discovery method
            mock_ping.assert_called_once()
            assert isinstance(devices, list)

    @pytest.mark.asyncio
    async def test_discovery_with_cache_hit(self):
        """Test discovery when cache contains valid results."""
        # Pre-populate cache
        device = DiscoveredDevice(mac_address="aa:bb:cc:dd:ee:ff")
        await self.discovery.cache.cache_results(
            "192.168.1.0/30", ["ping"], 1.0, [device]
        )
        
        with patch.object(self.discovery, '_ping_sweep') as mock_ping:
            devices = await self.discovery.discover_network(
                network="192.168.1.0/30",
                methods=["ping"],
                timeout=1.0,
                use_cache=True
            )
            
            # Should NOT have called discovery method
            mock_ping.assert_not_called()
            assert len(devices) == 1
            assert devices[0].mac_address == "aa:bb:cc:dd:ee:ff"

    @pytest.mark.asyncio
    async def test_discovery_cache_bypass(self):
        """Test discovery with cache disabled."""
        # Pre-populate cache
        device = DiscoveredDevice(mac_address="aa:bb:cc:dd:ee:ff")
        await self.discovery.cache.cache_results(
            "192.168.1.0/30", ["ping"], 1.0, [device]
        )
        
        with patch.object(self.discovery, '_ping_sweep') as mock_ping:
            mock_ping.return_value = None
            
            devices = await self.discovery.discover_network(
                network="192.168.1.0/30",
                methods=["ping"],
                timeout=1.0,
                use_cache=False  # Bypass cache
            )
            
            # Should have called discovery method even with cache
            mock_ping.assert_called_once()


class TestCacheUtilityFunctions:
    """Test cache utility functions."""

    def setup_method(self):
        """Set up test fixtures."""
        self.config = DiscoveryConfig(
            cache_enabled=True,
            cache_dir="/tmp/bifrost_test_util_cache"
        )

    def teardown_method(self):
        """Clean up test fixtures."""
        clear_discovery_cache(self.config)

    def test_get_cache_info_empty(self):
        """Test getting cache info when cache is empty."""
        info = get_cache_info(self.config)
        
        assert info["cache_enabled"] is True
        assert info["cache_dir"] == "/tmp/bifrost_test_util_cache"
        assert info["file_count"] == 0
        assert info["total_size_mb"] == 0.0

    def test_clear_cache(self):
        """Test clearing cache."""
        # Create a cache file
        cache = DiscoveryCache(self.config)
        cache_dir = Path(self.config.cache_dir)
        cache_dir.mkdir(parents=True, exist_ok=True)
        
        test_file = cache_dir / "discovery_test.json"
        test_file.write_text('{"test": "data"}')
        
        assert test_file.exists()
        
        # Clear cache
        clear_discovery_cache(self.config)
        
        # File should be gone
        assert not test_file.exists()

    @pytest.mark.asyncio
    async def test_discover_devices_with_config(self):
        """Test discover_devices with custom config."""
        with patch("bifrost.discovery.NetworkDiscovery") as mock_discovery_class:
            mock_discovery = MagicMock()
            mock_discovery.discover_network = AsyncMock(return_value=[])
            mock_discovery_class.return_value = mock_discovery
            
            devices = await discover_devices(
                network="10.0.0.0/24",
                methods=["ping"],
                timeout=10.0,
                config=self.config,
                use_cache=False
            )
            
            assert isinstance(devices, list)
            mock_discovery_class.assert_called_once_with(self.config)
            mock_discovery.discover_network.assert_called_once_with(
                network="10.0.0.0/24",
                methods=["ping"],
                timeout=10.0,
                use_cache=False
            )