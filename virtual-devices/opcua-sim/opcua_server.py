#!/usr/bin/env python3
"""OPC UA Simulator for Bifrost Testing

Provides a realistic OPC UA server with industrial data nodes, subscriptions,
and error simulation for comprehensive testing of the Bifrost framework.
"""

import argparse
import asyncio
import logging
import math
import random
import signal
import sys
import time
from datetime import datetime

from asyncua import Server, ua
from asyncua.common.node import Node


class IndustrialDataSimulator:
    """Simulates realistic industrial data for OPC UA server."""

    def __init__(self):
        self.running = False
        self.start_time = time.time()
        self.nodes: dict[str, Node] = {}

    async def setup_nodes(self, server: Server):
        """Create industrial data nodes."""
        # Get Objects node
        objects = server.get_objects_node()

        # Create device folder
        device_folder = await objects.add_folder(
            "ns=2;s=Factory", "Factory Floor"
        )

        # Temperature sensors
        temp_folder = await device_folder.add_folder(
            "ns=2;s=Temperature", "Temperature Sensors"
        )
        for i in range(5):
            node = await temp_folder.add_variable(
                f"ns=2;s=TempSensor{i + 1}",
                f"Temperature Sensor {i + 1}",
                20.0,
                ua.VariantType.Float,
            )
            await node.set_writable(False)  # Read-only sensor
            self.nodes[f"temp_{i}"] = node

        # Pressure sensors
        pressure_folder = await device_folder.add_folder(
            "ns=2;s=Pressure", "Pressure Sensors"
        )
        for i in range(3):
            node = await pressure_folder.add_variable(
                f"ns=2;s=PressureSensor{i + 1}",
                f"Pressure Sensor {i + 1}",
                1.013,
                ua.VariantType.Float,
            )
            await node.set_writable(False)
            self.nodes[f"pressure_{i}"] = node

        # Flow meters
        flow_folder = await device_folder.add_folder(
            "ns=2;s=Flow", "Flow Meters"
        )
        for i in range(3):
            node = await flow_folder.add_variable(
                f"ns=2;s=FlowMeter{i + 1}",
                f"Flow Meter {i + 1}",
                50.0,
                ua.VariantType.Float,
            )
            await node.set_writable(False)
            self.nodes[f"flow_{i}"] = node

        # Control setpoints (writable)
        control_folder = await device_folder.add_folder(
            "ns=2;s=Control", "Control Setpoints"
        )
        for i in range(3):
            node = await control_folder.add_variable(
                f"ns=2;s=Setpoint{i + 1}",
                f"Setpoint {i + 1}",
                25.0,
                ua.VariantType.Float,
            )
            await node.set_writable(True)
            self.nodes[f"setpoint_{i}"] = node

        # Status indicators
        status_folder = await device_folder.add_folder(
            "ns=2;s=Status", "Status Indicators"
        )
        for i in range(5):
            node = await status_folder.add_variable(
                f"ns=2;s=Status{i + 1}",
                f"Status {i + 1}",
                random.choice([True, False]),
                ua.VariantType.Boolean,
            )
            self.nodes[f"status_{i}"] = node

        # System information
        info_folder = await device_folder.add_folder(
            "ns=2;s=System", "System Information"
        )

        # Runtime counter
        runtime_node = await info_folder.add_variable(
            "ns=2;s=Runtime", "Runtime Hours", 0.0, ua.VariantType.Float
        )
        await runtime_node.set_writable(False)
        self.nodes["runtime"] = runtime_node

        # Last update timestamp
        timestamp_node = await info_folder.add_variable(
            "ns=2;s=LastUpdate",
            "Last Update",
            datetime.now(),
            ua.VariantType.DateTime,
        )
        await timestamp_node.set_writable(False)
        self.nodes["timestamp"] = timestamp_node

        logging.info(f"Created {len(self.nodes)} data nodes")

    async def start_simulation(self):
        """Start the data simulation loop."""
        self.running = True
        logging.info("Starting data simulation")

        while self.running:
            try:
                await self.update_data()
                await asyncio.sleep(1.0)  # Update every second
            except asyncio.CancelledError:
                break
            except Exception as e:
                logging.error(f"Error in simulation loop: {e}")
                await asyncio.sleep(1.0)

    async def stop_simulation(self):
        """Stop the data simulation."""
        self.running = False
        logging.info("Stopping data simulation")

    async def update_data(self):
        """Update all simulated data values."""
        current_time = time.time()
        elapsed = current_time - self.start_time

        # Update temperature sensors (with realistic variation)
        for i in range(5):
            base_temp = 20.0 + i * 2.0
            # Add sine wave variation + random noise
            variation = 3.0 * math.sin(elapsed / 60.0 + i) + random.uniform(
                -1.0, 1.0
            )
            temp = base_temp + variation

            node_key = f"temp_{i}"
            if node_key in self.nodes:
                await self.nodes[node_key].write_value(temp)

        # Update pressure sensors
        for i in range(3):
            base_pressure = 1.013 + i * 0.1
            variation = 0.05 * math.sin(elapsed / 30.0 + i) + random.uniform(
                -0.02, 0.02
            )
            pressure = max(0, base_pressure + variation)

            node_key = f"pressure_{i}"
            if node_key in self.nodes:
                await self.nodes[node_key].write_value(pressure)

        # Update flow meters
        for i in range(3):
            base_flow = 50.0 + i * 10.0
            variation = 10.0 * math.sin(elapsed / 45.0 + i) + random.uniform(
                -5.0, 5.0
            )
            flow = max(0, base_flow + variation)

            node_key = f"flow_{i}"
            if node_key in self.nodes:
                await self.nodes[node_key].write_value(flow)

        # Update status indicators (change occasionally)
        status_change_rate = 0.1  # 10% chance to change each second
        if random.random() < status_change_rate:
            status_index = random.randint(0, 4)
            node_key = f"status_{status_index}"
            if node_key in self.nodes:
                current_value = await self.nodes[node_key].read_value()
                await self.nodes[node_key].write_value(not current_value)

        # Update system runtime
        runtime_hours = elapsed / 3600.0
        if "runtime" in self.nodes:
            await self.nodes["runtime"].write_value(runtime_hours)

        # Update timestamp
        if "timestamp" in self.nodes:
            await self.nodes["timestamp"].write_value(datetime.now())


class BifrostOPCUAServer:
    """Enhanced OPC UA server for Bifrost testing.

    Features:
    - Realistic industrial data simulation
    - Subscription support
    - Error simulation capabilities
    - Security configurations
    """

    def __init__(
        self,
        endpoint: str = "opc.tcp://0.0.0.0:4840",
        namespace: str = "http://bifrost.industrial.example",
    ):
        self.endpoint = endpoint
        self.namespace = namespace
        self.server = Server()
        self.simulator = IndustrialDataSimulator()
        self.simulation_task: asyncio.Task | None = None

        # Statistics
        self.stats = {
            "connections": 0,
            "subscriptions": 0,
            "read_requests": 0,
            "write_requests": 0,
            "start_time": time.time(),
        }

    async def setup_server(self):
        """Configure the OPC UA server."""
        # Set endpoint
        await self.server.init()
        self.server.set_endpoint(self.endpoint)

        # Set server name
        self.server.set_server_name("Bifrost Industrial Simulator")

        # Add namespace
        await self.server.register_namespace(self.namespace)

        # Configure security (allow anonymous for testing)
        self.server.set_security_policy([ua.SecurityPolicyType.NoSecurity])

        # Setup data simulation
        await self.simulator.setup_nodes(self.server)

        logging.info(f"OPC UA server configured at {self.endpoint}")

    async def start(self):
        """Start the OPC UA server."""
        try:
            await self.setup_server()

            # Start the server
            async with self.server:
                logging.info("OPC UA server started")

                # Start data simulation
                self.simulation_task = asyncio.create_task(
                    self.simulator.start_simulation()
                )

                # Keep server running
                try:
                    await self.simulation_task
                except asyncio.CancelledError:
                    logging.info("Simulation cancelled")

        except Exception as e:
            logging.error(f"Server error: {e}")
            raise

    async def stop(self):
        """Stop the server gracefully."""
        logging.info("Stopping OPC UA server...")

        # Stop simulation
        await self.simulator.stop_simulation()
        if self.simulation_task:
            self.simulation_task.cancel()
            try:
                await self.simulation_task
            except asyncio.CancelledError:
                pass

        # Server cleanup is handled by context manager

    def get_statistics(self) -> dict:
        """Get server statistics."""
        uptime = time.time() - self.stats["start_time"]
        return {
            "uptime_seconds": uptime,
            "connections": self.stats["connections"],
            "subscriptions": self.stats["subscriptions"],
            "read_requests": self.stats["read_requests"],
            "write_requests": self.stats["write_requests"],
        }


async def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(description="Bifrost OPC UA Simulator")
    parser.add_argument(
        "--endpoint",
        default="opc.tcp://0.0.0.0:4840",
        help="OPC UA endpoint URL",
    )
    parser.add_argument(
        "--namespace",
        default="http://bifrost.industrial.example",
        help="Server namespace URI",
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

    # Create server
    server = BifrostOPCUAServer(args.endpoint, args.namespace)

    # Setup signal handling
    def signal_handler():
        logging.info("Received shutdown signal")
        asyncio.create_task(server.stop())

    # Setup signal handlers
    loop = asyncio.get_event_loop()
    for sig in [signal.SIGTERM, signal.SIGINT]:
        loop.add_signal_handler(sig, signal_handler)

    # Start server
    logging.info("Starting Bifrost OPC UA Simulator")
    logging.info(f"Endpoint: {args.endpoint}")

    try:
        await server.start()
    except KeyboardInterrupt:
        logging.info("Received keyboard interrupt")
    except Exception as e:
        logging.error(f"Server failed: {e}")
        return 1
    finally:
        await server.stop()

    return 0


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
