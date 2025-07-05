"""Network device discovery for Bifrost."""

import asyncio
import hashlib
import ipaddress
import json
import logging
import os
import socket
import struct
import time
from pathlib import Path
from typing import List, Optional, Dict, Any
from dataclasses import dataclass, asdict

from bifrost_core.base import JsonDict
from bifrost_core import ProtocolType


logger = logging.getLogger(__name__)


@dataclass
class DiscoveryConfig:
    """Configuration for discovery operations."""
    cache_enabled: bool = True
    cache_ttl_seconds: int = 300  # 5 minutes default
    cache_dir: Optional[str] = None  # None means use system temp dir
    max_cache_size_mb: int = 10
    
    def __post_init__(self):
        if self.cache_dir is None:
            self.cache_dir = os.path.join(os.path.expanduser("~"), ".bifrost", "cache")


class DiscoveryCache:
    """Handles caching of discovery results for performance."""
    
    def __init__(self, config: DiscoveryConfig):
        self.config = config
        self.cache_dir = Path(config.cache_dir)
        self.cache_dir.mkdir(parents=True, exist_ok=True)
        self._memory_cache: Dict[str, Dict[str, Any]] = {}
    
    def _generate_cache_key(
        self, 
        network: str, 
        methods: List[str], 
        timeout: float
    ) -> str:
        """Generate a unique cache key for discovery parameters."""
        # Create a normalized string of parameters
        methods_str = ",".join(sorted(methods))
        params = f"{network}|{methods_str}|{timeout}"
        
        # Generate SHA256 hash for consistent key
        return hashlib.sha256(params.encode()).hexdigest()[:16]
    
    def _get_cache_file_path(self, cache_key: str) -> Path:
        """Get the file path for a cache key."""
        return self.cache_dir / f"discovery_{cache_key}.json"
    
    def _is_cache_valid(self, cache_data: Dict[str, Any]) -> bool:
        """Check if cached data is still valid."""
        if not self.config.cache_enabled:
            return False
        
        timestamp = cache_data.get("timestamp", 0)
        age = time.time() - timestamp
        return age < self.config.cache_ttl_seconds
    
    async def get_cached_results(
        self, 
        network: str, 
        methods: List[str], 
        timeout: float
    ) -> Optional[List["DiscoveredDevice"]]:
        """Get cached discovery results if available and valid."""
        if not self.config.cache_enabled:
            return None
        
        cache_key = self._generate_cache_key(network, methods, timeout)
        
        # Check memory cache first
        if cache_key in self._memory_cache:
            cache_data = self._memory_cache[cache_key]
            if self._is_cache_valid(cache_data):
                logger.debug(f"Using memory cache for discovery: {cache_key}")
                return self._deserialize_devices(cache_data["devices"])
            else:
                # Remove expired entry
                del self._memory_cache[cache_key]
        
        # Check file cache
        cache_file = self._get_cache_file_path(cache_key)
        if cache_file.exists():
            try:
                with open(cache_file, 'r') as f:
                    cache_data = json.load(f)
                
                if self._is_cache_valid(cache_data):
                    logger.debug(f"Using file cache for discovery: {cache_key}")
                    # Load into memory cache for faster access
                    self._memory_cache[cache_key] = cache_data
                    return self._deserialize_devices(cache_data["devices"])
                else:
                    # Remove expired file
                    cache_file.unlink(missing_ok=True)
                    
            except (json.JSONDecodeError, IOError) as e:
                logger.warning(f"Failed to read cache file {cache_file}: {e}")
                cache_file.unlink(missing_ok=True)
        
        return None
    
    async def cache_results(
        self, 
        network: str, 
        methods: List[str], 
        timeout: float,
        devices: List["DiscoveredDevice"]
    ) -> None:
        """Cache discovery results for future use."""
        if not self.config.cache_enabled:
            return
        
        cache_key = self._generate_cache_key(network, methods, timeout)
        cache_data = {
            "timestamp": time.time(),
            "network": network,
            "methods": methods,
            "timeout": timeout,
            "devices": self._serialize_devices(devices)
        }
        
        # Store in memory cache
        self._memory_cache[cache_key] = cache_data
        
        # Store in file cache
        cache_file = self._get_cache_file_path(cache_key)
        try:
            with open(cache_file, 'w') as f:
                json.dump(cache_data, f, indent=2)
            logger.debug(f"Cached discovery results: {cache_key}")
        except IOError as e:
            logger.warning(f"Failed to write cache file {cache_file}: {e}")
        
        # Clean up old cache files if needed
        await self._cleanup_cache()
    
    def _serialize_devices(self, devices: List["DiscoveredDevice"]) -> List[Dict[str, Any]]:
        """Serialize device objects for caching."""
        serialized = []
        for device in devices:
            device_dict = {
                "mac_address": device.mac_address,
                "ip_address": device.ip_address,
                "hostname": device.hostname,
                "vendor": device.vendor,
                "device_type": device.device_type,
                "protocol": device.protocol.value if device.protocol else None,
                "ports": device.ports,
                "last_seen": device.last_seen,
                "additional_info": device.additional_info
            }
            serialized.append(device_dict)
        return serialized
    
    def _deserialize_devices(self, device_data: List[Dict[str, Any]]) -> List["DiscoveredDevice"]:
        """Deserialize device objects from cache."""
        devices = []
        for data in device_data:
            protocol = None
            if data.get("protocol"):
                try:
                    protocol = ProtocolType(data["protocol"])
                except ValueError:
                    protocol = None
            
            device = DiscoveredDevice(
                mac_address=data["mac_address"],
                ip_address=data.get("ip_address"),
                hostname=data.get("hostname"),
                vendor=data.get("vendor"),
                device_type=data.get("device_type"),
                protocol=protocol,
                ports=data.get("ports", []),
                last_seen=data.get("last_seen", 0.0),
                additional_info=data.get("additional_info", {})
            )
            devices.append(device)
        return devices
    
    async def _cleanup_cache(self):
        """Clean up old cache files to stay within size limits."""
        try:
            cache_files = list(self.cache_dir.glob("discovery_*.json"))
            total_size = sum(f.stat().st_size for f in cache_files if f.exists())
            max_size_bytes = self.config.max_cache_size_mb * 1024 * 1024
            
            if total_size > max_size_bytes:
                # Sort by last modified time (oldest first)
                cache_files.sort(key=lambda f: f.stat().st_mtime)
                
                for cache_file in cache_files:
                    if total_size <= max_size_bytes:
                        break
                    
                    file_size = cache_file.stat().st_size
                    cache_file.unlink(missing_ok=True)
                    total_size -= file_size
                    logger.debug(f"Removed old cache file: {cache_file}")
                    
        except Exception as e:
            logger.warning(f"Error during cache cleanup: {e}")
    
    def clear_cache(self):
        """Clear all cached discovery results."""
        # Clear memory cache
        self._memory_cache.clear()
        
        # Clear file cache
        try:
            for cache_file in self.cache_dir.glob("discovery_*.json"):
                cache_file.unlink(missing_ok=True)
            logger.info("Discovery cache cleared")
        except Exception as e:
            logger.warning(f"Error clearing cache: {e}")


class BootPRequest:
    """BootP/DHCP request packet for device discovery."""
    
    def __init__(self, xid: int = 0, ciaddr: str = "0.0.0.0", chaddr: bytes = None):
        self.op = 1  # Boot request
        self.htype = 1  # Ethernet
        self.hlen = 6  # MAC address length
        self.hops = 0
        self.xid = xid
        self.secs = 0
        self.flags = 0
        self.ciaddr = ciaddr
        self.yiaddr = "0.0.0.0"
        self.siaddr = "0.0.0.0"
        self.giaddr = "0.0.0.0"
        self.chaddr = chaddr or (b"\x00" * 16)
        self.sname = b"\x00" * 64
        self.file = b"\x00" * 128


class DiscoveredDevice:
    """Represents a discovered network device."""
    
    def __init__(
        self,
        mac_address: str,
        ip_address: Optional[str] = None,
        hostname: Optional[str] = None,
        vendor: Optional[str] = None,
        device_type: Optional[str] = None,
        protocol: Optional[ProtocolType] = None,
        ports: List[int] = None,
        last_seen: float = 0.0,
        additional_info: Dict[str, Any] = None
    ):
        self.mac_address = mac_address
        self.ip_address = ip_address
        self.hostname = hostname
        self.vendor = vendor
        self.device_type = device_type
        self.protocol = protocol
        self.ports = ports or []
        self.last_seen = last_seen
        self.additional_info = additional_info or {}


class NetworkDiscovery:
    """Network discovery functionality for finding devices."""
    
    def __init__(self, config: DiscoveryConfig = None):
        self.config = config or DiscoveryConfig()
        self.cache = DiscoveryCache(self.config)
        self.discovered_devices: Dict[str, DiscoveredDevice] = {}
    
    async def discover_network(
        self, 
        network: str = "192.168.1.0/24",
        methods: List[str] = None,
        timeout: float = 5.0,
        use_cache: bool = True
    ) -> List[DiscoveredDevice]:
        """Discover devices on the network using various methods."""
        if methods is None:
            methods = ["ping", "arp", "bootp", "modbus"]
        
        # Try to get cached results first
        if use_cache:
            cached_devices = await self.cache.get_cached_results(network, methods, timeout)
            if cached_devices is not None:
                logger.info(f"Using cached discovery results for {network}")
                return cached_devices
        
        logger.info(f"Starting network discovery for {network} using methods: {methods}")
        
        # Clear previous results
        self.discovered_devices.clear()
        
        devices = []
        
        for method in methods:
            try:
                if method == "ping":
                    await self._ping_sweep(network, timeout)
                elif method == "arp":
                    await self._arp_discovery()
                elif method == "bootp":
                    await self._bootp_discovery(timeout)
                elif method == "modbus":
                    await self._modbus_discovery(network, timeout)
                else:
                    logger.warning(f"Unknown discovery method: {method}")
            except Exception as e:
                logger.error(f"Error in {method} discovery: {e}")
        
        devices = list(self.discovered_devices.values())
        
        # Cache the results
        if use_cache:
            await self.cache.cache_results(network, methods, timeout, devices)
        
        logger.info(f"Discovery completed. Found {len(devices)} devices.")
        return devices
    
    async def _ping_sweep(self, network: str, timeout: float):
        """Perform ping sweep on network range."""
        try:
            net = ipaddress.ip_network(network, strict=False)
            tasks = []
            
            for ip in net.hosts():
                if len(tasks) < 50:  # Limit concurrent pings
                    task = self._ping_host(str(ip), timeout)
                    tasks.append(task)
                else:
                    await asyncio.gather(*tasks, return_exceptions=True)
                    tasks = []
            
            if tasks:
                await asyncio.gather(*tasks, return_exceptions=True)
                
        except Exception as e:
            logger.error(f"Error in ping sweep: {e}")
    
    async def _ping_host(self, ip: str, timeout: float):
        """Ping a single host."""
        try:
            process = await asyncio.create_subprocess_exec(
                "ping", "-c", "1", "-W", str(int(timeout)), ip,
                stdout=asyncio.subprocess.DEVNULL,
                stderr=asyncio.subprocess.DEVNULL
            )
            await process.communicate()
            
            if process.returncode == 0:
                # Host is alive, add to discovered devices
                device = DiscoveredDevice(
                    mac_address="00:00:00:00:00:00",  # Placeholder
                    ip_address=ip,
                    last_seen=time.time()
                )
                self.discovered_devices[ip] = device
                
        except Exception as e:
            logger.debug(f"Ping failed for {ip}: {e}")
    
    async def _arp_discovery(self):
        """Discover devices using ARP table."""
        try:
            # This is a simplified implementation
            # In reality, you'd parse /proc/net/arp or use system commands
            pass
        except Exception as e:
            logger.error(f"Error in ARP discovery: {e}")
    
    async def _bootp_discovery(self, timeout: float):
        """Discover devices using BootP/DHCP."""
        try:
            with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as sock:
                sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)
                sock.settimeout(timeout)
                
                # Create BootP request
                request = BootPRequest()
                
                # Send broadcast request
                try:
                    sock.sendto(b"bootp_request", ("255.255.255.255", 67))
                    
                    # Listen for responses
                    while True:
                        try:
                            data, addr = sock.recvfrom(1024)
                            # Process response (simplified)
                            device = DiscoveredDevice(
                                mac_address="00:00:00:00:00:00",  # Would parse from response
                                ip_address=addr[0],
                                last_seen=time.time()
                            )
                            self.discovered_devices[addr[0]] = device
                        except socket.timeout:
                            break
                except Exception as e:
                    logger.debug(f"BootP discovery error: {e}")
                    
        except Exception as e:
            logger.error(f"Error in BootP discovery: {e}")
    
    async def _modbus_discovery(self, network: str, timeout: float):
        """Discover Modbus devices on the network."""
        try:
            net = ipaddress.ip_network(network, strict=False)
            tasks = []
            
            for ip in net.hosts():
                if len(tasks) < 20:  # Limit concurrent connections
                    task = self._check_modbus_device(str(ip), 502, timeout)
                    tasks.append(task)
                else:
                    await asyncio.gather(*tasks, return_exceptions=True)
                    tasks = []
            
            if tasks:
                await asyncio.gather(*tasks, return_exceptions=True)
                
        except Exception as e:
            logger.error(f"Error in Modbus discovery: {e}")
    
    async def _check_modbus_device(self, ip: str, port: int, timeout: float):
        """Check if a device supports Modbus at given IP and port."""
        try:
            reader, writer = await asyncio.wait_for(
                asyncio.open_connection(ip, port), 
                timeout=timeout
            )
            
            # Device responded, likely a Modbus device
            device = DiscoveredDevice(
                mac_address="00:00:00:00:00:00",  # Placeholder
                ip_address=ip,
                protocol=ProtocolType.MODBUS_TCP,
                ports=[port],
                device_type="PLC",
                last_seen=time.time()
            )
            self.discovered_devices[ip] = device
            
            writer.close()
            await writer.wait_closed()
            
        except Exception as e:
            logger.debug(f"Modbus check failed for {ip}:{port}: {e}")


async def discover_devices(
    network: str = "192.168.1.0/24",
    methods: List[str] = None,
    timeout: float = 5.0,
    config: DiscoveryConfig = None,
    use_cache: bool = True
) -> List[DiscoveredDevice]:
    """Discover devices on the network."""
    discovery = NetworkDiscovery(config)
    return await discovery.discover_network(
        network=network, 
        methods=methods, 
        timeout=timeout,
        use_cache=use_cache
    )


def clear_discovery_cache(config: DiscoveryConfig = None) -> None:
    """Clear all cached discovery results."""
    cache_config = config or DiscoveryConfig()
    cache = DiscoveryCache(cache_config)
    cache.clear_cache()


def get_cache_info(config: DiscoveryConfig = None) -> Dict[str, Any]:
    """Get information about the discovery cache."""
    cache_config = config or DiscoveryConfig()
    cache_dir = Path(cache_config.cache_dir)
    
    if not cache_dir.exists():
        return {
            "cache_enabled": cache_config.cache_enabled,
            "cache_dir": str(cache_dir),
            "cache_exists": False,
            "file_count": 0,
            "total_size_mb": 0.0
        }
    
    cache_files = list(cache_dir.glob("discovery_*.json"))
    total_size = sum(f.stat().st_size for f in cache_files if f.exists())
    
    return {
        "cache_enabled": cache_config.cache_enabled,
        "cache_dir": str(cache_dir),
        "cache_exists": True,
        "file_count": len(cache_files),
        "total_size_mb": total_size / (1024 * 1024),
        "ttl_seconds": cache_config.cache_ttl_seconds,
        "max_size_mb": cache_config.max_cache_size_mb
    }


async def assign_device_ip(
    mac_address: str,
    ip_address: str,
    netmask: str = "255.255.255.0",
    gateway: str = None
) -> bool:
    """Assign IP address to a device via BOOTP/DHCP."""
    try:
        # Validate MAC address format
        if len(mac_address.split(":")) != 6:
            return False
        
        # Validate IP address format
        try:
            ipaddress.ip_address(ip_address)
        except ValueError:
            return False
        
        with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as sock:
            sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)
            
            # Create assignment packet (simplified)
            packet = f"assign:{mac_address}:{ip_address}:{netmask}"
            if gateway:
                packet += f":{gateway}"
            
            sock.sendto(packet.encode(), ("255.255.255.255", 67))
            return True
            
    except Exception as e:
        logger.error(f"Error assigning IP to {mac_address}: {e}")
        return False
