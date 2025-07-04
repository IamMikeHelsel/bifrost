"""Device discovery and network configuration for industrial devices."""

import asyncio
import logging
import socket
import struct
import time
from dataclasses import dataclass
from ipaddress import IPv4Address, IPv4Network

from bifrost_core import DeviceInfo, ProtocolType

logger = logging.getLogger(__name__)


@dataclass
class BootPRequest:
    """BootP/DHCP request packet structure."""

    op: int = 1  # 1 = request, 2 = reply
    htype: int = 1  # Hardware type (1 = Ethernet)
    hlen: int = 6  # Hardware address length
    hops: int = 0
    xid: int = 0  # Transaction ID
    secs: int = 0
    flags: int = 0
    ciaddr: str = "0.0.0.0"  # Client IP
    yiaddr: str = "0.0.0.0"  # Your IP
    siaddr: str = "0.0.0.0"  # Server IP
    giaddr: str = "0.0.0.0"  # Gateway IP
    chaddr: bytes = b"\x00" * 16  # Client hardware address
    sname: bytes = b"\x00" * 64  # Server name
    file: bytes = b"\x00" * 128  # Boot file name
    options: bytes = b""  # DHCP options


@dataclass
class DiscoveredDevice:
    """Information about a discovered device."""

    mac_address: str
    ip_address: str | None
    hostname: str | None
    vendor: str | None
    device_type: str | None
    protocol: ProtocolType | None
    ports: list[int]
    last_seen: float
    additional_info: dict[str, str]


class NetworkDiscovery:
    """Network discovery using multiple methods."""

    def __init__(self):
        self.discovered_devices: dict[str, DiscoveredDevice] = {}

    async def discover_network(
        self,
        network: str = "192.168.1.0/24",
        methods: list[str] = None,
        timeout: float = 5.0,
    ) -> list[DeviceInfo]:
        """Discover devices on the network using multiple methods.

        Args:
            network: Network to scan in CIDR notation
            methods: Discovery methods to use (ping, arp, bootp, modbus)
            timeout: Timeout per method

        Returns:
            List of discovered devices
        """
        if methods is None:
            methods = ["ping", "arp", "bootp", "modbus"]

        logger.info(f"Starting network discovery on {network}")

        # Run discovery methods concurrently
        tasks = []

        if "ping" in methods:
            tasks.append(self._ping_sweep(network, timeout))
        if "arp" in methods:
            tasks.append(self._arp_discovery(network))
        if "bootp" in methods:
            tasks.append(self._bootp_discovery(timeout))
        if "modbus" in methods:
            tasks.append(self._modbus_discovery(network, timeout))

        # Execute all discovery methods
        await asyncio.gather(*tasks, return_exceptions=True)

        # Convert to DeviceInfo objects
        devices = []
        for device in self.discovered_devices.values():
            if device.ip_address:
                devices.append(
                    DeviceInfo(
                        device_id=device.mac_address,
                        protocol=device.protocol or ProtocolType.MODBUS_TCP,
                        host=device.ip_address,
                        port=502
                        if ProtocolType.MODBUS_TCP in (device.protocol or [])
                        else None,
                        name=device.hostname,
                        manufacturer=device.vendor,
                        additional_info=device.additional_info,
                    )
                )

        logger.info(f"Discovered {len(devices)} devices")
        return devices

    async def _ping_sweep(self, network: str, timeout: float) -> None:
        """Discover devices using ping sweep."""
        net = IPv4Network(network)
        logger.debug(f"Starting ping sweep on {network}")

        async def ping_host(ip: str) -> None:
            try:
                proc = await asyncio.create_subprocess_exec(
                    "ping",
                    "-c",
                    "1",
                    "-W",
                    str(int(timeout * 1000)),
                    str(ip),
                    stdout=asyncio.subprocess.DEVNULL,
                    stderr=asyncio.subprocess.DEVNULL,
                )
                returncode = await proc.wait()
                if returncode == 0:
                    # Host is alive, try to get more info
                    await self._probe_host(str(ip))
            except Exception as e:
                logger.debug(f"Ping failed for {ip}: {e}")

        # Ping all hosts in parallel (but limit concurrency)
        semaphore = asyncio.Semaphore(50)  # Limit concurrent pings

        async def ping_with_semaphore(ip: str) -> None:
            async with semaphore:
                await ping_host(ip)

        tasks = [ping_with_semaphore(str(ip)) for ip in net.hosts()]
        await asyncio.gather(*tasks, return_exceptions=True)

    async def _arp_discovery(self, network: str) -> None:
        """Discover devices using ARP table."""
        try:
            # Read system ARP table
            proc = await asyncio.create_subprocess_exec(
                "arp",
                "-a",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            stdout, _ = await proc.communicate()

            net = IPv4Network(network)
            for line in stdout.decode().split("\n"):
                if "(" in line and ")" in line:
                    # Parse ARP entry: hostname (ip) at mac [ether] on interface
                    parts = line.split()
                    if len(parts) >= 4:
                        ip_part = parts[1].strip("()")
                        mac_part = parts[3] if len(parts) > 3 else ""

                        try:
                            ip = IPv4Address(ip_part)
                            if ip in net and mac_part and ":" in mac_part:
                                await self._add_discovered_device(
                                    mac_part, str(ip), None, "arp"
                                )
                        except ValueError:
                            continue
        except Exception as e:
            logger.debug(f"ARP discovery failed: {e}")

    async def _bootp_discovery(self, timeout: float) -> None:
        """Discover devices using BootP/DHCP requests."""
        try:
            # Create UDP socket for BootP
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)
            sock.settimeout(timeout)

            # Create BootP request packet
            xid = int(time.time()) & 0xFFFFFFFF
            bootp_request = self._create_bootp_request(xid)

            # Send broadcast BootP request
            sock.sendto(bootp_request, ("255.255.255.255", 67))
            logger.debug("Sent BootP discovery request")

            # Listen for responses
            start_time = time.time()
            while time.time() - start_time < timeout:
                try:
                    data, addr = sock.recvfrom(1024)
                    if len(data) >= 236:  # Minimum BootP packet size
                        await self._parse_bootp_response(data, addr[0])
                except TimeoutError:
                    break
                except Exception as e:
                    logger.debug(f"BootP receive error: {e}")
                    break

            sock.close()

        except Exception as e:
            logger.debug(f"BootP discovery failed: {e}")

    async def _modbus_discovery(self, network: str, timeout: float) -> None:
        """Discover Modbus devices by scanning common ports."""
        net = IPv4Network(network)
        modbus_ports = [502, 503, 505]  # Common Modbus ports

        async def check_modbus(ip: str, port: int) -> None:
            try:
                # Try to connect to Modbus port
                reader, writer = await asyncio.wait_for(
                    asyncio.open_connection(ip, port), timeout=timeout
                )
                writer.close()
                await writer.wait_closed()

                # If connection successful, it's likely a Modbus device
                await self._add_discovered_device(
                    f"unknown_{ip}_{port}",
                    ip,
                    None,
                    "modbus",
                    protocol=ProtocolType.MODBUS_TCP,
                    ports=[port],
                )

            except Exception:
                pass  # Host not responding on this port

        # Check all IP/port combinations
        tasks = []
        for ip in net.hosts():
            for port in modbus_ports:
                tasks.append(check_modbus(str(ip), port))

        # Limit concurrency to avoid overwhelming the network
        semaphore = asyncio.Semaphore(20)

        async def check_with_semaphore(task):
            async with semaphore:
                await task

        await asyncio.gather(
            *[check_with_semaphore(task) for task in tasks], return_exceptions=True
        )

    async def _probe_host(self, ip: str) -> None:
        """Probe a host for additional information."""
        try:
            # Try reverse DNS lookup
            hostname = None
            try:
                hostname = socket.gethostbyaddr(ip)[0]
            except Exception:
                pass

            # Check for common industrial ports
            ports = []
            common_ports = [
                22,
                23,
                80,
                443,
                502,
                503,
                505,
                4840,
            ]  # SSH, Telnet, HTTP, HTTPS, Modbus, OPC UA

            for port in common_ports:
                try:
                    reader, writer = await asyncio.wait_for(
                        asyncio.open_connection(ip, port), timeout=1.0
                    )
                    ports.append(port)
                    writer.close()
                    await writer.wait_closed()
                except Exception:
                    pass

            # Determine likely protocol
            protocol = None
            if 502 in ports or 503 in ports or 505 in ports:
                protocol = ProtocolType.MODBUS_TCP
            elif 4840 in ports:
                protocol = ProtocolType.OPCUA

            await self._add_discovered_device(
                f"unknown_{ip}", ip, hostname, "probe", protocol=protocol, ports=ports
            )

        except Exception as e:
            logger.debug(f"Host probe failed for {ip}: {e}")

    async def _add_discovered_device(
        self,
        mac: str,
        ip: str,
        hostname: str | None = None,
        method: str = "unknown",
        protocol: ProtocolType | None = None,
        ports: list[int] = None,
    ) -> None:
        """Add or update a discovered device."""
        device_key = mac or f"ip_{ip}"

        if device_key in self.discovered_devices:
            # Update existing device
            device = self.discovered_devices[device_key]
            if ip and not device.ip_address:
                device.ip_address = ip
            if hostname and not device.hostname:
                device.hostname = hostname
            if protocol and not device.protocol:
                device.protocol = protocol
            if ports:
                device.ports.extend(p for p in ports if p not in device.ports)
            device.last_seen = time.time()
        else:
            # Add new device
            self.discovered_devices[device_key] = DiscoveredDevice(
                mac_address=mac,
                ip_address=ip,
                hostname=hostname,
                vendor=self._lookup_vendor(mac) if mac else None,
                device_type=self._guess_device_type(hostname, ports or []),
                protocol=protocol,
                ports=ports or [],
                last_seen=time.time(),
                additional_info={"discovery_method": method},
            )

    def _create_bootp_request(self, xid: int) -> bytes:
        """Create a BootP request packet."""
        # BootP header (without options)
        packet = struct.pack(
            "!BBBB4I16s64s128s",
            1,  # op (request)
            1,  # htype (Ethernet)
            6,  # hlen (MAC length)
            0,  # hops
            xid,  # xid
            0,  # secs
            0,  # flags
            0,  # ciaddr
            0,
            0,
            0,
            0,  # yiaddr, siaddr, giaddr (all 0)
            b"\x00" * 16,  # chaddr (client hardware address)
            b"\x00" * 64,  # sname (server name)
            b"\x00" * 128,  # file (boot file)
        )

        # Add DHCP magic cookie and basic options
        dhcp_options = b"\x63\x82\x53\x63"  # DHCP magic cookie
        dhcp_options += b"\x35\x01\x01"  # DHCP Message Type: Discover
        dhcp_options += b"\x37\x03\x01\x03\x06"  # Parameter Request List
        dhcp_options += b"\xff"  # End option

        return packet + dhcp_options

    async def _parse_bootp_response(self, data: bytes, sender_ip: str) -> None:
        """Parse BootP response packet."""
        try:
            if len(data) < 236:
                return

            # Parse BootP header
            header = struct.unpack("!BBBB4I16s64s128s", data[:236])
            op, htype, hlen, hops, xid, secs, flags = header[:7]
            ciaddr, yiaddr, siaddr, giaddr = header[7:11]
            chaddr = header[11]

            if op == 2:  # BootP reply
                # Extract MAC address
                mac_bytes = chaddr[:6]
                mac = ":".join(f"{b:02x}" for b in mac_bytes)

                # Extract IP addresses
                client_ip = socket.inet_ntoa(struct.pack("!I", ciaddr))
                your_ip = socket.inet_ntoa(struct.pack("!I", yiaddr))

                # Use the assigned IP or client IP
                device_ip = your_ip if your_ip != "0.0.0.0" else client_ip

                await self._add_discovered_device(mac, device_ip, None, "bootp")

        except Exception as e:
            logger.debug(f"Failed to parse BootP response: {e}")

    def _lookup_vendor(self, mac: str) -> str | None:
        """Lookup vendor from MAC address OUI."""
        # Simple OUI lookup for common industrial vendors
        oui_map = {
            "00:50:c2": "IEEE Registration Authority",
            "00:80:a3": "Lantronix",
            "00:a0:45": "Yokogawa",
            "00:e0:4c": "Realtek",
            "08:00:30": "Network Research Corporation",
            "00:00:bc": "Allen-Bradley",
            "00:80:f4": "Micronics Computer",
        }

        if len(mac) >= 8:
            oui = mac[:8].lower()
            return oui_map.get(oui)
        return None

    def _guess_device_type(self, hostname: str | None, ports: list[int]) -> str | None:
        """Guess device type based on hostname and open ports."""
        if hostname:
            hostname_lower = hostname.lower()
            if any(term in hostname_lower for term in ["plc", "controller"]):
                return "PLC"
            elif any(term in hostname_lower for term in ["hmi", "panel"]):
                return "HMI"
            elif any(term in hostname_lower for term in ["gateway", "bridge"]):
                return "Gateway"

        if 502 in ports or 503 in ports:
            return "Modbus Device"
        elif 4840 in ports:
            return "OPC UA Server"
        elif 80 in ports or 443 in ports:
            return "Web-enabled Device"

        return None


class IPAssignment:
    """IP address assignment for industrial devices."""

    @staticmethod
    async def assign_ip_via_bootp(
        mac_address: str,
        new_ip: str,
        subnet_mask: str = "255.255.255.0",
        gateway: str = None,
        timeout: float = 10.0,
    ) -> bool:
        """Assign IP address to device via BootP/DHCP.

        Args:
            mac_address: Target device MAC address
            new_ip: New IP address to assign
            subnet_mask: Subnet mask
            gateway: Gateway IP (optional)
            timeout: Timeout for operation

        Returns:
            True if assignment successful
        """
        try:
            # Create custom BootP response for IP assignment
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.setsockopt(socket.SOL_SOCKET, socket.SO_BROADCAST, 1)

            # Create BootP assignment packet
            xid = int(time.time()) & 0xFFFFFFFF
            packet = IPAssignment._create_ip_assignment_packet(
                xid, mac_address, new_ip, subnet_mask, gateway
            )

            # Send to device (broadcast)
            sock.sendto(packet, ("255.255.255.255", 67))

            # Wait for acknowledgment
            sock.settimeout(timeout)
            try:
                data, addr = sock.recvfrom(1024)
                # TODO: Parse response to confirm assignment
                logger.info(f"IP assignment response from {addr}")
                return True
            except TimeoutError:
                logger.warning("No response to IP assignment")
                return False
            finally:
                sock.close()

        except Exception as e:
            logger.error(f"IP assignment failed: {e}")
            return False

    @staticmethod
    def _create_ip_assignment_packet(
        xid: int,
        mac_address: str,
        ip_address: str,
        subnet_mask: str,
        gateway: str | None,
    ) -> bytes:
        """Create BootP packet for IP assignment."""
        # Convert MAC address to bytes
        mac_bytes = bytes.fromhex(mac_address.replace(":", ""))
        chaddr = mac_bytes + b"\x00" * (16 - len(mac_bytes))

        # Convert IP addresses
        yiaddr = struct.unpack("!I", socket.inet_aton(ip_address))[0]
        siaddr = struct.unpack("!I", socket.inet_aton("0.0.0.0"))[0]

        # Create BootP response packet
        packet = struct.pack(
            "!BBBB4I16s64s128s",
            2,  # op (reply)
            1,  # htype (Ethernet)
            6,  # hlen
            0,  # hops
            xid,  # xid
            0,  # secs
            0,  # flags
            0,  # ciaddr
            yiaddr,  # yiaddr (your IP)
            siaddr,  # siaddr (server IP)
            0,  # giaddr
            chaddr,  # chaddr
            b"\x00" * 64,  # sname
            b"\x00" * 128,  # file
        )

        # Add DHCP options
        options = b"\x63\x82\x53\x63"  # Magic cookie
        options += b"\x35\x01\x02"  # DHCP Message Type: Offer

        # Subnet mask
        mask_bytes = socket.inet_aton(subnet_mask)
        options += b"\x01\x04" + mask_bytes

        # Gateway (if provided)
        if gateway:
            gw_bytes = socket.inet_aton(gateway)
            options += b"\x03\x04" + gw_bytes

        options += b"\xff"  # End option

        return packet + options


# Global discovery instance
_discovery = NetworkDiscovery()


async def discover_devices(
    network: str = "192.168.1.0/24", methods: list[str] = None, timeout: float = 5.0
) -> list[DeviceInfo]:
    """Discover devices on the network.

    This is the main entry point for device discovery.
    """
    return await _discovery.discover_network(network, methods, timeout)


async def assign_device_ip(
    mac_address: str,
    ip_address: str,
    subnet_mask: str = "255.255.255.0",
    gateway: str = "",
) -> bool:
    """Assign IP address to a device.

    This is the main entry point for IP assignment.
    """
    return await IPAssignment.assign_ip_via_bootp(
        mac_address, ip_address, subnet_mask, gateway
    )
