# EtherCAT Simulators

## Purpose
EtherCAT slave device simulators for motion control and high-performance automation testing.

## Planned Contents
- **EtherCAT Slave Simulator**: General-purpose slave device simulation
- **Servo Drive Simulator**: Motion control servo drive profiles
- **Stepper Motor Controller**: Stepper motor device simulation
- **I/O Modules**: Digital and analog I/O device profiles
- **Safety Devices**: Functional safety device simulation

## Key Features
- Real-time EtherCAT slave implementation using ethercrab
- Configurable device profiles (CoE, FoE, SoE)
- Distributed clock synchronization simulation
- Process data and mailbox communication
- Device state machine (Init, PreOp, SafeOp, Op)
- ESI (EtherCAT Slave Information) file support

## Performance Targets
- 1-10ms cycle times with configurable jitter
- Support 100+ simulated slave devices
- Real-time I/O data exchange
- Distributed clock accuracy within 1µs

## Testing Focus
- Real-time cycle time validation
- Distributed clock synchronization
- Device commissioning and configuration
- Motion control parameter access
- Network topology variations (line, star, tree)
- Error injection and recovery testing

## Implementation Plan

### Phase 1: Basic EtherCAT Slave
- Simple I/O device simulation
- Basic process data exchange
- Device discovery and identification

### Phase 2: Motion Control Devices
- Servo drive profile implementation
- Position, velocity, and torque control modes
- Stepper motor controller simulation

### Phase 3: Advanced Features
- Distributed clock synchronization
- Safety over EtherCAT (FSoE) simulation
- Complex device profiles and commissioning

### Phase 4: Network Simulation
- Multi-slave network topology
- Timing analysis and performance validation
- Fault injection and error handling

## Technical Architecture

### Rust Implementation
```rust
// EtherCAT slave simulator using ethercrab
use ethercrab::SlaveDevice;

pub struct EtherCATSlave {
    pub address: u16,
    pub vendor_id: u32,
    pub product_code: u32,
    pub process_data: ProcessDataMapping,
    pub mailbox: MailboxHandler,
    pub state: SlaveState,
}

impl EtherCATSlave {
    pub fn new(config: SlaveConfig) -> Self;
    pub fn process_cycle(&mut self, master_data: &[u8]) -> Vec<u8>;
    pub fn handle_mailbox(&mut self, request: MailboxRequest) -> MailboxResponse;
}
```

### Device Profiles
- **Generic I/O**: Digital inputs/outputs, analog sensors
- **Servo Drive**: Position control, velocity control, torque control
- **Stepper Motor**: Step/direction control, microstepping
- **Safety Device**: Emergency stop, safety inputs/outputs

### Configuration
YAML-based configuration for device simulation:
```yaml
ethercat_slave:
  address: 1001
  vendor_id: 0x00000002  # Beckhoff
  product_code: 0x044c2c52  # EL7041 servo terminal
  device_profile: "servo_drive"
  process_data:
    inputs:
      - name: "position_feedback"
        size: 4  # bytes
        type: "int32"
    outputs:
      - name: "target_position"
        size: 4
        type: "int32"
  
  motion_control:
    max_velocity: 3000  # rpm
    max_acceleration: 10000  # rpm/s
    position_range: [-2147483648, 2147483647]
```

## Testing Scenarios

### Basic Communication
- Device discovery and enumeration
- Process data exchange validation
- Mailbox communication testing

### Motion Control
- Position control accuracy
- Velocity profiling
- Acceleration/deceleration curves
- Following error monitoring

### Real-time Performance
- Cycle time consistency
- Distributed clock accuracy
- Jitter measurement and analysis
- Overrun detection and handling

### Error Conditions
- Communication timeouts
- Device state transitions
- Emergency stop scenarios
- Network topology changes

## Integration with Bifrost

The EtherCAT simulators will integrate with the Bifrost testing framework:

1. **Docker Containers**: Containerized simulators for CI/CD
2. **Test Scenarios**: Automated testing with various device configurations
3. **Performance Benchmarks**: Real-time performance validation
4. **Error Injection**: Fault tolerance testing

## Development Roadmap

- **Q2 2026**: Basic EtherCAT slave simulator
- **Q3 2026**: Motion control device profiles
- **Q4 2026**: Advanced features and network simulation
- **Q1 2027**: Production-ready testing framework