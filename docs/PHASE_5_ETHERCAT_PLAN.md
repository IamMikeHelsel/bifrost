# Phase 5: EtherCAT Integration Plan

This document outlines the design and implementation plan for adding EtherCAT support to Bifrost, following the established patterns from the roadmap and existing protocol implementations.

## 1. Goals

- High-performance EtherCAT master implementation based on ethercrab (Rust)
- Safe Rust wrapper integration with Go gateway via FFI
- Async Go API consistent with existing ProtocolHandler interface
- Real-time capable I/O operations for motion control applications
- Meet performance targets defined for industrial automation

## 2. Core Components

### 2.1. Rust Wrapper (`ethercat-wrapper`)

A new Rust crate will be created at `packages/bifrost/native/src/ethercat/` to wrap the ethercrab library.

**Key responsibilities:**
- Provide safe abstractions over ethercrab async operations
- Manage EtherCAT master lifecycle and device connections
- Handle data type conversions between Rust and Go types
- Implement error handling, converting Rust Results to C-compatible error codes
- Expose a C-compatible API for Go integration via CGO

### 2.2. Build System Integration

- The `ethercrab` library will be added as a dependency to the native Rust crate
- Build.rs script will compile the Rust EtherCAT wrapper to a static library
- CGO bindings will link against the compiled Rust library
- Cross-compilation support for Linux (primary) and Windows

### 2.3. Go Protocol Handler (`ethercat.go`)

A new Go protocol handler will be implemented in `go-gateway/internal/protocols/` following the established ProtocolHandler interface.

**Key responsibilities:**
- Implement all ProtocolHandler interface methods for EtherCAT
- Manage CGO calls to the Rust wrapper library
- Provide EtherCAT-specific configuration and diagnostics
- Handle device discovery and process data operations
- Implement real-time I/O operations with timing guarantees

### 2.4. Virtual Device Simulator

EtherCAT virtual devices for testing will be created in `virtual-devices/simulators/ethercat/`.

**Components:**
- EtherCAT slave simulator with configurable I/O
- Motion control device profiles (servo drives, stepper motors)
- Distributed clock simulation for timing testing
- Network topology simulation (line, star, tree configurations)

## 3. Implementation Steps

### 3.1. Setup Build System (Weeks 1-2)

1. **Create Rust Module:**
   - Create `packages/bifrost/native/src/ethercat/mod.rs`
   - Add ethercrab dependency to `Cargo.toml`
   - Configure cross-compilation for target platforms

2. **Build Integration:**
   - Update `build.rs` to compile EtherCAT wrapper
   - Configure static library generation
   - Set up CGO build flags and linker options

3. **FFI Interface:**
   - Define C-compatible API for Go integration
   - Implement error handling and memory management
   - Create header files for CGO bindings

### 3.2. Develop Rust Wrapper (Weeks 3-6)

1. **Master Implementation:**
   - Wrap ethercrab Master initialization and configuration
   - Implement device scanning and topology detection
   - Handle distributed clocks synchronization

2. **Device Management:**
   - Implement slave device discovery and configuration
   - Support ESI (EtherCAT Slave Information) file parsing
   - Manage device state machine (Init, PreOp, SafeOp, Op)

3. **Process Data Operations:**
   - Implement cyclic I/O operations
   - Support distributed clocks for synchronized operation
   - Handle emergency and mailbox communication

4. **Error Handling:**
   - Comprehensive error categorization and recovery
   - Network diagnostics and health monitoring
   - Real-time performance monitoring

### 3.3. Go Protocol Handler (Weeks 7-10)

1. **ProtocolHandler Implementation:**
   ```go
   type EtherCATHandler struct {
       master     *C.ethercat_master_t
       devices    map[string]*EtherCATDevice
       config     *EtherCATConfig
       logger     *slog.Logger
       metrics    *prometheus.Registry
   }
   
   func (e *EtherCATHandler) Connect(device *Device) error
   func (e *EtherCATHandler) ReadTag(device *Device, tag *Tag) (interface{}, error)
   func (e *EtherCATHandler) WriteTag(device *Device, tag *Tag, value interface{}) error
   // ... other ProtocolHandler methods
   ```

2. **EtherCAT-Specific Features:**
   - Real-time process data operations
   - Distributed clock synchronization
   - Device configuration and commissioning
   - Motion control parameter access

3. **Performance Optimization:**
   - Connection pooling for multiple devices
   - Batch operations for process data
   - Memory-mapped I/O where possible
   - Goroutine scheduling optimization

### 3.4. Testing Framework (Weeks 11-12)

1. **Unit Tests:**
   - Test all ProtocolHandler interface methods
   - Mock EtherCAT master for isolated testing
   - Validate data type conversions and error handling

2. **Integration Tests:**
   - Test with virtual EtherCAT devices
   - Validate real-time performance requirements
   - Test network topology variations

3. **Performance Benchmarks:**
   - Cycle time measurement and jitter analysis
   - Throughput testing with multiple devices
   - Memory usage and CPU utilization profiling

### 3.5. Virtual Device Development (Weeks 13-14)

1. **EtherCAT Slave Simulator:**
   ```go
   type EtherCATSlave struct {
       Address      uint16
       ProductCode  uint32
       VendorID     uint32
       ProcessData  []byte
       Mailbox      MailboxHandler
   }
   ```

2. **Device Profiles:**
   - Generic I/O devices (digital/analog inputs/outputs)
   - Servo drive simulators with position/velocity control
   - Stepper motor controllers
   - Safety devices for functional safety testing

3. **Network Simulation:**
   - Configurable network topology
   - Timing simulation with distributed clocks
   - Error injection for robustness testing

## 4. Architecture Design

### 4.1. System Architecture

```
Go Application Layer
    ↓
EtherCAT Protocol Handler (Go)
    ↓
CGO Interface
    ↓
Rust EtherCAT Wrapper
    ↓
ethercrab Library (Rust)
    ↓
Raw Socket Interface
    ↓
Ethernet Driver
```

### 4.2. Data Flow

1. **Configuration Phase:**
   - Load ESI files and device configurations
   - Initialize EtherCAT master and scan network
   - Configure device state machines and process data mapping

2. **Operational Phase:**
   - Cyclic process data exchange (1-10ms cycles)
   - Acyclic mailbox communication for configuration
   - Distributed clock synchronization
   - Diagnostic monitoring and error handling

### 4.3. Threading Model

- **Master Thread**: Dedicated thread for EtherCAT master operations
- **Goroutine Pool**: Handle API requests with async operations
- **Timing Thread**: High-priority thread for real-time operations (if supported)

## 5. Configuration Schema

### 5.1. EtherCAT Master Configuration

```yaml
ethercat:
  master:
    interface: "eth0"              # Network interface
    cycle_time: 1000              # Microseconds (1ms)
    distributed_clocks: true       # Enable DC synchronization
    redundancy: false             # Hot standby master
  
  devices:
    - name: "servo_drive_1"
      position: 0                 # Auto-increment slave position
      vendor_id: 0x00000002      # Beckhoff
      product_code: 0x044c2c52   # EL7041 servo terminal
      config_file: "servo.xml"   # ESI configuration
      
  process_data:
    inputs:
      - name: "position_feedback"
        slave: "servo_drive_1"
        index: 0x6064
        subindex: 0x00
        type: "int32"
    
    outputs:
      - name: "target_position"
        slave: "servo_drive_1"  
        index: 0x607A
        subindex: 0x00
        type: "int32"
```

### 5.2. Device Addressing

EtherCAT devices use structured addressing:
- **Slave Position**: Physical position in network (0-65534)
- **CoE Objects**: CANopen object dictionary (index.subindex)
- **Process Data**: Direct memory mapping for real-time I/O
- **Logical Addressing**: Named tags for application use

## 6. Performance Targets

### 6.1. Real-time Performance
- **Cycle Time**: 1ms to 10ms configurable
- **Jitter**: < 1% of cycle time (< 10µs for 1ms cycle)
- **Latency**: < 100µs for urgent commands
- **Device Count**: Support 100+ EtherCAT slaves

### 6.2. Throughput Targets
- **Process Data**: 10,000+ I/O points per cycle
- **Mailbox Messages**: 1,000+ acyclic operations per second
- **Configuration Speed**: Device commissioning < 1 second per slave

### 6.3. Resource Usage
- **Memory**: < 1MB per 100 devices
- **CPU**: < 10% for cyclic operations
- **Network**: Full Ethernet bandwidth utilization

## 7. Error Handling Strategy

### 7.1. Error Categories

1. **Network Errors:**
   - Cable disconnection / connection issues
   - Frame loss / corruption
   - Timing violations

2. **Device Errors:**
   - Slave not responding
   - Configuration mismatch
   - Emergency messages

3. **Master Errors:**
   - Resource exhaustion
   - Timing budget exceeded
   - Invalid configuration

### 7.2. Recovery Mechanisms

- **Automatic Retry**: For transient network issues
- **Device Restart**: Reinitialize non-responsive slaves
- **Graceful Degradation**: Continue with reduced functionality
- **Error Reporting**: Comprehensive diagnostics and logging

## 8. Security Considerations

### 8.1. Network Security
- **Raw Socket Access**: Requires elevated privileges
- **Network Isolation**: Dedicated EtherCAT network segment
- **Access Control**: Limit master functionality to authorized users

### 8.2. Safety Requirements
- **Functional Safety**: Support Safety over EtherCAT (FSoE)
- **Emergency Stops**: Immediate response to safety signals
- **Fail-Safe Operation**: Defined behavior on communication loss

## 9. Testing Strategy

### 9.1. Unit Testing
- Mock EtherCAT master for isolated component testing
- Validate all error conditions and recovery paths
- Test data type conversions and boundary conditions

### 9.2. Integration Testing
- Virtual device testing with complete EtherCAT stack
- Real hardware testing with common device types
- Performance validation under various load conditions

### 9.3. Compliance Testing
- EtherCAT Technology Group (ETG) conformance testing
- Interoperability testing with major device vendors
- Safety certification if functional safety features implemented

## 10. Documentation Requirements

### 10.1. User Documentation
- **Quick Start Guide**: Basic EtherCAT setup and configuration
- **Configuration Reference**: Complete parameter documentation
- **Device Integration**: How to add new EtherCAT devices
- **Troubleshooting Guide**: Common issues and solutions

### 10.2. Developer Documentation
- **API Reference**: Complete Go API documentation
- **FFI Interface**: Rust wrapper and CGO binding documentation
- **Performance Guide**: Optimization recommendations
- **Contributing Guide**: How to extend EtherCAT support

## 11. Deployment Considerations

### 11.1. System Requirements
- **Operating System**: Linux (primary), Windows (secondary)
- **Privileges**: Raw socket access (typically root/administrator)
- **Hardware**: Dedicated Ethernet interface for EtherCAT
- **Real-time**: RT kernel recommended for best performance

### 11.2. Distribution
- **Static Binary**: Include Rust library in Go binary
- **Container Support**: Docker images with proper capabilities
- **Package Managers**: Native packages for major distributions

## 12. Roadmap Integration

### 12.1. Bifrost Roadmap Updates

Add EtherCAT as Phase 5 in the development roadmap:

**Phase 5: EtherCAT Protocol Support (Q2-Q3 2026)**
- Native EtherCAT master implementation using ethercrab
- Real-time I/O operations for motion control
- Distributed clock synchronization
- Virtual device testing framework
- Production-ready deployment with performance monitoring

### 12.2. Success Metrics
- **Performance**: Meet industrial real-time requirements
- **Reliability**: 99.9% uptime in production environments
- **Compatibility**: Support major EtherCAT device vendors
- **Adoption**: Usage in at least 3 production industrial applications

## 13. Risk Mitigation

### 13.1. Technical Risks
- **Real-time Performance**: Go runtime limitations for hard real-time
  - *Mitigation*: Dedicated threads, performance profiling, fallback options
- **Rust Integration**: FFI complexity and memory management
  - *Mitigation*: Comprehensive testing, memory leak detection
- **Hardware Dependencies**: Raw socket access requirements
  - *Mitigation*: Clear documentation, containerized deployment options

### 13.2. Project Risks
- **Resource Requirements**: Complex implementation requiring specialized knowledge
  - *Mitigation*: Phased approach, external consulting if needed
- **Library Maturity**: ethercrab is relatively new
  - *Mitigation*: Active contribution to library, fallback to commercial options
- **Market Adoption**: Limited demand for Go-based EtherCAT solutions
  - *Mitigation*: Focus on unique value proposition, performance advantages

## 14. Conclusion

EtherCAT integration will significantly expand Bifrost's capabilities in industrial automation, particularly for motion control and high-performance applications. The proposed approach using ethercrab provides a good balance of permissive licensing, modern implementation, and achievable real-time performance.

The phased implementation plan allows for iterative development and early validation of critical performance requirements. Success depends on careful attention to real-time performance characteristics and comprehensive testing with both virtual and real EtherCAT devices.