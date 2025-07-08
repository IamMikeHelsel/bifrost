#!/usr/bin/env python3
"""
Ethernet/IP (EtherNet/IP) Simulator for Bifrost Virtual Device Testing

This simulator provides a realistic Ethernet/IP device implementation for testing
device discovery and communication protocols.

Features:
- CIP (Common Industrial Protocol) over Ethernet/IP
- Identity Object support for device discovery
- Realistic industrial device behavior
- Configurable device parameters
"""

import argparse
import asyncio
import logging
import signal
import sys
import time
import json
from typing import Dict, Any, Optional

try:
    import cpppo
    from cpppo.server.enip import main as enip_main
    from cpppo.server.enip.get_attribute import proxy_simple as enip_proxy
except ImportError:
    print("Error: cpppo library not found. Install with: pip install cpppo")
    sys.exit(1)


class BifrostEthernetIPServer:
    """Bifrost Ethernet/IP device simulator."""

    def __init__(
        self,
        host: str = "0.0.0.0",
        port: int = 44818,
        device_name: str = "Bifrost_EIP_Device",
        log_level: str = "INFO"
    ):
        self.host = host
        self.port = port
        self.device_name = device_name
        self.running = False
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

        # Device identity information
        self.device_identity = {
            "vendor_id": 0x01F7,  # Custom vendor ID
            "device_type": 0x00,  # Generic Device
            "product_code": 0x0001,
            "revision": {"major": 1, "minor": 0},
            "status": 0x0000,
            "serial_number": 0x12345678,
            "product_name": device_name
        }

        # Setup signal handlers
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

    def _signal_handler(self, signum, frame):
        """Handle shutdown signals gracefully."""
        self.logger.info(f"Received signal {signum}, shutting down...")
        self.running = False

    async def start_server(self):
        """Start the Ethernet/IP server."""
        self.logger.info("Starting Bifrost Ethernet/IP Simulator")
        self.logger.info(f"Configuration: host={self.host}, port={self.port}")
        self.logger.info(f"Device: {self.device_name}")
        
        try:
            # Prepare ENIP server arguments
            enip_args = [
                '--address', f'{self.host}:{self.port}',
                '--print',  # Enable debug printing
                '--log', '1',  # Logging level
                'SCADA_40001=INT[1000]'  # Define some tags for testing
            ]
            
            self.logger.info(f"Ethernet/IP server starting on {self.host}:{self.port}")
            self.running = True
            
            # Run the ENIP server in a separate thread/process
            # Note: cpppo's enip_main is synchronous, so we'll need to adapt it
            await self._run_enip_server(enip_args)
            
        except Exception as e:
            self.logger.error(f"Failed to start Ethernet/IP server: {e}")
            raise

    async def _run_enip_server(self, args):
        """Run the ENIP server with proper async handling."""
        try:
            # This is a simplified version - in practice, you'd want to
            # integrate more deeply with cpppo's async capabilities
            loop = asyncio.get_event_loop()
            
            # Run enip server in executor to avoid blocking
            await loop.run_in_executor(
                None, 
                lambda: enip_main(argv=args)
            )
            
        except Exception as e:
            self.logger.error(f"ENIP server error: {e}")
            self.stats["requests_failed"] += 1

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
        self.logger.info("Stopping Ethernet/IP server...")
        self.running = False


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(description="Bifrost Ethernet/IP Simulator")
    parser.add_argument("--host", default="0.0.0.0", help="Server host")
    parser.add_argument("--port", type=int, default=44818, help="Server port")
    parser.add_argument("--device-name", default="Bifrost_EIP_Device", help="Device name")
    parser.add_argument("--log-level", default="INFO", choices=["DEBUG", "INFO", "WARNING", "ERROR"])
    
    args = parser.parse_args()
    
    # Create and start server
    server = BifrostEthernetIPServer(
        host=args.host,
        port=args.port,
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