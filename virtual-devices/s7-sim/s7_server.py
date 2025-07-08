#!/usr/bin/env python3
"""
Siemens S7 PLC Simulator for Bifrost Virtual Device Testing

This simulator provides a simplified S7 PLC implementation for testing
device discovery and communication protocols. It simulates S7 protocol
responses without requiring the full snap7 library.

Features:
- S7 communication protocol simulation
- Basic TCP connectivity on port 102
- Device identification responses
- Simulated memory areas
"""

import argparse
import asyncio
import logging
import signal
import sys
import time
import struct
from typing import Dict, Any, Optional

class BifrostS7Server:
    """Bifrost S7 PLC simulator (simplified implementation)."""

    def __init__(
        self,
        host: str = "0.0.0.0", 
        port: int = 102,
        rack: int = 0,
        slot: int = 1,
        device_name: str = "Bifrost_S7_PLC",
        log_level: str = "INFO"
    ):
        self.host = host
        self.port = port
        self.rack = rack
        self.slot = slot
        self.device_name = device_name
        self.running = False
        self.server = None
        self.stats = {
            "connections_total": 0,
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

        # Initialize simulated memory areas
        self.memory = {
            "db1": bytearray(200),  # Data block 1 (100 words)
            "markers": bytearray(100),  # Marker area 
            "inputs": bytearray(50),    # Input area
            "outputs": bytearray(50),   # Output area
        }
        
        self._init_memory_data()

        # Setup signal handlers
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

    def _signal_handler(self, signum, frame):
        """Handle shutdown signals gracefully."""
        self.logger.info(f"Received signal {signum}, shutting down...")
        self.running = False

    def _init_memory_data(self):
        """Initialize memory areas with realistic data."""
        # DB1: Process data
        struct.pack_into('>H', self.memory["db1"], 0, 1234)  # Production counter
        struct.pack_into('>f', self.memory["db1"], 4, 25.5)  # Temperature
        struct.pack_into('>f', self.memory["db1"], 8, 1.2)   # Pressure
        
        # Marker area
        self.memory["markers"][0] = 0x55  # Status bits
        
        # Input/Output areas
        self.memory["inputs"][0] = 0xFF   # All inputs on
        self.memory["outputs"][0] = 0xAA  # Alternating pattern

    async def handle_client(self, reader, writer):
        """Handle S7 client connection."""
        client_addr = writer.get_extra_info('peername')
        self.logger.info(f"S7 client connected from {client_addr}")
        self.stats["connections_total"] += 1
        
        try:
            while self.running:
                # Read S7 protocol data (simplified)
                data = await asyncio.wait_for(reader.read(1024), timeout=30.0)
                if not data:
                    break
                
                self.stats["requests_total"] += 1
                self.logger.debug(f"Received {len(data)} bytes from {client_addr}")
                
                # Simple S7 protocol response simulation
                response = await self._process_s7_request(data)
                if response:
                    writer.write(response)
                    await writer.drain()
                    self.stats["requests_successful"] += 1
                else:
                    self.stats["requests_failed"] += 1
                    
        except asyncio.TimeoutError:
            self.logger.debug(f"Client {client_addr} timeout")
        except Exception as e:
            self.logger.error(f"Error handling client {client_addr}: {e}")
            self.stats["requests_failed"] += 1
        finally:
            self.logger.info(f"S7 client {client_addr} disconnected")
            writer.close()
            await writer.wait_closed()

    async def _process_s7_request(self, data: bytes) -> Optional[bytes]:
        """Process S7 protocol request and return response."""
        try:
            # This is a simplified S7 protocol handler
            # In a real implementation, this would parse S7 PDUs
            
            if len(data) < 4:
                return None
            
            # Simple response for any S7 request
            # S7 protocol uses COTP and S7 communication
            # This is a minimal response to keep connections alive
            
            # Check if it looks like an S7 communication setup
            if len(data) >= 22 and data[5:7] == b'\xd0\x01':  # COTP connection request
                # COTP connection confirm
                response = bytearray([
                    0x03, 0x00, 0x00, 0x16,  # TPKT header (length 22)
                    0x11, 0xd0, 0x00, 0x01,  # COTP connection confirm  
                    0x00, 0x00, 0x00, 0x00,  # Parameters
                    0x00, 0x00, 0xc1, 0x02,  # Source TSAP
                    0x01, 0x00, 0xc2, 0x02,  # Dest TSAP  
                    0x02, 0x00                # Additional params
                ])
                return bytes(response)
            
            elif len(data) >= 19 and data[7] == 0x32:  # S7 communication
                # S7 communication response
                response = bytearray([
                    0x03, 0x00, 0x00, 0x1b,  # TPKT header
                    0x02, 0xf0, 0x80,        # COTP data
                    0x32, 0x03, 0x00, 0x00,  # S7 header
                    0x00, 0x00, 0x08, 0x00,  # S7 parameters
                    0x00, 0xf0, 0x00, 0x00,  # S7 data
                    0x01, 0x00, 0x01, 0xff,  # Result
                    0x04, 0x00, 0x08         # Data
                ])
                return bytes(response)
            
            # Default response for other requests
            return b'\x03\x00\x00\x07\x02\xf0\x80'  # Minimal COTP data
            
        except Exception as e:
            self.logger.error(f"Error processing S7 request: {e}")
            return None

    async def start_server(self):
        """Start the S7 server."""
        self.logger.info("Starting Bifrost S7 PLC Simulator")
        self.logger.info(f"Configuration: host={self.host}, port={self.port}")
        self.logger.info(f"Device: {self.device_name}, Rack={self.rack}, Slot={self.slot}")
        
        try:
            self.server = await asyncio.start_server(
                self.handle_client,
                self.host,
                self.port
            )
            
            self.logger.info(f"S7 server started on {self.host}:{self.port}")
            self.running = True
            
            # Start data update task
            asyncio.create_task(self._update_loop())
            
            # Serve until stopped
            async with self.server:
                await self.server.serve_forever()
                
        except Exception as e:
            self.logger.error(f"Failed to start S7 server: {e}")
            raise

    async def _update_loop(self):
        """Periodically update simulated data."""
        last_stats_time = time.time()
        
        while self.running:
            try:
                # Update simulated process data
                await self._update_simulated_data()
                
                # Log stats every 60 seconds
                current_time = time.time()
                if current_time - last_stats_time >= 60:
                    self._log_stats()
                    last_stats_time = current_time
                
                await asyncio.sleep(5)  # Update every 5 seconds
                
            except Exception as e:
                self.logger.error(f"Error in update loop: {e}")
                await asyncio.sleep(10)

    async def _update_simulated_data(self):
        """Update simulated process data."""
        try:
            current_time = time.time()
            elapsed = current_time - self.stats["start_time"]
            
            # Update production counter
            counter = struct.unpack('>H', self.memory["db1"][0:2])[0] + 1
            if counter > 9999:
                counter = 0
            struct.pack_into('>H', self.memory["db1"], 0, counter)
            
            # Simulate temperature variation (20-30°C)
            import math
            temp = 25.0 + 3.0 * math.sin(elapsed / 60.0) + (elapsed % 10) * 0.1
            struct.pack_into('>f', self.memory["db1"], 4, temp)
            
            # Simulate pressure variation (1.0-1.5 bar)
            pressure = 1.25 + 0.25 * math.cos(elapsed / 45.0)
            struct.pack_into('>f', self.memory["db1"], 8, pressure)
            
        except Exception as e:
            self.logger.error(f"Error updating simulated data: {e}")

    def _log_stats(self):
        """Log current statistics."""
        uptime = time.time() - self.stats["start_time"]
        self.logger.info(
            f"Stats - Uptime: {uptime:.1f}s, "
            f"Connections: {self.stats['connections_total']}, "
            f"Requests: {self.stats['requests_total']}, "
            f"Success: {self.stats['requests_successful']}, "
            f"Failed: {self.stats['requests_failed']}"
        )

    def get_stats(self) -> Dict[str, Any]:
        """Get current server statistics."""
        uptime = time.time() - self.stats["start_time"]
        return {
            **self.stats,
            "uptime_seconds": uptime,
            "device_name": self.device_name,
            "rack": self.rack,
            "slot": self.slot
        }

    async def stop_server(self):
        """Stop the server gracefully."""
        self.logger.info("Stopping S7 server...")
        self.running = False
        if self.server:
            self.server.close()
            await self.server.wait_closed()


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(description="Bifrost S7 PLC Simulator")
    parser.add_argument("--host", default="0.0.0.0", help="Server host")
    parser.add_argument("--port", type=int, default=102, help="Server port")
    parser.add_argument("--rack", type=int, default=0, help="PLC rack number")
    parser.add_argument("--slot", type=int, default=1, help="PLC slot number")
    parser.add_argument("--device-name", default="Bifrost_S7_PLC", help="Device name")
    parser.add_argument("--log-level", default="INFO", choices=["DEBUG", "INFO", "WARNING", "ERROR"])
    
    args = parser.parse_args()
    
    # Create and start server
    server = BifrostS7Server(
        host=args.host,
        port=args.port,
        rack=args.rack,
        slot=args.slot,
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