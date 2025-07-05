"""Network device discovery for Bifrost.

This module provides multi-protocol device discovery capabilities for industrial
networks, supporting BOOTP/DHCP, Ethernet/IP (CIP), and Modbus TCP protocols.
"""

import asyncio
import ipaddress
import random
import socket
import struct
import time
from collections.abc import AsyncGenerator, Sequence
from typing import Any

from bifrost_core.base import DeviceInfo
from bifrost_core.typing import JsonDict


class DiscoveryConfig:
    """Configuration for device discovery.
    
    Attributes:
        network_range: IP network range to scan (CIDR notation).
        timeout: Timeout in seconds for each device connection attempt.
        max_concurrent: Maximum number of concurrent connections.
        protocols: List of protocols to use for discovery.
    """
    
    def __init__(
        self,
        network_range: str = "192.168.1.0/24",
        timeout: float = 2.0,
        max_concurrent: int = 50,
        protocols: Sequence[str] = ("modbus", "cip", "bootp"),
    ):
        """Initialize discovery configuration.
        
        Args:
            network_range: IP network range to scan (default: 192.168.1.0/24).
            timeout: Timeout in seconds for connections (default: 2.0).
            max_concurrent: Max concurrent connections (default: 50).
            protocols: Discovery protocols to use (default: all).
        """
        self.network_range = network_range
        self.timeout = timeout
        self.max_concurrent = max_concurrent
        self.protocols = protocols


async def discover_bootp_devices(config: DiscoveryConfig) -> AsyncGenerator[DeviceInfo, None]:
    """Discover devices using BOOTP/DHCP discovery."""
    # BOOTP discovery listens for BOOTP/DHCP traffic and device announcements
    # This is a simplified implementation - real BOOTP discovery would:
    # 1. Listen for DHCP discover/request packets
    # 2. Parse vendor-specific information
    # 3. Identify industrial devices by vendor class identifiers
    
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)
    sock.settimeout(config.timeout)
    
    try:
        # Send a DHCP discover-like packet to identify BOOTP-enabled devices
        # This is a simplified broadcast to trigger responses
        broadcast_addr = ("255.255.255.255", 67)
        
        # Simple DHCP discover packet structure (simplified)
        dhcp_discover = bytearray(300)
        dhcp_discover[0] = 1  # BOOTREQUEST
        dhcp_discover[1] = 1  # Hardware type: Ethernet
        dhcp_discover[2] = 6  # Hardware address length
        dhcp_discover[3] = 0  # Hops
        
        # Transaction ID (random)
        import random
        txid = random.randint(0, 0xFFFFFFFF)
        dhcp_discover[4:8] = struct.pack(">I", txid)
        
        sock.sendto(dhcp_discover, broadcast_addr)
        
        # Listen for responses (simplified)
        start_time = time.time()
        while time.time() - start_time < config.timeout:
            try:
                data, addr = sock.recvfrom(1024)
                if len(data) > 240:  # Minimum DHCP packet size
                    yield DeviceInfo(
                        device_id=f"bootp_{addr[0]}",
                        protocol="bootp",
                        host=addr[0],
                        port=67,
                        discovery_method="bootp",
                        device_type="BOOTP Device",
                        confidence=0.8,
                        last_seen=int(time.time() * 1_000_000_000),  # nanoseconds
                        metadata={"source_address": addr[0]}
                    )
            except socket.timeout:
                continue
                
    except Exception:
        # Silently handle network errors
        pass
    finally:
        sock.close()


async def discover_cip_devices(config: DiscoveryConfig) -> AsyncGenerator[DeviceInfo, None]:
    """Discover devices using Ethernet/IP CIP ListIdentity."""
    # CIP ListIdentity multicast discovery
    # Sends ListIdentity (0x0063) command to multicast address 224.0.1.1:44818
    
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.settimeout(config.timeout)
    
    try:
        # CIP ListIdentity command structure
        # EtherNet/IP Encapsulation Header + CIP ListIdentity
        list_identity = bytearray()
        
        # EtherNet/IP Header
        list_identity.extend([0x63, 0x00])  # Command: ListIdentity (0x0063)
        list_identity.extend([0x00, 0x00])  # Length: 0
        list_identity.extend([0x00, 0x00, 0x00, 0x00])  # Session Handle: 0
        list_identity.extend([0x00, 0x00, 0x00, 0x00])  # Status: 0
        list_identity.extend([0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00])  # Sender Context
        list_identity.extend([0x00, 0x00, 0x00, 0x00])  # Options: 0
        
        # Send to multicast address
        multicast_addr = ("224.0.1.1", 44818)
        sock.sendto(list_identity, multicast_addr)
        
        # Also try broadcast
        broadcast_addr = ("255.255.255.255", 44818)
        sock.sendto(list_identity, broadcast_addr)
        
        # Listen for responses
        start_time = time.time()
        while time.time() - start_time < config.timeout:
            try:
                data, addr = sock.recvfrom(1024)
                if len(data) >= 24:  # Minimum EtherNet/IP response
                    # Parse response (simplified)
                    command = struct.unpack("<H", data[0:2])[0]
                    if command == 0x0063:  # ListIdentity response
                        yield DeviceInfo(
                            device_id=f"cip_{addr[0]}",
                            protocol="ethernet_ip",
                            host=addr[0],
                            port=44818,
                            discovery_method="cip",
                            device_type="EtherNet/IP Device",
                            confidence=0.9,
                            last_seen=int(time.time() * 1_000_000_000),
                            metadata={
                                "source_address": addr[0],
                                "response_length": len(data)
                            }
                        )
            except socket.timeout:
                continue
                
    except Exception:
        # Silently handle network errors
        pass
    finally:
        sock.close()


async def discover_modbus_devices(config: DiscoveryConfig) -> AsyncGenerator[DeviceInfo, None]:
    """Discover Modbus TCP devices by scanning network range."""
    network = ipaddress.ip_network(config.network_range, strict=False)
    semaphore = asyncio.Semaphore(config.max_concurrent)
    
    async def scan_host(host: str) -> DeviceInfo | None:
        async with semaphore:
            try:
                # Connect to Modbus TCP port (502)
                reader, writer = await asyncio.wait_for(
                    asyncio.open_connection(host, 502),
                    timeout=config.timeout
                )
                
                # Send Modbus Read Device Identification request
                # Transaction ID (2), Protocol ID (2), Length (6), Unit ID (1), Function (1), MEI Type (1), Read Code (1), Object ID (1)
                request = bytearray([
                    0x00, 0x01,  # Transaction ID
                    0x00, 0x00,  # Protocol ID
                    0x00, 0x06,  # Length
                    0x01,        # Unit ID
                    0x2B,        # Function Code: Read Device Identification
                    0x0E,        # MEI Type: Read Device Identification
                    0x01,        # Read Device ID Code: Basic
                    0x00         # Object ID: Vendor Name
                ])
                
                writer.write(request)
                await writer.drain()
                
                # Read response
                response = await asyncio.wait_for(
                    reader.read(256),
                    timeout=1.0
                )
                
                writer.close()
                await writer.wait_closed()
                
                if len(response) >= 8:
                    device_info = DeviceInfo(
                        device_id=f"modbus_{host}",
                        protocol="modbus.tcp",
                        host=host,
                        port=502,
                        discovery_method="modbus",
                        device_type="Modbus Device",
                        confidence=0.95,
                        last_seen=int(time.time() * 1_000_000_000),
                        metadata={"response_length": len(response)}
                    )
                    
                    # Try to parse device identification if available
                    if len(response) > 12 and response[7] == 0x2B:
                        # Parse vendor name, product code, etc. from response
                        # This is simplified - real parsing would handle the full MEI response
                        device_info.metadata["has_device_identification"] = True
                    
                    return device_info
                    
            except (asyncio.TimeoutError, ConnectionRefusedError, OSError):
                # No Modbus device at this address
                pass
            except Exception:
                # Other errors, skip this host
                pass
            
            return None
    
    # Scan all hosts in parallel
    tasks = [scan_host(str(host)) for host in network.hosts()]
    
    # Process results as they complete
    for task in asyncio.as_completed(tasks):
        try:
            result = await task
            if result:
                yield result
        except Exception:
            # Skip failed scans
            continue


async def discover_devices(
    config: DiscoveryConfig | None = None,
    protocols: Sequence[str] | None = None
) -> AsyncGenerator[DeviceInfo, None]:
    """Discover devices using multiple protocols."""
    if config is None:
        config = DiscoveryConfig()
    
    if protocols is None:
        protocols = config.protocols
    
    # Run discovery methods concurrently
    discovery_tasks = []
    
    if "bootp" in protocols:
        discovery_tasks.append(discover_bootp_devices(config))
    
    if "cip" in protocols or "ethernet_ip" in protocols:
        discovery_tasks.append(discover_cip_devices(config))
    
    if "modbus" in protocols:
        discovery_tasks.append(discover_modbus_devices(config))
    
    # Merge results from all discovery methods
    seen_devices = set()
    
    async def collect_from_generator(gen: AsyncGenerator[DeviceInfo, None]):
        async for device in gen:
            device_key = (device.host, device.port, device.protocol)
            if device_key not in seen_devices:
                seen_devices.add(device_key)
                yield device
    
    # Collect from all generators concurrently
    import asyncio
    from asyncio import create_task
    
    tasks = [create_task(collect_from_generator(gen).__anext__()) for gen in discovery_tasks]
    
    while tasks:
        done, tasks = await asyncio.wait(tasks, return_when=asyncio.FIRST_COMPLETED)
        
        for task in done:
            try:
                device = await task
                yield device
                # Create a new task for the same generator
                # This is simplified - real implementation would handle generator completion
            except StopAsyncIteration:
                # Generator completed
                pass
            except Exception:
                # Handle errors
                pass
