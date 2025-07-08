# Hardware Testing Framework

## Overview

The Bifrost Hardware Testing Framework provides comprehensive automated testing capabilities for real industrial hardware devices. It complements the virtual device testing framework by enabling validation against actual PLCs, HMIs, and other industrial equipment.

## Architecture

The framework consists of several key components:

### Core Components

1. **Device Registry** (`registry.go`)
   - Manages hardware device inventory
   - Tracks device configurations, firmware versions, and test schedules
   - Stores test results and device status

2. **Test Executor** (`executor.go`)
   - Executes test scenarios against hardware devices
   - Manages concurrent test execution
   - Provides detailed step-by-step test results

3. **Test Scheduler** (`scheduler.go`)
   - Automated test scheduling based on device configuration
   - Priority-based task queue management
   - Configurable test frequencies (daily, weekly, monthly)

4. **Hardware Test Manager** (`manager.go`)
   - Coordinates all framework components
   - Provides unified interface for test management
   - Handles configuration loading and protocol handler registration

### CLI Tool

The `hardware_test` command-line tool provides easy access to all framework functionality:

```bash
# List all registered devices
./hardware_test -cmd list-devices

# Show current status
./hardware_test -cmd status

# Run a specific test
./hardware_test -cmd test -device ab_compactlogix_001 -scenario basic_io

# Show test schedule
./hardware_test -cmd schedule

# Run as daemon with scheduler
./hardware_test -cmd run -daemon
```

## Configuration

### Hardware Test Lab Configuration

The main configuration file (`configs/hardware_test_lab.yaml`) defines:

- **Lab Configuration**: Network settings, scheduling parameters, reporting options
- **Device Registry**: Physical devices with their network configuration and test schedules

Example device configuration:
```yaml
devices:
  - device_id: "ab_compactlogix_001"
    manufacturer: "Allen-Bradley"
    model: "CompactLogix 1769-L33ER"
    firmware: "33.011"
    protocols:
      - "ethernet_ip"
      - "modbus_tcp"
    network:
      ip: "192.168.100.10"
      port: 44818
      subnet: "test_lab_vlan_1"
    test_schedule:
      frequency: "weekly"
      scenarios:
        - "basic_io"
        - "performance"
        - "stress"
      enabled: true
      priority: 1
```

### Test Scenarios Configuration

Test scenarios (`configs/test_scenarios.yaml`) define the actual test procedures:

```yaml
scenarios:
  - name: "basic_io"
    description: "Basic I/O connectivity and read/write operations"
    protocol: "modbus_tcp"
    timeout: "2m"
    steps:
      - name: "Connect to device"
        type: "connect"
        timeout: "30s"
        retry_count: 3
      - name: "Read holding register 1"
        type: "read"
        address: "40001"
        timeout: "5s"
      - name: "Write test value"
        type: "write"
        address: "40010"
        value: 12345
        timeout: "5s"
```

## Test Categories

The framework supports various test categories:

### 1. Functional Testing (`basic_io`)
- Basic protocol operations
- Read/write operations
- Device connectivity validation

### 2. Performance Testing (`performance`)
- Throughput measurement
- Latency testing
- Rapid operation cycles

### 3. Stress Testing (`stress`)
- Connection limits
- Error recovery
- High-frequency operations

### 4. Compatibility Testing (`compatibility`)
- Vendor-specific behavior
- Protocol conformance
- Feature validation

### 5. Interoperability Testing (`interoperability`)
- Multi-vendor scenarios
- Cross-protocol validation
- System integration testing

## Test Step Types

The framework supports various test step types:

- **connect**: Establish connection to device
- **disconnect**: Close device connection
- **ping**: Test basic connectivity
- **read**: Read data from device address
- **write**: Write data to device address
- **device_info**: Retrieve device information
- **diagnostics**: Get device health status

## Integration with Existing Framework

The hardware testing framework integrates seamlessly with the existing Bifrost infrastructure:

### Protocol Handlers
Reuses existing protocol implementations:
- Modbus TCP/RTU
- EtherNet/IP
- OPC UA
- Siemens S7

### Device Management
Builds on the existing device abstraction layer with extensions for:
- Hardware-specific metadata
- Test scheduling configuration
- Result storage and tracking

### Result Integration
Test results can be integrated with:
- Release card generation
- Performance benchmarking
- CI/CD pipelines
- Quality assurance processes

## Usage Examples

### Basic Device Testing

```bash
# Test a specific device with basic I/O scenario
./hardware_test -cmd test -device ab_compactlogix_001 -scenario basic_io
```

### Automated Scheduling

```bash
# Run the framework as a daemon with automated scheduling
./hardware_test -cmd run -daemon

# This will:
# - Load device configurations
# - Schedule tests based on device settings
# - Execute tests automatically
# - Store results for analysis
```

### Device Management

```bash
# List all registered devices
./hardware_test -cmd list-devices

# Check current status
./hardware_test -cmd status

# View test schedule
./hardware_test -cmd schedule
```

## Development and Testing

### Running Tests

```bash
cd go-gateway
go test ./internal/hardware/... -v
```

### Building the CLI Tool

```bash
cd go-gateway
go build -o bin/hardware_test ./cmd/hardware_test/
```

### Adding New Test Scenarios

1. Edit `configs/test_scenarios.yaml`
2. Add scenario configuration with appropriate steps
3. Restart the hardware test manager to load new scenarios

### Adding New Devices

1. Edit `configs/hardware_test_lab.yaml`
2. Add device configuration with network and schedule settings
3. Reload configuration or restart the framework

## Security and Safety

### Network Security
- Isolated test lab networks
- VLAN segmentation
- Secure communication protocols

### Device Safety
- Controlled test procedures
- Rollback capabilities
- Emergency stop functionality
- Read-only testing modes

### Access Control
- Device reservation system
- Test execution permissions
- Audit logging

## Monitoring and Diagnostics

### Test Result Analysis
- Success/failure rates
- Performance metrics
- Trend analysis
- Historical comparisons

### Device Health Monitoring
- Connection status tracking
- Error rate monitoring
- Performance degradation detection
- Maintenance scheduling

### Alert and Notification
- Test failure notifications
- Device offline alerts
- Performance threshold violations
- Scheduled maintenance reminders

## Future Enhancements

### Planned Features
- Web-based dashboard
- Real-time test monitoring
- Advanced scheduling algorithms
- Machine learning for anomaly detection
- Integration with CMMS systems
- Mobile device support

### Scalability
- Multi-lab support
- Cloud integration
- Remote device access
- Distributed test execution

## Troubleshooting

### Common Issues

1. **Device Connection Failures**
   - Check network connectivity
   - Verify device configuration
   - Review firewall settings

2. **Test Execution Errors**
   - Validate test scenarios
   - Check protocol handler registration
   - Review device availability

3. **Configuration Issues**
   - Validate YAML syntax
   - Check file permissions
   - Verify path configuration

### Logging and Debugging

Enable verbose logging for debugging:
```bash
./hardware_test -v -cmd test -device device_id -scenario scenario_name
```

## Contributing

When contributing to the hardware testing framework:

1. Follow existing code patterns and interfaces
2. Add comprehensive tests for new functionality
3. Update documentation for new features
4. Ensure backward compatibility
5. Test with real hardware when possible

## License

This hardware testing framework is part of the Bifrost project and follows the same licensing terms.