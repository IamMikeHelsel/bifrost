# Network Condition Simulators

## Purpose
Simulate realistic industrial network conditions for resilience testing.

## Contents
- Network latency injection
- Packet loss simulation
- Bandwidth limiting
- Connection failure scenarios

## Capabilities
- **Latency**: 1ms - 1000ms injection
- **Packet Loss**: 0.1% - 10% simulation
- **Bandwidth**: 56K - 1Gbps limiting
- **Disconnection**: Planned and random failures

## Usage
- Test protocol resilience under poor network conditions
- Validate retry and recovery mechanisms
- Performance testing under constrained networks
- Simulate real-world industrial environments

## Integration
- Transparent proxy for existing tests
- Docker network simulation
- Linux traffic control (tc) integration
- Configurable via YAML/JSON