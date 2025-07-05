"""Network device discovery for Bifrost."""

import asyncio
import ipaddress
import socket
import struct
import time
from typing import List, Optional, Dict, Any
import logging

from bifrost_core.base import JsonDict
from bifrost_core import ProtocolType


logger = logging.getLogger(__name__)


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
    
    def __init__(self):
        self.discovered_devices: Dict[str, DiscoveredDevice] = {}
    
    async def discover_network(
        self, 
        network: str = "192.168.1.0/24",
        methods: List[str] = None,
        timeout: float = 5.0
    ) -> List[DiscoveredDevice]:
        """Discover devices on the network using various methods."""
        if methods is None:
            methods = ["ping", "arp", "bootp", "modbus"]
        
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
        
        return list(self.discovered_devices.values())
    
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
    timeout: float = 5.0
) -> List[DiscoveredDevice]:
    """Discover devices on the network."""
    discovery = NetworkDiscovery()
    return await discovery.discover_network(
        network=network, 
        methods=methods, 
        timeout=timeout
    )


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
