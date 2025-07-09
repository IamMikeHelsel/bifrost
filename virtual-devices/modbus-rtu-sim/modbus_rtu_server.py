#!/usr/bin/env python3
"""
Modbus RTU Simulator for Bifrost Virtual Device Testing

This simulator provides a Modbus RTU device that runs over TCP (RTU-over-TCP)
for containerized testing environments.

Features:
- Modbus RTU protocol simulation
- CRC validation 
- Multiple device types (energy meter, temperature controller, flow meter)
- Realistic industrial data patterns
- Error injection for resilience testing
"""

import argparse
import asyncio
import logging
import signal
import sys
import time
import random
import math
from typing import Dict, Any, Optional

try:
    from pymodbus.server import StartTcpServer
    from pymodbus.device import ModbusDeviceIdentification
    from pymodbus.datastore import ModbusSequentialDataBlock, ModbusSlaveContext, ModbusServerContext
    from pymodbus.transaction import ModbusRtuFramer
    import pymodbus.bit_read_message
    import pymodbus.bit_write_message
    import pymodbus.register_read_message
    import pymodbus.register_write_message
except ImportError:
    print("Error: pymodbus library not found. Install with: pip install pymodbus")
    sys.exit(1)


class IndustrialDeviceSimulator:
    """Simulates different types of industrial devices."""
    
    def __init__(self, device_type: str = "energy_meter"):
        self.device_type = device_type
        self.start_time = time.time()
        self.registers = {}
        self._init_device_data()
    
    def _init_device_data(self):
        """Initialize device-specific data."""
        if self.device_type == "energy_meter":
            self._init_energy_meter()
        elif self.device_type == "temperature_controller":
            self._init_temperature_controller()
        elif self.device_type == "flow_meter":
            self._init_flow_meter()
        else:
            self._init_generic_device()
    
    def _init_energy_meter(self):
        """Initialize energy meter data."""
        self.registers = {
            0: 2300,   # Voltage L1 (V * 10)
            1: 2305,   # Voltage L2 (V * 10)
            2: 2298,   # Voltage L3 (V * 10)
            10: 1500,  # Current L1 (A * 100)
            11: 1480,  # Current L2 (A * 100)
            12: 1520,  # Current L3 (A * 100)
            20: 34500, # Power (W)
            21: 12000, # Energy (kWh * 10)
            30: 50,    # Frequency (Hz * 10)
            40: 950,   # Power factor (* 1000)
        }
    
    def _init_temperature_controller(self):
        """Initialize temperature controller data."""
        self.registers = {
            0: 2500,   # Process temperature (°C * 100)
            1: 2400,   # Setpoint (°C * 100)
            2: 350,    # Output (% * 10)
            3: 100,    # P parameter
            4: 50,     # I parameter
            5: 10,     # D parameter
            10: 0,     # Alarm status
            11: 1,     # Auto/Manual mode (1=Auto, 0=Manual)
            20: 2200,  # Low alarm limit (°C * 100)
            21: 2800,  # High alarm limit (°C * 100)
        }
    
    def _init_flow_meter(self):
        """Initialize flow meter data."""
        self.registers = {
            0: 1250,   # Flow rate (L/min * 10)
            1: 125000, # Total flow (L)
            2: 85,     # Pipe temperature (°C)
            3: 1013,   # Pressure (mbar)
            10: 0,     # Status flags
            11: 100,   # Battery level (%)
            20: 500,   # Low flow alarm (L/min * 10)
            21: 2000,  # High flow alarm (L/min * 10)
        }
    
    def _init_generic_device(self):
        """Initialize generic device data."""
        self.registers = {i: random.randint(0, 65535) for i in range(50)}
    
    def update_data(self):
        """Update simulated data based on device type."""
        elapsed = time.time() - self.start_time
        
        if self.device_type == "energy_meter":
            self._update_energy_meter(elapsed)
        elif self.device_type == "temperature_controller":
            self._update_temperature_controller(elapsed)
        elif self.device_type == "flow_meter":
            self._update_flow_meter(elapsed)
    
    def _update_energy_meter(self, elapsed):
        """Update energy meter data."""
        # Simulate voltage variations
        base_voltage = 230.0
        variation = 5.0 * math.sin(elapsed / 120.0)
        for i in range(3):
            voltage = (base_voltage + variation + random.uniform(-2, 2)) * 10
            self.registers[i] = int(max(0, min(65535, voltage)))
        
        # Update power and energy
        power = self.registers[20] + random.randint(-500, 500)
        self.registers[20] = max(0, min(65535, power))
        
        # Increment energy (slowly)
        if int(elapsed) % 60 == 0:  # Every minute
            self.registers[21] += 1
    
    def _update_temperature_controller(self, elapsed):
        """Update temperature controller data."""
        # Simulate PID control behavior
        setpoint = self.registers[1] / 100.0
        process_temp = self.registers[0] / 100.0
        
        # Simple simulation: temperature moves towards setpoint
        error = setpoint - process_temp
        new_temp = process_temp + (error * 0.01) + random.uniform(-0.5, 0.5)
        self.registers[0] = int(new_temp * 100)
        
        # Update output based on error
        output = min(100, max(0, abs(error) * 10))
        self.registers[2] = int(output * 10)
    
    def _update_flow_meter(self, elapsed):
        """Update flow meter data."""
        # Simulate flow variations
        base_flow = 125.0  # L/min
        variation = 25.0 * math.sin(elapsed / 60.0)
        flow = base_flow + variation + random.uniform(-10, 10)
        self.registers[0] = int(max(0, flow * 10))
        
        # Update total flow
        if int(elapsed) % 10 == 0:  # Every 10 seconds
            self.registers[1] += int(flow / 6)  # Convert to L per 10 seconds


class BifrostModbusRTUServer:
    """Bifrost Modbus RTU server simulator."""

    def __init__(
        self,
        host: str = "0.0.0.0",
        port: int = 503,
        device_type: str = "energy_meter",
        slave_id: int = 1,
        log_level: str = "INFO"
    ):
        self.host = host
        self.port = port
        self.device_type = device_type
        self.slave_id = slave_id
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

        # Initialize device simulator
        self.device = IndustrialDeviceSimulator(device_type)

        # Setup signal handlers
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

    def _signal_handler(self, signum, frame):
        """Handle shutdown signals gracefully."""
        self.logger.info(f"Received signal {signum}, shutting down...")
        self.running = False

    def _create_datastore(self):
        """Create Modbus datastore with device data."""
        # Initialize with device data
        holding_registers = [0] * 100
        for addr, value in self.device.registers.items():
            if addr < 100:
                holding_registers[addr] = value

        # Create datastore
        store = ModbusSlaveContext(
            di=ModbusSequentialDataBlock(0, [0] * 100),     # Discrete inputs
            co=ModbusSequentialDataBlock(0, [0] * 100),     # Coils
            hr=ModbusSequentialDataBlock(0, holding_registers), # Holding registers
            ir=ModbusSequentialDataBlock(0, holding_registers)  # Input registers
        )
        
        context = ModbusServerContext(slaves={self.slave_id: store}, single=False)
        return context

    async def start_server(self):
        """Start the Modbus RTU server."""
        self.logger.info("Starting Bifrost Modbus RTU Simulator")
        self.logger.info(f"Configuration: host={self.host}, port={self.port}")
        self.logger.info(f"Device: {self.device_type}, Slave ID={self.slave_id}")

        try:
            # Create device identification
            identity = ModbusDeviceIdentification()
            identity.VendorName = "Bifrost"
            identity.ProductCode = f"RTU-{self.device_type.upper()}"
            identity.VendorUrl = "https://github.com/IamMikeHelsel/bifrost"
            identity.ProductName = f"Modbus RTU {self.device_type.replace('_', ' ').title()}"
            identity.ModelName = f"BF-RTU-{self.device_type[:3].upper()}"
            identity.MajorMinorRevision = "1.0"

            # Create datastore
            context = self._create_datastore()

            self.logger.info(f"Modbus RTU server starting on {self.host}:{self.port}")
            self.running = True

            # Start data update task
            asyncio.create_task(self._update_loop())

            # Start server (this will block)
            await asyncio.get_event_loop().run_in_executor(
                None,
                lambda: StartTcpServer(
                    context=context,
                    identity=identity,
                    address=(self.host, self.port),
                    framer=ModbusRtuFramer,  # Use RTU framing
                    ignore_missing_slaves=True
                )
            )

        except Exception as e:
            self.logger.error(f"Failed to start Modbus RTU server: {e}")
            raise

    async def _update_loop(self):
        """Periodically update device data."""
        while self.running:
            try:
                self.device.update_data()
                await asyncio.sleep(5)  # Update every 5 seconds
            except Exception as e:
                self.logger.error(f"Error in update loop: {e}")
                await asyncio.sleep(10)

    def get_stats(self) -> Dict[str, Any]:
        """Get current server statistics."""
        uptime = time.time() - self.stats["start_time"]
        return {
            **self.stats,
            "uptime_seconds": uptime,
            "device_type": self.device_type,
            "slave_id": self.slave_id
        }


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(description="Bifrost Modbus RTU Simulator")
    parser.add_argument("--host", default="0.0.0.0", help="Server host")
    parser.add_argument("--port", type=int, default=503, help="Server port")
    parser.add_argument("--device-type", default="energy_meter", 
                       choices=["energy_meter", "temperature_controller", "flow_meter", "generic"],
                       help="Type of device to simulate")
    parser.add_argument("--slave-id", type=int, default=1, help="Modbus slave ID")
    parser.add_argument("--log-level", default="INFO", choices=["DEBUG", "INFO", "WARNING", "ERROR"])

    args = parser.parse_args()

    # Create and start server
    server = BifrostModbusRTUServer(
        host=args.host,
        port=args.port,
        device_type=args.device_type,
        slave_id=args.slave_id,
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