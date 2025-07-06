#!/usr/bin/env python3
"""Modbus TCP Simulator for Bifrost Testing

This simulator provides a realistic Modbus TCP device for testing the Bifrost
framework. It includes error simulation, realistic timing, and configurable
device behavior.
"""

import argparse
import logging
import math
import random
import signal
import sys
import threading
import time

from pyModbusTCP.server import DataBank, ModbusServer


class SimulatedDevice:
    """Represents a simulated industrial device with realistic behavior."""

    def __init__(self, device_id: str, config: dict):
        self.device_id = device_id
        self.config = config
        self.registers = {}
        self.coils = {}
        self.running = False
        self.last_update = time.time()

        # Initialize registers with default values
        self._init_registers()

        # Setup periodic updates for dynamic data
        self.update_thread = threading.Thread(
            target=self._update_loop, daemon=True
        )

    def _init_registers(self):
        """Initialize registers with realistic industrial data."""
        # Temperature sensors (40001-40010)
        for i in range(10):
            self.registers[i] = int(20.0 * 100)  # 20°C in 0.01°C units

        # Pressure sensors (40011-40020)
        for i in range(10, 20):
            self.registers[i] = int(1.013 * 1000)  # 1.013 bar in mbar

        # Flow rates (40021-40030)
        for i in range(20, 30):
            self.registers[i] = int(50.0 * 10)  # 50 L/min in 0.1 L/min units

        # Status coils (00001-00010)
        for i in range(10):
            self.coils[i] = random.choice([True, False])

    def _update_loop(self):
        """Simulate realistic sensor value changes."""
        while self.running:
            try:
                self._update_sensors()
                time.sleep(1.0)  # Update every second
            except Exception as e:
                logging.error(f"Error in update loop: {e}")

    def _update_sensors(self):
        """Update sensor values with realistic variations."""
        current_time = time.time()

        # Temperature variation (±2°C)
        for i in range(10):
            base_temp = 20.0 + i * 0.5  # Each sensor has slight offset
            variation = random.uniform(-2.0, 2.0)
            # Add daily cycle simulation
            daily_cycle = 5.0 * math.sin(
                (current_time % 86400) / 86400 * 2 * math.pi
            )
            temp = base_temp + variation + daily_cycle
            self.registers[i] = max(0, int(temp * 100))

        # Pressure variation (±0.1 bar)
        for i in range(10, 20):
            base_pressure = 1.013 + (i - 10) * 0.01
            variation = random.uniform(-0.1, 0.1)
            pressure = base_pressure + variation
            self.registers[i] = max(0, int(pressure * 1000))

        # Flow rate variation (±5 L/min)
        for i in range(20, 30):
            base_flow = 50.0 + (i - 20) * 2.0
            variation = random.uniform(-5.0, 5.0)
            flow = max(0, base_flow + variation)
            self.registers[i] = int(flow * 10)

    def start(self):
        """Start the device simulation."""
        self.running = True
        self.update_thread.start()
        logging.info(f"Started device simulation: {self.device_id}")

    def stop(self):
        """Stop the device simulation."""
        self.running = False
        if self.update_thread.is_alive():
            self.update_thread.join()
        logging.info(f"Stopped device simulation: {self.device_id}")


class BifrostModbusServer:
    """Enhanced Modbus TCP server for Bifrost testing.

    Features:
    - Realistic device simulation
    - Error injection for testing
    - Performance monitoring
    - Graceful error handling
    """

    def __init__(
        self, host: str = "0.0.0.0", port: int = 502, unit_id: int = 1
    ):
        self.host = host
        self.port = port
        self.unit_id = unit_id

        # Initialize Modbus server
        self.server = ModbusServer(host=host, port=port, no_block=True)

        # Device simulation
        self.device = SimulatedDevice("factory_plc_001", {})

        # Error simulation settings
        self.error_rate = 0.0  # 0% error rate by default
        self.response_delay = 0.0  # No artificial delay by default

        # Statistics
        self.stats = {
            "requests_total": 0,
            "requests_successful": 0,
            "requests_failed": 0,
            "start_time": time.time(),
        }

        # Setup graceful shutdown
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

        # Setup custom hook for request handling
        self.server.data_bank = DataBank()
        self._setup_data_hooks()

    def _setup_data_hooks(self):
        """Setup hooks for realistic data behavior."""
        # Override default data bank behavior

        def custom_get_holding_registers(
            address: int, nb_reg: int
        ) -> list[int] | None:
            """Custom holding register read with error simulation."""
            self.stats["requests_total"] += 1

            # Simulate communication errors
            if random.random() < self.error_rate:
                self.stats["requests_failed"] += 1
                logging.warning(
                    f"Simulated communication error for address {address}"
                )
                return None

            # Simulate response delay
            if self.response_delay > 0:
                time.sleep(self.response_delay)

            # Return realistic data from our device simulation
            try:
                result = []
                for i in range(nb_reg):
                    reg_addr = address + i
                    value = self.device.registers.get(reg_addr, 0)
                    result.append(value)

                self.stats["requests_successful"] += 1
                return result

            except Exception as e:
                logging.error(
                    f"Error reading registers {address}-{address + nb_reg - 1}: {e}"
                )
                self.stats["requests_failed"] += 1
                return None

        def custom_set_holding_registers(
            address: int, values: list[int]
        ) -> bool:
            """Custom holding register write with validation."""
            self.stats["requests_total"] += 1

            # Simulate communication errors
            if random.random() < self.error_rate:
                self.stats["requests_failed"] += 1
                return False

            # Validate write ranges (some registers are read-only)
            readonly_limit = 30  # Sensor data is read-only
            if address < readonly_limit:
                logging.warning(
                    f"Attempt to write to read-only address {address}"
                )
                self.stats["requests_failed"] += 1
                return False

            # Update device registers
            try:
                for i, value in enumerate(values):
                    self.device.registers[address + i] = value

                self.stats["requests_successful"] += 1
                return True

            except Exception as e:
                logging.error(f"Error writing registers {address}: {e}")
                self.stats["requests_failed"] += 1
                return False

        # Install custom handlers
        self.server.data_bank.get_holding_registers = (
            custom_get_holding_registers
        )
        self.server.data_bank.set_holding_registers = (
            custom_set_holding_registers
        )

    def set_error_simulation(
        self, error_rate: float, response_delay: float = 0.0
    ):
        """Configure error simulation for testing error handling."""
        self.error_rate = max(0.0, min(1.0, error_rate))
        self.response_delay = max(0.0, response_delay)
        logging.info(
            f"Error simulation: {self.error_rate * 100:.1f}% error rate, "
            f"{self.response_delay * 1000:.1f}ms delay"
        )

    def get_statistics(self) -> dict:
        """Get server performance statistics."""
        uptime = time.time() - self.stats["start_time"]
        success_rate = 0.0
        if self.stats["requests_total"] > 0:
            success_rate = (
                self.stats["requests_successful"] / self.stats["requests_total"]
            )

        return {
            "uptime_seconds": uptime,
            "requests_total": self.stats["requests_total"],
            "requests_successful": self.stats["requests_successful"],
            "requests_failed": self.stats["requests_failed"],
            "success_rate": success_rate,
            "requests_per_second": self.stats["requests_total"]
            / max(uptime, 1.0),
        }

    def start(self):
        """Start the Modbus server and device simulation."""
        try:
            # Start device simulation
            self.device.start()

            # Start Modbus server
            self.server.start()
            logging.info(
                f"Modbus TCP server started on {self.host}:{self.port}"
            )

            # Main loop
            try:
                while True:
                    time.sleep(1)

                    # Log statistics periodically
                    if (
                        self.stats["requests_total"] % 100 == 0
                        and self.stats["requests_total"] > 0
                    ):
                        stats = self.get_statistics()
                        logging.info(
                            f"Stats: {stats['requests_total']} requests, "
                            f"{stats['success_rate'] * 100:.1f}% success, "
                            f"{stats['requests_per_second']:.1f} req/s"
                        )

            except KeyboardInterrupt:
                logging.info("Received shutdown signal")

        except Exception as e:
            logging.error(f"Server error: {e}")
            raise

        finally:
            self.stop()

    def stop(self):
        """Stop the server and clean up resources."""
        logging.info("Shutting down Modbus server...")

        # Stop device simulation
        self.device.stop()

        # Stop Modbus server
        self.server.stop()

        # Log final statistics
        stats = self.get_statistics()
        logging.info(f"Final stats: {stats}")

    def _signal_handler(self, signum, _frame):
        """Handle shutdown signals gracefully."""
        logging.info(f"Received signal {signum}")
        sys.exit(0)


def main():
    """Main entry point with CLI configuration."""
    parser = argparse.ArgumentParser(description="Bifrost Modbus TCP Simulator")
    parser.add_argument("--host", default="0.0.0.0", help="Host to bind to")
    parser.add_argument("--port", type=int, default=502, help="Port to bind to")
    parser.add_argument("--unit-id", type=int, default=1, help="Modbus unit ID")
    parser.add_argument(
        "--error-rate",
        type=float,
        default=0.0,
        help="Error simulation rate (0.0-1.0)",
    )
    parser.add_argument(
        "--response-delay",
        type=float,
        default=0.0,
        help="Artificial response delay in seconds",
    )
    parser.add_argument(
        "--log-level",
        default="INFO",
        choices=["DEBUG", "INFO", "WARNING", "ERROR"],
        help="Logging level",
    )

    args = parser.parse_args()

    # Setup logging
    logging.basicConfig(
        level=getattr(logging, args.log_level),
        format="%(asctime)s - %(levelname)s - %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    )

    # Create and start server
    server = BifrostModbusServer(args.host, args.port, args.unit_id)

    # Configure error simulation
    if args.error_rate > 0 or args.response_delay > 0:
        server.set_error_simulation(args.error_rate, args.response_delay)

    # Start server
    logging.info("Starting Bifrost Modbus TCP Simulator")
    logging.info(
        f"Configuration: host={args.host}, port={args.port}, "
        f"unit_id={args.unit_id}"
    )

    try:
        server.start()
    except Exception as e:
        logging.error(f"Failed to start server: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
