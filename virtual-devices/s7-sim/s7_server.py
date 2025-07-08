#!/usr/bin/env python3
"""
Siemens S7 PLC Simulator for Bifrost Virtual Device Testing

This simulator provides a realistic S7 PLC implementation for testing
device discovery and communication protocols.

Features:
- S7 communication protocol
- Memory areas (DB, M, I, Q)
- Realistic PLC behavior
- Device identification responses
"""

import argparse
import asyncio
import logging
import signal
import sys
import time
import threading
from typing import Dict, Any, Optional

try:
    import snap7
    from snap7 import util
except ImportError:
    print("Error: snap7 library not found. Install with: pip install python-snap7")
    sys.exit(1)


class BifrostS7Server:
    """Bifrost S7 PLC simulator."""

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

        # Initialize memory areas
        self._init_memory_areas()

        # Setup signal handlers
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

    def _signal_handler(self, signum, frame):
        """Handle shutdown signals gracefully."""
        self.logger.info(f"Received signal {signum}, shutting down...")
        self.running = False
        if self.server:
            self.server.stop()

    def _init_memory_areas(self):
        """Initialize S7 memory areas with realistic data."""
        try:
            self.server = snap7.server.Server()
            
            # Create memory areas
            # DB1: Process data (100 words)
            db1_data = bytearray(200)  # 100 words = 200 bytes
            util.set_int(db1_data, 0, 1234)  # Production counter
            util.set_real(db1_data, 4, 25.5)  # Temperature
            util.set_real(db1_data, 8, 1.2)   # Pressure
            self.server.register_area(snap7.types.srvAreaDB, 1, db1_data)
            
            # Marker area (M): 100 bytes
            marker_data = bytearray(100)
            marker_data[0] = 0x55  # Some status bits
            self.server.register_area(snap7.types.srvAreaMK, 0, marker_data)
            
            # Input area (I): 50 bytes  
            input_data = bytearray(50)
            input_data[0] = 0xFF  # All inputs on
            self.server.register_area(snap7.types.srvAreaPE, 0, input_data)
            
            # Output area (Q): 50 bytes
            output_data = bytearray(50)
            output_data[0] = 0xAA  # Alternating pattern
            self.server.register_area(snap7.types.srvAreaPA, 0, output_data)
            
            self.logger.info("Initialized S7 memory areas")
            
        except Exception as e:
            self.logger.error(f"Failed to initialize memory areas: {e}")
            raise

    async def start_server(self):
        """Start the S7 server."""
        self.logger.info("Starting Bifrost S7 PLC Simulator")
        self.logger.info(f"Configuration: host={self.host}, port={self.port}")
        self.logger.info(f"Device: {self.device_name}, Rack={self.rack}, Slot={self.slot}")
        
        try:
            # Start the server
            result = self.server.start(tcp_port=self.port)
            if result == 0:
                self.logger.info(f"S7 server started on {self.host}:{self.port}")
                self.running = True
                
                # Run server loop
                await self._run_server_loop()
                
            else:
                self.logger.error(f"Failed to start S7 server. Error code: {result}")
                raise RuntimeError(f"S7 server start failed with code {result}")
                
        except Exception as e:
            self.logger.error(f"Failed to start S7 server: {e}")
            raise

    async def _run_server_loop(self):
        """Main server loop."""
        last_stats_time = time.time()
        
        while self.running:
            try:
                # Check server status
                if self.server.get_status() != snap7.types.SrvStatusStopped:
                    self.stats["requests_successful"] += 1
                else:
                    self.logger.warning("Server status indicates stopped")
                    break
                
                # Update simulated data periodically
                await self._update_simulated_data()
                
                # Log stats every 60 seconds
                current_time = time.time()
                if current_time - last_stats_time >= 60:
                    self._log_stats()
                    last_stats_time = current_time
                
                # Sleep to prevent busy waiting
                await asyncio.sleep(1)
                
            except Exception as e:
                self.logger.error(f"Server loop error: {e}")
                self.stats["requests_failed"] += 1
                await asyncio.sleep(5)

    async def _update_simulated_data(self):
        """Update simulated process data."""
        try:
            # Get current time for simulation
            current_time = time.time()
            elapsed = current_time - self.stats["start_time"]
            
            # Update DB1 with simulated process values
            db1_data = self.server.pick_area(snap7.types.srvAreaDB, 1)
            if db1_data:
                # Increment production counter
                counter = util.get_int(db1_data, 0) + 1
                if counter > 9999:
                    counter = 0
                util.set_int(db1_data, 0, counter)
                
                # Simulate temperature variation (20-30°C)
                import math
                temp = 25.0 + 3.0 * math.sin(elapsed / 60.0) + (elapsed % 10) * 0.1
                util.set_real(db1_data, 4, temp)
                
                # Simulate pressure variation (1.0-1.5 bar)
                pressure = 1.25 + 0.25 * math.cos(elapsed / 45.0)
                util.set_real(db1_data, 8, pressure)
                
                self.stats["requests_total"] += 1
                
        except Exception as e:
            self.logger.error(f"Error updating simulated data: {e}")

    def _log_stats(self):
        """Log current statistics."""
        uptime = time.time() - self.stats["start_time"]
        self.logger.info(
            f"Stats - Uptime: {uptime:.1f}s, "
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
            self.server.stop()


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