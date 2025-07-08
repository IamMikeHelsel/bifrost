# 🔧 Real Hardware Testing Framework

> **Comprehensive automated testing framework for real industrial hardware devices**

## 🎯 Overview

The Bifrost Real Hardware Testing Framework provides automated testing capabilities for physical industrial devices including PLCs, HMIs, gateways, and other fieldbus equipment. It complements the virtual device testing framework by enabling validation against actual hardware in controlled test lab environments.

## ✨ Key Features

- **📋 Device Registry**: Comprehensive tracking of hardware devices with firmware versions, network configurations, and test schedules
- **🧪 Test Execution**: Automated execution of test scenarios with detailed step-by-step results
- **📅 Scheduling**: Automated test scheduling with configurable frequencies (daily, weekly, monthly)
- **🔄 Multi-Protocol Support**: Works with Modbus TCP, EtherNet/IP, OPC UA, and Siemens S7 protocols
- **📊 Result Analysis**: Detailed test metrics, performance data, and historical tracking
- **🖥️ CLI Interface**: Comprehensive command-line tool for all framework operations
- **🏗️ Extensible Architecture**: Easy integration with existing protocol handlers and testing infrastructure

## 🚀 Quick Start

### 1. Build the Framework

```bash
cd go-gateway
make hardware-test-build
```

### 2. Configure Your Test Lab

Edit `configs/hardware_test_lab.yaml` to define your hardware devices:

```yaml
devices:
  - device_id: "my_plc_001"
    manufacturer: "Allen-Bradley"
    model: "CompactLogix 1769-L33ER"
    firmware: "33.011"
    protocols: ["ethernet_ip", "modbus_tcp"]
    network:
      ip: "192.168.100.10"
      port: 44818
    test_schedule:
      frequency: "weekly"
      scenarios: ["basic_io", "performance"]
      enabled: true
```

### 3. Define Test Scenarios

Edit `configs/test_scenarios.yaml` to create your test procedures:

```yaml
scenarios:
  - name: "basic_io"
    description: "Basic I/O connectivity test"
    protocol: "modbus_tcp"
    timeout: "2m"
    steps:
      - name: "Connect to device"
        type: "connect"
        timeout: "30s"
      - name: "Read test register"
        type: "read"
        address: "40001"
        timeout: "5s"
```

### 4. Run Tests

```bash
# List all registered devices
make hardware-test-run

# Check framework status
make hardware-test-status

# Run specific test
./bin/hardware_test -cmd test -device my_plc_001 -scenario basic_io

# Start automated scheduler
make hardware-test-daemon
```

## 📁 Framework Architecture

```
internal/hardware/
├── registry.go      # Device registry and configuration management
├── executor.go      # Test execution engine
├── scheduler.go     # Automated test scheduling
├── manager.go       # Main framework coordinator
└── hardware_test.go # Comprehensive test suite

cmd/hardware_test/
└── main.go          # CLI application

configs/
├── hardware_test_lab.yaml  # Lab and device configuration
└── test_scenarios.yaml     # Test scenario definitions

docs/
└── HARDWARE_TESTING_FRAMEWORK.md  # Detailed documentation
```

## 🔧 Configuration

### Device Configuration

Each device in your test lab is configured with:

- **Identification**: Unique ID, manufacturer, model, firmware version
- **Network**: IP address, port, subnet, VLAN information
- **Protocols**: Supported communication protocols
- **Test Schedule**: Frequency, scenarios, priority, enabled status
- **Metadata**: Custom fields for location, maintenance dates, etc.

### Test Scenarios

Test scenarios define the actual test procedures:

- **Test Steps**: Connect, disconnect, read, write, ping, diagnostics
- **Timeouts**: Per-step and overall scenario timeouts
- **Retry Logic**: Configurable retry attempts with delays
- **Expected Values**: Validation of read operations
- **Performance Metrics**: Timing and throughput measurements

## 🧪 Test Categories

### 🔌 Functional Testing (`basic_io`)
- Connection establishment and teardown
- Basic read/write operations
- Device information retrieval
- Protocol compliance validation

### ⚡ Performance Testing (`performance`)
- Throughput measurement
- Latency testing
- Rapid operation cycles
- Multi-register batch operations

### 💪 Stress Testing (`stress`)
- Connection limits testing
- Error recovery validation
- High-frequency operations
- Resource exhaustion scenarios

### 🔄 Compatibility Testing (`compatibility`)
- Vendor-specific feature validation
- Protocol conformance testing
- Cross-version compatibility
- Feature matrix verification

### 🌐 Interoperability Testing (`interoperability`)
- Multi-vendor device scenarios
- Cross-protocol communication
- System integration validation
- End-to-end workflow testing

## 📊 Test Execution

### Available Test Step Types

| Step Type | Description | Parameters |
|-----------|-------------|------------|
| `connect` | Establish device connection | timeout, retry_count |
| `disconnect` | Close device connection | timeout |
| `ping` | Test basic connectivity | timeout, retry_count |
| `read` | Read data from device | address, expected, timeout |
| `write` | Write data to device | address, value, timeout |
| `device_info` | Get device information | timeout |
| `diagnostics` | Retrieve device health | timeout |

### Example Test Execution

```bash
🧪 Running Test: basic_io on ab_compactlogix_001
================================

📱 Device: Allen-Bradley CompactLogix 1769-L33ER (192.168.100.10)
🔧 Test Scenario: basic_io

🚀 Test execution started (ID: hw-test-1751984300123456789)
⏳ Waiting for test to complete...

✅ Test completed successfully!

📊 Test Step Results:
=====================
✅ Step 1: Connect to device (0.25s)
✅ Step 2: Ping device (0.05s)
✅ Step 3: Read holding register 1 (0.08s)
✅ Step 4: Write test value (0.12s)
✅ Step 5: Read back written value (0.07s)
✅ Step 6: Disconnect from device (0.03s)
```

## 🎛️ CLI Commands

### Device Management
```bash
# List all registered devices
./hardware_test -cmd list-devices

# Show device details and status
./hardware_test -cmd status
```

### Test Execution
```bash
# Run a specific test scenario
./hardware_test -cmd test -device DEVICE_ID -scenario SCENARIO_NAME

# Example: Test Allen-Bradley PLC with basic I/O
./hardware_test -cmd test -device ab_compactlogix_001 -scenario basic_io
```

### Scheduling and Automation
```bash
# View test schedule
./hardware_test -cmd schedule

# Run as daemon with automated scheduling
./hardware_test -cmd run -daemon

# Just show status (default)
./hardware_test -cmd run
```

### Advanced Options
```bash
# Use custom configuration files
./hardware_test -config /path/to/lab.yaml -scenarios /path/to/scenarios.yaml

# Enable verbose logging
./hardware_test -v -cmd test -device DEVICE_ID -scenario SCENARIO_NAME
```

## 🔄 Integration

### Protocol Handler Integration

The framework integrates seamlessly with existing protocol handlers:

```go
// Register protocol handlers
manager.RegisterProtocolHandler("modbus_tcp", protocols.NewModbusHandler(logger))
manager.RegisterProtocolHandler("ethernet_ip", protocols.NewEtherNetIPHandler(logger))
manager.RegisterProtocolHandler("opcua", protocols.NewOPCUAHandler(logger))
```

### Result Integration

Test results are automatically:
- Stored in device registry with historical tracking
- Available for analysis and reporting
- Integrated with release card generation
- Compatible with CI/CD pipeline integration

### Existing Framework Compatibility

The hardware testing framework builds on:
- **Virtual Device Testing**: Complements virtual testing with real hardware validation
- **Protocol Handlers**: Reuses existing protocol implementations
- **Device Abstractions**: Extends the existing device model
- **Testing Patterns**: Follows established testing conventions

## 🏗️ Development

### Running Tests

```bash
# Run hardware framework tests
go test ./internal/hardware/... -v

# Run all tests
make test
```

### Building

```bash
# Build CLI tool
make hardware-test-build

# Build all platform binaries
make build-all
```

### Adding New Test Scenarios

1. Edit `configs/test_scenarios.yaml`
2. Define new scenario with appropriate steps
3. Test the scenario with a device
4. Document any special requirements

### Adding New Devices

1. Edit `configs/hardware_test_lab.yaml`
2. Add device configuration with network settings
3. Configure test schedule and scenarios
4. Verify connectivity and test execution

## 📈 Monitoring and Analysis

### Test Results
- Success/failure rates per device and scenario
- Performance metrics and trends
- Error analysis and troubleshooting
- Historical comparison and baseline tracking

### Device Health
- Connection status monitoring
- Error rate tracking
- Performance degradation detection
- Maintenance scheduling integration

### Reporting
- JSON, YAML, and CSV export formats
- Webhook notifications for test results
- Integration with external monitoring systems
- Custom reporting and dashboard integration

## 🔐 Security and Safety

### Network Security
- Isolated test lab networks with VLAN segmentation
- Secure communication protocols
- Access control and authentication
- Audit logging and monitoring

### Device Safety
- Controlled test procedures with safety checks
- Emergency stop functionality
- Read-only testing modes for production devices
- Rollback capabilities for configuration changes

### Lab Management
- Device reservation system
- Concurrent test execution limits
- Priority-based scheduling
- Resource allocation and conflict resolution

## 🚀 Future Enhancements

### Planned Features
- **Web Dashboard**: Real-time monitoring and control interface
- **Advanced Analytics**: Machine learning for anomaly detection
- **Mobile Support**: Mobile device management and monitoring
- **Cloud Integration**: Remote test lab access and management
- **CMMS Integration**: Maintenance management system connectivity

### Scalability Improvements
- **Multi-Lab Support**: Distributed test lab management
- **Load Balancing**: Intelligent test distribution
- **High Availability**: Redundant test execution capabilities
- **Performance Optimization**: Enhanced concurrent test handling

## 🤝 Contributing

When contributing to the hardware testing framework:

1. **Follow Patterns**: Use existing code patterns and interfaces
2. **Add Tests**: Include comprehensive tests for new functionality
3. **Document Changes**: Update documentation for new features
4. **Maintain Compatibility**: Ensure backward compatibility
5. **Real Hardware Testing**: Test with actual hardware when possible

## 📄 License

This hardware testing framework is part of the Bifrost project and follows the same licensing terms.

---

**🌉 Ready to validate your industrial hardware with confidence!**

For detailed implementation information, see [HARDWARE_TESTING_FRAMEWORK.md](docs/HARDWARE_TESTING_FRAMEWORK.md).