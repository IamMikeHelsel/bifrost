# Virtual Device Research for Industrial Protocol Testing

## Overview

Research and evaluation of virtual device simulators for containerized testing of industrial protocols.

## Modbus Simulators

### 1. pyModbusTCP Simulator

**Repository**: https://github.com/sourceperl/pyModbusTCP
**License**: MIT
**Language**: Python

**Pros**:

- Pure Python, easy to containerize
- Supports both client and server
- Well-maintained with good documentation
- Lightweight and fast startup
- Easy to configure register maps

**Cons**:

- TCP only (no RTU simulation)
- Basic simulation features

**Docker Strategy**:

```python
# Simple Modbus TCP server
from pyModbusTCP.server import ModbusServer
server = ModbusServer(host="0.0.0.0", port=502, no_block=True)
server.start()
```

### 2. pymodbus Simulator

**Repository**: https://github.com/pymodbus-dev/pymodbus
**License**: BSD-3-Clause\
**Language**: Python

**Pros**:

- Comprehensive Modbus implementation
- Supports TCP, RTU, ASCII
- Advanced simulation capabilities
- Can simulate device failures and delays
- Built-in register data models

**Cons**:

- Heavier than pyModbusTCP
- More complex setup

**Docker Strategy**:

```python
from pymodbus.server.sync import StartTcpServer
from pymodbus.datastore import ModbusSlaveContext, ModbusServerContext
# Configurable device simulation
```

### 3. ModbusPal

**Repository**: https://github.com/zeha/modbuspal
**License**: GPL
**Language**: Java

**Pros**:

- GUI-based configuration
- Very realistic device simulation
- Support for multiple slaves
- Complex automation scripts

**Cons**:

- Java dependency
- GUI-oriented (harder to automate)
- GPL license may be restrictive

**Docker Strategy**: Possible but complex due to GUI requirements

### 4. diagslave (Evaluation Version)

**Website**: https://www.modbusdriver.com/diagslave.html
**License**: Commercial (free evaluation)
**Language**: C

**Pros**:

- Very lightweight and fast
- Extremely realistic behavior
- Supports all Modbus variants
- Command-line configurable

**Cons**:

- Commercial license for production
- Binary distribution only
- Limited free version

## OPC UA Simulators

### 1. open62541 Examples

**Repository**: https://github.com/open62541/open62541
**License**: Mozilla Public License 2.0
**Language**: C

**Pros**:

- Industry-standard implementation
- Multiple example servers
- High performance
- Well-documented

**Cons**:

- C compilation required
- More complex setup

### 2. opcua-asyncio Server

**Repository**: https://github.com/FreeOpcUa/opcua-asyncio
**License**: LGPL
**Language**: Python

**Pros**:

- Pure Python, easy containerization
- Async implementation
- Good simulation capabilities

**Cons**:

- LGPL license
- Less mature than open62541

### 3. Prosys OPC UA Simulation Server

**Website**: https://www.prosysopc.com/products/opc-ua-simulation-server/
**License**: Commercial (free version available)

**Pros**:

- Very realistic simulation
- Professional quality
- Comprehensive node tree

**Cons**:

- Commercial license
- Java-based
- Limited free version

## Error Handling Strategy

### Core Principles for Industrial Reliability

1. **Never Crash on External Failures**

   - Device disconnections
   - Network timeouts
   - Malformed protocol responses
   - Resource exhaustion

1. **Graceful Degradation**

   - Continue operating with reduced functionality
   - Cache last known values
   - Provide health status indicators

1. **Automatic Recovery**

   - Connection retry with exponential backoff
   - Circuit breaker patterns
   - Health monitoring and reconnection

1. **Comprehensive Logging**

   - Structured logging for diagnostics
   - Performance metrics
   - Error categorization

## EtherCAT Simulators

### 1. ethercrab Test Slave

**Repository**: https://github.com/ethercrab-rs/ethercrab
**License**: Apache 2.0 / MIT
**Language**: Rust

**Pros**:
- Same library ecosystem as planned master implementation
- Memory safe Rust implementation
- No-std support for embedded simulation
- Modern async/await patterns

**Cons**:
- Limited device profiles compared to commercial solutions
- Relatively new project (2023+)
- May lack advanced EtherCAT features

### 2. TwinCAT Virtual Devices (Beckhoff)

**Platform**: TwinCAT 3 Engineering
**License**: Commercial (free development license)
**Language**: Proprietary

**Pros**:
- Professional EtherCAT simulation
- Complete device database
- Industry-standard implementation
- Real-time simulation capabilities

**Cons**:
- Windows-only
- Commercial licensing for production
- Requires TwinCAT installation
- Not suitable for CI/CD pipelines

### 3. SOEM Test Slaves

**Repository**: https://github.com/OpenEtherCATsociety/SOEM
**License**: GPL v2 (restrictive)
**Language**: C

**Pros**:
- Proven implementation
- Basic slave simulation examples
- Cross-platform support

**Cons**:
- GPL licensing restrictions
- Limited to basic testing
- Requires compilation and setup

### 4. Custom EtherCAT Slave Simulator

**Approach**: Implement using ethercrab slave capabilities
**License**: MIT (matching project)
**Language**: Rust with Go FFI

**Pros**:
- Complete control over simulation features
- Permissive licensing
- Integration with existing test framework
- Custom device profiles for specific testing

**Cons**:
- Significant development effort
- Need to implement device profiles from scratch
- Testing coverage compared to commercial solutions

## Recommended Initial Selection

### For Immediate Implementation:

1. **pyModbusTCP** - Primary Modbus TCP simulator
1. **pymodbus** - Advanced Modbus testing with failure simulation
1. **opcua-asyncio** - OPC UA testing (despite LGPL, acceptable for testing)
1. **ethercrab test slave** - EtherCAT slave simulation for basic testing

### Error Handling Implementation Plan:

1. Connection management with retry logic
1. Protocol-level error recovery
1. Device health monitoring
1. Graceful timeout handling
1. Resource cleanup and memory management

## Next Steps

1. Create Docker containers for selected simulators
1. Implement error handling patterns in Rust Modbus codec
1. Set up comprehensive test scenarios
1. Document failure modes and recovery strategies
1. **NEW**: Develop EtherCAT slave simulation framework using ethercrab
