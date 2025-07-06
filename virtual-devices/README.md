# Virtual Device Testing Framework

## Overview

This directory contains virtual devices and testing infrastructure for comprehensive end-to-end, integration, and functional testing of Bifrost's industrial IoT capabilities. The framework simulates real industrial environments and devices to enable reliable testing without requiring physical hardware.

## Current Implementation (Phase 1)

The following simulators are currently implemented and ready for use:

### Modbus TCP Simulator
- **Location**: `modbus-tcp-sim/`
- **Features**: Realistic device simulation with error injection, dynamic sensor data
- **Docker**: Ready-to-use container with health checks
- **Port**: 502 (standard), 503 (faulty variant)

### OPC UA Simulator  
- **Location**: `opcua-sim/`
- **Features**: Industrial node hierarchy, real-time data updates, subscription support
- **Docker**: Ready-to-use container with health checks
- **Port**: 4840

### Quick Start
```bash
# Start all simulators
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all simulators  
docker-compose down
```

### Testing Your Implementation
```bash
# Test Modbus TCP
python -c "
from pyModbusTCP.client import ModbusClient
client = ModbusClient(host='localhost', port=502)
client.open()
print('Temperature sensors:', client.read_holding_registers(0, 10))
client.close()
"

# Test OPC UA
python -c "
from opcua import Client
client = Client('opc.tcp://localhost:4840')
client.connect()
factory = client.get_node('ns=2;s=Factory')
print('Factory nodes:', [child.get_browse_name() for child in factory.get_children()])
client.disconnect()
"
```

See the detailed documentation sections below for the complete framework vision and future development plans.

## Directory Structure

```
virtual-devices/
├── README.md                    # This documentation
├── simulators/                  # Full device simulators
│   ├── modbus/                 # Modbus TCP/RTU simulators
│   ├── opcua/                  # OPC UA server simulators
│   ├── plc/                    # Generic PLC simulators
│   ├── ethernet_ip/            # Ethernet/IP device simulators
│   └── s7/                     # Siemens S7 simulators
├── mocks/                      # Lightweight mocks for unit tests
│   ├── modbus/                 # Modbus protocol mocks
│   ├── opcua/                  # OPC UA mocks
│   ├── plc/                    # PLC communication mocks
│   ├── ethernet_ip/            # Ethernet/IP mocks
│   └── s7/                     # S7 protocol mocks
├── scenarios/                  # Pre-configured industrial scenarios
│   ├── factory_floor/          # Manufacturing line scenarios
│   ├── process_control/        # Process industry scenarios
│   ├── scada/                  # SCADA system scenarios
│   ├── digital_twin/           # Digital twin sync scenarios
│   └── edge_gateway/           # Edge gateway scenarios
├── network/                    # Network condition simulators
│   ├── latency/                # Network latency simulation
│   ├── packet_loss/            # Packet loss simulation
│   ├── bandwidth/              # Bandwidth limiting
│   └── disconnection/          # Connection failure simulation
├── fixtures/                   # Test data and configurations
│   ├── configs/                # Device configurations
│   ├── data/                   # Sample data sets
│   └── certificates/           # Test certificates for security
└── benchmarks/                 # Performance testing scenarios
    ├── throughput/             # Throughput testing
    ├── latency/                # Latency testing
    ├── concurrent/             # Concurrent connection testing
    └── stress/                 # Stress testing scenarios
```

## Testing Strategy

### 1. Unit Testing with Mocks

**Purpose**: Fast, isolated testing of individual components
**Location**: `mocks/`
**Characteristics**:
- Lightweight, in-memory implementations
- Predictable behavior for edge cases
- No network dependencies
- Fast execution (< 1ms per test)

**Example**:
```python
from virtual_devices.mocks.modbus import MockModbusDevice

async def test_modbus_connection():
    mock_device = MockModbusDevice(address="127.0.0.1", port=502)
    async with mock_device:
        # Test connection logic
        pass
```

### 2. Integration Testing with Simulators

**Purpose**: Test protocol implementations against realistic device behavior
**Location**: `simulators/`
**Characteristics**:
- Full protocol implementations
- Realistic timing and behavior
- Network-based communication
- Stateful device simulation

**Example**:
```python
from virtual_devices.simulators.modbus import ModbusSimulator

async def test_modbus_bulk_read():
    simulator = ModbusSimulator(port=5020)
    await simulator.start()
    
    # Test against running simulator
    # Validates actual network protocol behavior
    
    await simulator.stop()
```

### 3. End-to-End Testing with Scenarios

**Purpose**: Test complete workflows in realistic industrial environments
**Location**: `scenarios/`
**Characteristics**:
- Multi-device, multi-protocol setups
- Realistic data patterns and timing
- Complex interaction scenarios
- Performance under realistic loads

**Example**:
```python
from virtual_devices.scenarios.factory_floor import FactoryFloorScenario

async def test_factory_floor_monitoring():
    scenario = FactoryFloorScenario()
    await scenario.setup()
    
    # Test complete factory monitoring workflow
    # Multiple PLCs, HMI, SCADA integration
    
    await scenario.teardown()
```

### 4. Performance Testing with Benchmarks

**Purpose**: Validate performance targets and identify bottlenecks
**Location**: `benchmarks/`
**Characteristics**:
- High-throughput scenarios
- Concurrent connection testing
- Stress testing under load
- Performance regression detection

**Example**:
```python
from virtual_devices.benchmarks.throughput import ThroughputBenchmark

async def test_10k_tags_per_second():
    benchmark = ThroughputBenchmark(
        protocol="opcua",
        tag_count=10000,
        target_rate=10000  # tags/second
    )
    
    result = await benchmark.run()
    assert result.rate >= 10000
    assert result.latency_p95 < 10  # ms
```

### 5. Network Condition Testing

**Purpose**: Test resilience under various network conditions
**Location**: `network/`
**Characteristics**:
- Latency injection (1ms - 1000ms)
- Packet loss simulation (0.1% - 10%)
- Bandwidth limiting (56K - 1Gbps)
- Connection failure scenarios

**Example**:
```python
from virtual_devices.network import NetworkConditioner

async def test_high_latency_resilience():
    conditioner = NetworkConditioner(latency="500ms", packet_loss="2%")
    
    async with conditioner:
        # Test protocol behavior under poor network conditions
        pass
```

## Device Simulators

### Modbus Simulators

**TCP Simulator**:
- Supports all function codes (1-23)
- Configurable register maps
- Realistic timing behavior
- Exception handling simulation

**RTU Simulator**:
- Serial communication over TCP
- CRC validation
- Configurable baud rates
- Realistic serial timing

### OPC UA Simulators

**Server Simulator**:
- Full OPC UA server implementation
- Configurable node namespace
- All security policies
- Subscription and monitoring
- Historical data access

**Client Simulator**:
- Simulates OPC UA client behavior
- Configurable browsing patterns
- Subscription management
- Certificate handling

### PLC Simulators

**Generic PLC**:
- Protocol-agnostic interface
- Configurable I/O points
- Realistic scan cycles
- Fault injection capabilities

**Manufacturer-Specific**:
- Siemens S7 behavior
- Allen-Bradley ControlLogix
- Schneider Electric Modicon
- Omron CJ/CS series

## Industrial Scenarios

### Factory Floor Scenario

**Components**:
- 5x Modbus PLCs (different manufacturers)
- 2x OPC UA servers (process data)
- 1x Ethernet/IP scanner
- HMI data collection
- SCADA system integration

**Data Patterns**:
- 1000+ data points
- Mixed update rates (1Hz - 100Hz)
- Realistic process values
- Alarm and event generation

### Process Control Scenario

**Components**:
- Distributed control system (DCS)
- Safety instrumented system (SIS)
- Historian integration
- Recipe management
- Batch control

**Characteristics**:
- High-precision analog values
- Regulatory compliance patterns
- Safety interlock simulation
- Historical data archiving

### Digital Twin Scenario

**Components**:
- Real-time equipment models
- Physics-based simulations
- Predictive maintenance data
- Cloud synchronization
- Machine learning integration

**Characteristics**:
- High-frequency data (>1kHz)
- Complex equipment models
- Multi-protocol communication
- Edge-to-cloud synchronization

## Performance Targets

### Throughput Targets

| Protocol | Single Connection | Concurrent (100) | Notes |
|----------|------------------|------------------|--------|
| Modbus TCP | 1000 regs/sec | 100,000 regs/sec | 502 port |
| OPC UA | 10,000 tags/sec | 1,000,000 tags/sec | Subscriptions |
| Ethernet/IP | 5,000 tags/sec | 500,000 tags/sec | CIP protocol |
| S7 | 2,000 tags/sec | 200,000 tags/sec | Siemens PLCs |

### Latency Targets

| Operation | Target | Maximum | Notes |
|-----------|--------|---------|--------|
| Single register read | < 1ms | < 5ms | Local network |
| Bulk read (100 regs) | < 10ms | < 50ms | Optimized |
| OPC UA browse | < 100ms | < 500ms | 1000 nodes |
| Connection setup | < 500ms | < 2000ms | Including auth |

### Resource Targets

| Resource | Target | Maximum | Notes |
|----------|--------|---------|--------|
| Memory usage | < 10MB | < 50MB | Per simulator |
| CPU usage | < 5% | < 20% | Single core |
| Network bandwidth | < 1Mbps | < 10Mbps | Typical load |
| File descriptors | < 100 | < 1000 | Per simulator |

## Usage Examples

### Running Individual Simulators

```bash
# Start Modbus TCP simulator
python -m virtual_devices.simulators.modbus.tcp --port 502

# Start OPC UA server simulator
python -m virtual_devices.simulators.opcua.server --port 4840

# Start with custom configuration
python -m virtual_devices.simulators.modbus.tcp --config factory_plc.json
```

### Running Scenarios

```bash
# Factory floor scenario
python -m virtual_devices.scenarios.factory_floor

# Process control scenario
python -m virtual_devices.scenarios.process_control --scale 10

# Custom scenario from config
python -m virtual_devices.scenarios.custom --config my_scenario.yaml
```

### Running Benchmarks

```bash
# Throughput benchmark
python -m virtual_devices.benchmarks.throughput --protocol modbus --connections 100

# Latency benchmark
python -m virtual_devices.benchmarks.latency --protocol opcua --iterations 1000

# Stress test
python -m virtual_devices.benchmarks.stress --duration 3600 --ramp-up 300
```

### Integration with pytest

```python
import pytest
from virtual_devices.fixtures import modbus_simulator, opcua_server

@pytest.fixture
async def factory_setup():
    scenario = FactoryFloorScenario()
    await scenario.setup()
    yield scenario
    await scenario.teardown()

async def test_data_collection(factory_setup):
    # Test against running factory scenario
    pass
```

## Development Guidelines

### Creating New Simulators

1. **Inherit from base classes**:
   ```python
   from virtual_devices.base import BaseSimulator
   
   class MyProtocolSimulator(BaseSimulator):
       async def start(self): ...
       async def stop(self): ...
   ```

2. **Implement realistic behavior**:
   - Proper timing characteristics
   - Error conditions and recovery
   - State management
   - Resource cleanup

3. **Add configuration support**:
   - JSON/YAML configuration files
   - Environment variable support
   - Command-line arguments
   - Validation with Pydantic

4. **Include comprehensive tests**:
   - Unit tests for core logic
   - Integration tests with real clients
   - Performance benchmarks
   - Error condition testing

### Creating New Scenarios

1. **Define scenario components**:
   - Required devices and protocols
   - Network topology
   - Data patterns and timing
   - Interaction workflows

2. **Implement setup/teardown**:
   - Async context manager support
   - Resource management
   - Error handling
   - Cleanup guarantees

3. **Add monitoring capabilities**:
   - Metrics collection
   - Performance monitoring
   - Health checking
   - Logging and diagnostics

## Security Considerations

### Certificate Management

- Test certificates for OPC UA security
- CA certificate chains
- Certificate rotation testing
- Invalid certificate scenarios

### Authentication Testing

- Username/password authentication
- Certificate-based authentication
- Anonymous access scenarios
- Failed authentication handling

### Network Security

- TLS/SSL encryption testing
- Firewall simulation
- Network segmentation
- Security policy validation

## Monitoring and Diagnostics

### Metrics Collection

- Connection counts and states
- Message rates and latencies
- Error rates and types
- Resource utilization

### Logging

- Structured logging with context
- Configurable log levels
- Performance metrics
- Security events

### Health Checks

- Device availability monitoring
- Protocol connectivity checks
- Resource usage alerts
- Performance degradation detection

## Future Enhancements

### Planned Features

- [ ] Cloud-based device simulation
- [ ] Machine learning-based behavior modeling
- [ ] Visual scenario builder
- [ ] Real-time performance dashboards
- [ ] Integration with CI/CD pipelines

### Protocol Extensions

- [ ] DNP3 simulator
- [ ] IEC 61850 simulator
- [ ] BACnet simulator
- [ ] MQTT device simulation
- [ ] Custom protocol framework

### Advanced Scenarios

- [ ] Cybersecurity attack simulation
- [ ] Disaster recovery scenarios
- [ ] Maintenance workflow testing
- [ ] Regulatory compliance validation
- [ ] Performance optimization scenarios