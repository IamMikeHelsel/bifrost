# Modbus Simulators

## Purpose
Full Modbus TCP/RTU simulators for comprehensive protocol testing.

## Planned Contents
- **TCP Simulator**: Modbus TCP server with configurable register maps
- **RTU Simulator**: Serial Modbus RTU over TCP for testing
- **Multi-slave Simulator**: Single simulator hosting multiple slave devices
- **Fault Injection**: Simulate communication errors and device faults

## Key Features
- All standard function codes (1-23)
- Configurable data maps and register types
- Realistic timing behavior
- Exception code simulation
- Connection state management

## Performance Targets
- 1000+ registers/second per connection
- 100+ concurrent connections
- < 1ms response latency
- Configurable scan rates

## Testing Focus
- Protocol compliance validation
- Performance under load
- Error handling and recovery
- Multi-device communication patterns