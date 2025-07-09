#!/usr/bin/env python3
"""
Ethernet/IP (EtherNet/IP) Simulator for Bifrost Virtual Device Testing

This simulator provides a simplified Ethernet/IP device implementation for testing
device discovery and communication protocols. It simulates basic EtherNet/IP responses
without requiring the full cpppo library.

Features:
- EtherNet/IP UDP and TCP simulation
- CIP (Common Industrial Protocol) basic responses
- Device discovery support
- Realistic industrial device behavior
"""

import argparse
import asyncio
import logging
import signal
import sys
import time
import socket
import struct
from typing import Dict, Any, Optional

class BifrostEthernetIPServer:
    """Bifrost Ethernet/IP device simulator (simplified implementation)."""

    def __init__(
        self,
        host: str = "0.0.0.0",
        udp_port: int = 44818,
        tcp_port: int = 2222,
        device_name: str = "Bifrost_EIP_Device",
        log_level: str = "INFO"
    ):
        self.host = host
        self.udp_port = udp_port
        self.tcp_port = tcp_port
        self.device_name = device_name
        self.running = False
        self.udp_server = None
        self.tcp_server = None
        self.stats = {
            "udp_requests": 0,
            "tcp_connections": 0,
            "requests_total": 0,
            "requests_successful": 0,
            "requests_failed": 0,
            "start_time": time.time()
        }

        # Configure logging
        logging.basicConfig(
            level=getattr(logging, log_level.upper()),
            format="%(asctime)s - %(levelname)s - %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S"
        )
        self.logger = logging.getLogger(__name__)

        # Device identity information
        self.device_identity = {
            "vendor_id": 0x01F7,  # Custom vendor ID
            "device_type": 0x00,  # Generic Device
            "product_code": 0x0001,
            "revision": {"major": 1, "minor": 0},
            "status": 0x0000,
            "serial_number": 0x12345678,
            "product_name": device_name.encode('ascii')[:32]
        }

        # Setup signal handlers
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

    def _signal_handler(self, signum, frame):
        """Handle shutdown signals gracefully."""
        self.logger.info(f"Received signal {signum}, shutting down...")
        self.running = False

    def _create_list_identity_response(self) -> bytes:
        """Create EtherNet/IP List Identity response."""
        try:
            # EtherNet/IP Encapsulation Header (24 bytes)
            encap_header = struct.pack('<HHIIIHI',
                0x0065,                     # Command: List Identity Response
                24 + 32,                   # Length (header + data)
                0x12345678,                # Session handle
                0x00000000,                # Status
                0x00000000,                # Sender context (8 bytes)
                0x00000000,
                0x0000                     # Options
            )
            
            # List Identity Response Data (32 bytes minimum)
            product_name = self.device_identity["product_name"][:16].ljust(16, b'\x00')
            
            identity_data = struct.pack('<HHBBBHIBBBB',
                0x000C,                    # Item type code: CIP Identity
                28,                        # Item length
                1,                         # Encapsulation version
                0x00,                      # Socket address (sin_family)
                0x00,                      # Socket address (sin_port)
                0x00000000,                # Socket address (sin_addr)
                self.device_identity["vendor_id"],     # Vendor ID
                self.device_identity["device_type"],   # Device type
                self.device_identity["product_code"],  # Product code
                self.device_identity["revision"]["major"],  # Major revision
                self.device_identity["revision"]["minor"],  # Minor revision
                self.device_identity["status"],        # Status
                self.device_identity["serial_number"] & 0xFFFF,  # Serial (low)
                (self.device_identity["serial_number"] >> 16) & 0xFFFF  # Serial (high)
            ) + product_name[:16]
            
            return encap_header + identity_data
            
        except Exception as e:
            self.logger.error(f"Error creating List Identity response: {e}")
            return b''

    async def handle_udp_discovery(self):
        """Handle UDP discovery requests."""
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        
        try:
            sock.bind((self.host, self.udp_port))
            sock.setblocking(False)
            
            self.logger.info(f"EtherNet/IP UDP discovery server started on {self.host}:{self.udp_port}")
            
            while self.running:
                try:
                    data, addr = await asyncio.get_event_loop().sock_recvfrom(sock, 1024)
                    self.stats["udp_requests"] += 1
                    self.stats["requests_total"] += 1
                    
                    self.logger.debug(f"UDP discovery request from {addr}")
                    
                    # Check if it's a List Services or List Identity request
                    if len(data) >= 24:  # Minimum EtherNet/IP header size
                        command = struct.unpack('<H', data[0:2])[0]
                        
                        if command == 0x0004:  # List Services
                            # Simple List Services response
                            response = struct.pack('<HHIIIHI', 0x0104, 24, 0, 0, 0, 0, 0)
                            await asyncio.get_event_loop().sock_sendto(sock, response, addr)
                            self.stats["requests_successful"] += 1
                            
                        elif command == 0x0063:  # List Identity
                            response = self._create_list_identity_response()
                            if response:
                                await asyncio.get_event_loop().sock_sendto(sock, response, addr)
                                self.stats["requests_successful"] += 1
                            else:
                                self.stats["requests_failed"] += 1
                        else:
                            self.logger.debug(f"Unknown UDP command: 0x{command:04x}")
                            self.stats["requests_failed"] += 1
                    
                except asyncio.CancelledError:
                    break
                except Exception as e:
                    self.logger.error(f"UDP discovery error: {e}")
                    self.stats["requests_failed"] += 1
                    await asyncio.sleep(1)
                    
        finally:
            sock.close()

    async def handle_tcp_client(self, reader, writer):
        """Handle TCP client connection."""
        client_addr = writer.get_extra_info('peername')
        self.logger.info(f"EtherNet/IP TCP client connected from {client_addr}")
        self.stats["tcp_connections"] += 1
        
        try:
            while self.running:
                data = await asyncio.wait_for(reader.read(1024), timeout=30.0)
                if not data:
                    break
                
                self.stats["requests_total"] += 1
                self.logger.debug(f"TCP request from {client_addr}: {len(data)} bytes")
                
                # Simple TCP response (EtherNet/IP explicit messaging)
                if len(data) >= 24:  # EtherNet/IP header
                    # Echo back with a simple success response
                    response = struct.pack('<HHIIIHI', 0x006F, 24, 0, 0, 0, 0, 0)
                    writer.write(response)
                    await writer.drain()
                    self.stats["requests_successful"] += 1
                else:
                    self.stats["requests_failed"] += 1
                    
        except asyncio.TimeoutError:
            self.logger.debug(f"TCP client {client_addr} timeout")
        except Exception as e:
            self.logger.error(f"TCP client error: {e}")
            self.stats["requests_failed"] += 1
        finally:
            self.logger.info(f"EtherNet/IP TCP client {client_addr} disconnected")
            writer.close()
            await writer.wait_closed()

    async def start_server(self):
        """Start the Ethernet/IP server."""
        self.logger.info("Starting Bifrost Ethernet/IP Simulator")
        self.logger.info(f"Configuration: host={self.host}")
        self.logger.info(f"UDP Discovery Port: {self.udp_port}")
        self.logger.info(f"TCP Explicit Port: {self.tcp_port}")
        self.logger.info(f"Device: {self.device_name}")
        
        try:
            self.running = True
            
            # Start UDP discovery server
            udp_task = asyncio.create_task(self.handle_udp_discovery())
            
            # Start TCP server for explicit messaging
            self.tcp_server = await asyncio.start_server(
                self.handle_tcp_client,
                self.host,
                self.tcp_port
            )
            
            self.logger.info(f"EtherNet/IP TCP server started on {self.host}:{self.tcp_port}")
            
            # Start stats logging task
            stats_task = asyncio.create_task(self._stats_loop())
            
            # Wait for servers
            async with self.tcp_server:
                await asyncio.gather(
                    self.tcp_server.serve_forever(),
                    udp_task,
                    stats_task,
                    return_exceptions=True
                )
                
        except Exception as e:
            self.logger.error(f"Failed to start EtherNet/IP server: {e}")
            raise

    async def _stats_loop(self):
        """Periodically log statistics."""
        last_stats_time = time.time()
        
        while self.running:
            try:
                await asyncio.sleep(60)  # Log stats every 60 seconds
                
                uptime = time.time() - self.stats["start_time"]
                self.logger.info(
                    f"Stats - Uptime: {uptime:.1f}s, "
                    f"UDP: {self.stats['udp_requests']}, "
                    f"TCP: {self.stats['tcp_connections']}, "
                    f"Total: {self.stats['requests_total']}, "
                    f"Success: {self.stats['requests_successful']}, "
                    f"Failed: {self.stats['requests_failed']}"
                )
                
            except Exception as e:
                self.logger.error(f"Stats loop error: {e}")

    def get_stats(self) -> Dict[str, Any]:
        """Get current server statistics."""
        uptime = time.time() - self.stats["start_time"]
        return {
            **self.stats,
            "uptime_seconds": uptime,
            "device_name": self.device_name,
            "device_identity": self.device_identity
        }

    async def stop_server(self):
        """Stop the server gracefully."""
        self.logger.info("Stopping EtherNet/IP server...")
        self.running = False
        if self.tcp_server:
            self.tcp_server.close()
            await self.tcp_server.wait_closed()


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(description="Bifrost EtherNet/IP Simulator")
    parser.add_argument("--host", default="0.0.0.0", help="Server host")
    parser.add_argument("--port", type=int, default=44818, help="UDP discovery port")
    parser.add_argument("--tcp-port", type=int, default=2222, help="TCP explicit messaging port")
    parser.add_argument("--device-name", default="Bifrost_EIP_Device", help="Device name")
    parser.add_argument("--log-level", default="INFO", choices=["DEBUG", "INFO", "WARNING", "ERROR"])
    
    args = parser.parse_args()
    
    # Create and start server
    server = BifrostEthernetIPServer(
        host=args.host,
        udp_port=args.port,
        tcp_port=args.tcp_port,
        device_name=args.device_name,
        log_level=args.log_level
    )
    
    try:
        asyncio.run(server.start_server())
    except KeyboardInterrupt:
        logging.info("Server stopped by user")
    except Exception as e:
        logging.error(f"Server error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()