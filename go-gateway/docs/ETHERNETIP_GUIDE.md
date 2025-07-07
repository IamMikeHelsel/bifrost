# EtherNet/IP Protocol Implementation Guide

## Overview

This guide provides comprehensive documentation for the EtherNet/IP protocol implementation in the Bifrost Go gateway. EtherNet/IP (Ethernet Industrial Protocol) is an industrial communication protocol that adapts the Common Industrial Protocol (CIP) to standard Ethernet networks.

## Features

### Core Capabilities
- **Full CIP Implementation**: Complete support for CIP over Ethernet
- **Explicit Messaging**: TCP-based request/response communication for configuration and diagnostics
- **Implicit Messaging**: UDP-based producer/consumer communication for real-time I/O (planned)
- **Session Management**: Automatic CIP session registration and management
- **Device Discovery**: Network scanning for EtherNet/IP devices
- **Allen-Bradley Support**: Optimized for ControlLogix, CompactLogix, and MicroLogix PLCs

### Advanced Features
- **Performance Optimization**: Connection pooling, tag caching, and batch operations
- **Comprehensive Diagnostics**: Detailed error tracking and performance monitoring
- **Health Checking**: Automated device health assessment
- **Error Handling**: Robust error detection and recovery mechanisms
- **Type Safety**: Full type validation for CIP data types

## Architecture

### Protocol Stack
```
Application Layer (Go Application)
    ↓
Protocol Handler Interface
    ↓
EtherNet/IP Handler
    ↓
CIP (Common Industrial Protocol)
    ↓
TCP/IP (Explicit) or UDP/IP (Implicit)
    ↓
Ethernet (IEEE 802.3)
```

### Key Components
1. **EtherNetIPHandler**: Main protocol handler implementing the ProtocolHandler interface
2. **CIP Layer**: Handles CIP encapsulation, session management, and data conversion
3. **Performance Optimizer**: Provides connection pooling, caching, and batch operations
4. **Error Handler**: Comprehensive error categorization and recovery logic
5. **Diagnostic Collector**: Monitors performance and device health

## Getting Started

### Basic Usage

```go
package main

import (
    "log"
    "go.uber.org/zap"
    "github.com/bifrost/go-gateway/internal/protocols"
)

func main() {
    // Create logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    // Create EtherNet/IP handler
    handler := protocols.NewEtherNetIPHandler(logger)
    
    // Define device
    device := &protocols.Device{
        ID:       "plc-001",
        Name:     "Production PLC",
        Protocol: "ethernet-ip",
        Address:  "192.168.1.100",
        Port:     44818, // Standard EtherNet/IP port
        Config:   make(map[string]interface{}),
    }
    
    // Connect to device
    if err := handler.Connect(device); err != nil {
        log.Fatal("Connection failed:", err)
    }
    defer handler.Disconnect(device)
    
    // Define tag
    tag := &protocols.Tag{
        ID:       "temperature",
        Name:     "Temperature Sensor",
        Address:  "TemperatureTag",
        DataType: string(protocols.DataTypeFloat32),
        Writable: false,
    }
    
    // Read tag value
    value, err := handler.ReadTag(device, tag)
    if err != nil {
        log.Fatal("Read failed:", err)
    }
    
    log.Printf("Temperature: %.2f°C", value.(float32))
}
```

### Device Discovery

```go
func discoverDevices(handler protocols.ProtocolHandler) {
    ctx := context.Background()
    
    // Scan network for EtherNet/IP devices
    devices, err := handler.DiscoverDevices(ctx, "192.168.1.0/24")
    if err != nil {
        log.Fatal("Discovery failed:", err)
    }
    
    for _, device := range devices {
        log.Printf("Found device: %s at %s:%d", 
            device.Name, device.Address, device.Port)
        
        // Get device information
        if err := handler.Connect(device); err == nil {
            if info, err := handler.GetDeviceInfo(device); err == nil {
                log.Printf("  Vendor: %s", info.Vendor)
                log.Printf("  Model: %s", info.Model)
                log.Printf("  Serial: %s", info.SerialNumber)
            }
            handler.Disconnect(device)
        }
    }
}
```

## Tag Addressing

### Symbolic Addressing
The most common addressing method for Allen-Bradley PLCs using tag names:

```go
// Simple tag
tag := &protocols.Tag{
    Address: "MyTag",
}

// Array element
tag := &protocols.Tag{
    Address: "MyArray[5]",
}

// Structure member
tag := &protocols.Tag{
    Address: "MyStruct.Member",
}
```

### Instance-Based Addressing
For advanced applications requiring direct CIP object access:

```go
// Symbol object instance 100, attribute 1
tag := &protocols.Tag{
    Address: "Symbol@100.1",
}
```

## Data Types

### Supported CIP Data Types
- **BOOL**: Boolean values (true/false)
- **SINT**: 8-bit signed integer (-128 to 127)
- **INT**: 16-bit signed integer (-32,768 to 32,767)
- **DINT**: 32-bit signed integer (-2,147,483,648 to 2,147,483,647)
- **LINT**: 64-bit signed integer
- **USINT**: 8-bit unsigned integer (0 to 255)
- **UINT**: 16-bit unsigned integer (0 to 65,535)
- **UDINT**: 32-bit unsigned integer (0 to 4,294,967,295)
- **ULINT**: 64-bit unsigned integer
- **REAL**: 32-bit floating point
- **LREAL**: 64-bit floating point
- **STRING**: Variable-length character string

### Data Type Mapping

| CIP Type | Go Type   | Size    | Range                    |
|----------|-----------|---------|--------------------------|
| BOOL     | bool      | 1 bit   | true/false               |
| SINT     | int8      | 1 byte  | -128 to 127              |
| INT      | int16     | 2 bytes | -32,768 to 32,767        |
| DINT     | int32     | 4 bytes | -2^31 to 2^31-1          |
| LINT     | int64     | 8 bytes | -2^63 to 2^63-1          |
| USINT    | uint8     | 1 byte  | 0 to 255                 |
| UINT     | uint16    | 2 bytes | 0 to 65,535             |
| UDINT    | uint32    | 4 bytes | 0 to 2^32-1              |
| ULINT    | uint64    | 8 bytes | 0 to 2^64-1              |
| REAL     | float32   | 4 bytes | IEEE 754 single         |
| LREAL    | float64   | 8 bytes | IEEE 754 double         |
| STRING   | string    | Variable| Up to 255 characters     |

## Performance Optimization

### Connection Pooling
The handler automatically manages connection pooling for optimal performance:

```go
// Connections are automatically pooled and reused
// No explicit pool management required
device1 := &protocols.Device{...}
device2 := &protocols.Device{...}

// Both devices can share connections efficiently
handler.ReadTag(device1, tag1)
handler.ReadTag(device2, tag2)
```

### Batch Operations
Read multiple tags efficiently using batch operations:

```go
tags := []*protocols.Tag{
    {ID: "tag1", Address: "Temperature"},
    {ID: "tag2", Address: "Pressure"},
    {ID: "tag3", Address: "Flow"},
}

// Single network operation for all tags
results, err := handler.ReadMultipleTags(device, tags)
for tagID, value := range results {
    log.Printf("%s: %v", tagID, value)
}
```

### Tag Caching
Enable automatic tag value caching for frequently accessed data:

```go
// Performance optimizer with caching
optimizer := protocols.NewEtherNetIPPerformanceOptimizer(handler, logger)

// Cached reads - much faster for repeated access
value1, _ := optimizer.OptimizedReadTag(device, tag) // Network access
value2, _ := optimizer.OptimizedReadTag(device, tag) // Cache hit
```

## Error Handling

### Error Categories
The implementation categorizes errors for better handling:

- **CONNECTION**: Network connectivity issues
- **SESSION**: CIP session problems
- **CIP**: Protocol-level errors
- **ENCAPSULATION**: Message format issues
- **TIMEOUT**: Communication timeouts
- **SECURITY**: Authentication/authorization failures
- **DATA**: Data conversion errors
- **DEVICE**: Device-specific errors
- **PROTOCOL**: General protocol violations

### Error Recovery
```go
value, err := handler.ReadTag(device, tag)
if err != nil {
    if ethernetIPErr, ok := err.(*protocols.EtherNetIPError); ok {
        if ethernetIPErr.IsRecoverable() {
            // Implement retry logic
            log.Printf("Recoverable error: %s", err)
            // ... retry logic
        } else {
            // Log and skip
            log.Printf("Non-recoverable error: %s", err)
        }
    }
}
```

### CIP Status Codes
Common CIP status codes and their meanings:

| Code | Name                    | Description                  | Recoverable |
|------|-------------------------|------------------------------|-------------|
| 0x00 | Success                 | Operation completed          | N/A         |
| 0x01 | Connection Failure      | Connection failed            | Yes         |
| 0x04 | Path Segment Error      | Invalid tag path             | No          |
| 0x05 | Path Destination Unknown| Tag does not exist           | No          |
| 0x08 | Service Not Supported   | Invalid operation            | No          |
| 0x0F | Privilege Violation     | Access denied                | No          |
| 0x16 | Object Does Not Exist   | Tag not found                | No          |

## Diagnostics and Monitoring

### Basic Diagnostics
```go
diag, err := handler.GetDiagnostics(device)
if err == nil {
    log.Printf("Device Health: %t", diag.IsHealthy)
    log.Printf("Response Time: %v", diag.ResponseTime)
    log.Printf("Success Rate: %.2f%%", diag.SuccessRate*100)
    log.Printf("Uptime: %v", diag.ConnectionUptime)
}
```

### Enhanced Diagnostics
```go
enhancedDiag, err := handler.GetEnhancedDiagnostics(device)
if err == nil {
    log.Printf("Session ID: 0x%08X", enhancedDiag.SessionInfo.SessionID)
    log.Printf("Vendor: %s", enhancedDiag.CIPInfo.VendorName)
    log.Printf("Product: %s", enhancedDiag.CIPInfo.ProductName)
    log.Printf("Serial: %d", enhancedDiag.CIPInfo.SerialNumber)
}
```

### Health Monitoring
```go
health, err := handler.HealthCheck(device)
if err == nil {
    log.Printf("Overall Status: %s", health.Status)
    for checkName, result := range health.Checks {
        log.Printf("  %s: %s - %s", checkName, result.Status, result.Message)
    }
}
```

## Configuration

### Device Configuration
```go
device := &protocols.Device{
    ID:       "plc-001",
    Name:     "Production PLC",
    Protocol: "ethernet-ip",
    Address:  "192.168.1.100",
    Port:     44818,
    Config: map[string]interface{}{
        "connection_timeout": "10s",
        "session_timeout":    "30s",
        "max_packet_size":    1500,
        "enable_keep_alive":  true,
        "retry_count":        3,
        "retry_delay":        "1s",
    },
}
```

### Handler Configuration
```go
handler := protocols.NewEtherNetIPHandler(logger)

// Configure handler-specific settings
if ethernetIPHandler, ok := handler.(*protocols.EtherNetIPHandler); ok {
    // Access handler-specific configuration
    // (Internal configuration is handled automatically)
}
```

## Supported Devices

### Allen-Bradley PLCs
- **ControlLogix**: L6x, L7x series controllers
- **CompactLogix**: L3x series controllers  
- **MicroLogix**: Micro8xx series controllers
- **SoftLogix**: Software-based controllers

### Other EtherNet/IP Devices
- **I/O Modules**: POINT I/O, Flex I/O
- **Drives**: PowerFlex series
- **Safety Systems**: GuardLogix controllers
- **Third-party**: Any CIP-compliant device

## Network Configuration

### IP Address Assignment
- **Static IP**: Recommended for production environments
- **DHCP**: Supported but not recommended for industrial use
- **Subnet**: Ensure devices are on the same subnet or properly routed

### Port Configuration
- **TCP Port 44818**: Standard EtherNet/IP explicit messaging
- **UDP Port 2222**: Standard EtherNet/IP implicit messaging
- **Firewall**: Ensure ports are open in firewalls

### Network Optimization
- **Switched Ethernet**: Use managed switches for best performance
- **VLAN Segmentation**: Separate control and enterprise networks
- **QoS**: Prioritize control traffic
- **Cable Quality**: Use Cat5e or better cables

## Troubleshooting

### Common Issues

#### Connection Failures
```
Error: Failed to connect to EtherNet/IP device
```
**Solutions:**
1. Verify IP address and port
2. Check network connectivity (ping test)
3. Ensure firewall allows port 44818
4. Verify device is powered and operational

#### Session Registration Failures
```
Error: register session failed with status: 0x00000069
```
**Solutions:**
1. Check protocol version compatibility
2. Verify device supports EtherNet/IP
3. Ensure device isn't at connection limit

#### Tag Read/Write Failures
```
Error: CIP error: status 0x16 - Object does not exist
```
**Solutions:**
1. Verify tag name spelling
2. Check tag exists in PLC program
3. Ensure tag is accessible (scope)
4. Verify data type compatibility

#### Performance Issues
```
Warning: High response times detected
```
**Solutions:**
1. Enable connection pooling
2. Use batch operations for multiple tags
3. Implement tag caching
4. Check network utilization

### Debug Logging
Enable debug logging for detailed troubleshooting:

```go
logger := zap.NewDevelopment()
handler := protocols.NewEtherNetIPHandler(logger)
```

### Network Analysis
Use Wireshark with EtherNet/IP dissector to analyze traffic:
1. Capture traffic on port 44818
2. Filter for EtherNet/IP protocol
3. Analyze CIP encapsulation headers
4. Check for error responses

## Best Practices

### Connection Management
- Reuse connections when possible
- Implement proper disconnect handling
- Monitor connection health
- Use connection timeouts

### Tag Organization
- Use descriptive tag names
- Group related tags for batch operations
- Implement tag caching for frequently read data
- Minimize tag address complexity

### Error Handling
- Implement retry logic for recoverable errors
- Log detailed error information
- Monitor error rates
- Implement graceful degradation

### Performance
- Use batch operations for multiple tags
- Enable connection pooling
- Implement tag caching
- Monitor response times

### Security
- Use VLANs to segment networks
- Implement access controls
- Monitor for unauthorized access
- Keep firmware updated

## Examples

### Complete Application Example
See `examples/ethernetip_demo.go` for a complete working example that demonstrates:
- Basic read/write operations
- Device discovery
- Performance optimization
- Diagnostic monitoring
- Error handling

### Running the Demo
```bash
# Basic demo
go run examples/ethernetip_demo.go -address=192.168.1.100 -demo=basic

# Discovery demo
go run examples/ethernetip_demo.go -scan=192.168.1.0/24 -demo=discovery

# Performance demo
go run examples/ethernetip_demo.go -address=192.168.1.100 -demo=performance

# Diagnostics demo
go run examples/ethernetip_demo.go -address=192.168.1.100 -demo=diagnostics
```

## Testing

### Unit Tests
Run the comprehensive test suite:
```bash
go test ./internal/protocols/...
```

### Integration Tests
Test with real hardware:
```bash
go test -tags integration ./internal/protocols/...
```

### Performance Tests
Benchmark performance:
```bash
go test -bench=. ./internal/protocols/...
```

## Roadmap

### Current Status (Phase 4 Complete)
- ✅ Core EtherNet/IP implementation
- ✅ CIP explicit messaging
- ✅ Device discovery
- ✅ Tag read/write operations
- ✅ Error handling and diagnostics
- ✅ Performance optimizations
- ✅ Comprehensive testing

### Future Enhancements
- 🔄 CIP implicit messaging (I/O connections)
- 🔄 CIP Safety protocol support
- 🔄 Advanced security features
- 🔄 OPC UA gateway integration
- 🔄 Real-time data historian
- 🔄 Web-based configuration interface

## Support

### Documentation
- API Reference: See Go package documentation
- Protocol Specification: ODVA CIP specifications
- Allen-Bradley Documentation: Rockwell Automation literature

### Community
- GitHub Issues: Report bugs and feature requests
- Discussions: Technical questions and best practices
- Examples: Community-contributed examples

### Commercial Support
Contact the Bifrost team for:
- Custom integrations
- Performance optimization
- Training and consulting
- Priority support