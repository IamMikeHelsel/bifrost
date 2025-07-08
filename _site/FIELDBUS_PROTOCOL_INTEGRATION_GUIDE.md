# Fieldbus Protocol Integration Guide

## Overview

This guide provides detailed technical specifications and implementation examples for integrating new fieldbus protocols into the Bifrost gateway system. It serves as a companion to the high-level implementation plan and focuses on practical development aspects.

## Protocol Handler Implementation Template

### Basic Structure

Each protocol handler should follow this template structure:

```go
package protocols

import (
    "context"
    "sync"
    "time"
    "go.uber.org/zap"
)

// ProtocolNameHandler implements the ProtocolHandler interface for [Protocol Name]
type ProtocolNameHandler struct {
    logger      *zap.Logger
    connections sync.Map // map[string]*ProtocolNameConnection
    config      *ProtocolNameConfig
    // Protocol-specific fields
}

// ProtocolNameConnection represents a connection to a [Protocol Name] device
type ProtocolNameConnection struct {
    deviceID     string
    address      string
    port         int
    isConnected  bool
    lastUsed     time.Time
    mutex        sync.RWMutex
    // Protocol-specific connection state
}

// ProtocolNameConfig holds protocol-specific configuration
type ProtocolNameConfig struct {
    DefaultTimeout    time.Duration `yaml:"default_timeout"`
    MaxConnections    int           `yaml:"max_connections"`
    ConnectionTimeout time.Duration `yaml:"connection_timeout"`
    // Protocol-specific configuration fields
}

// NewProtocolNameHandler creates a new protocol handler
func NewProtocolNameHandler(logger *zap.Logger) ProtocolHandler {
    return &ProtocolNameHandler{
        logger: logger,
        config: &ProtocolNameConfig{
            DefaultTimeout:    5 * time.Second,
            MaxConnections:    50,
            ConnectionTimeout: 10 * time.Second,
        },
    }
}
```

### Required Interface Methods

Each handler must implement all methods of the `ProtocolHandler` interface:

```go
// Connection management
func (h *ProtocolNameHandler) Connect(device *Device) error {
    // Implementation details
}

func (h *ProtocolNameHandler) Disconnect(device *Device) error {
    // Implementation details
}

func (h *ProtocolNameHandler) IsConnected(device *Device) bool {
    // Implementation details
}

// Data operations
func (h *ProtocolNameHandler) ReadTag(device *Device, tag *Tag) (interface{}, error) {
    // Implementation details
}

func (h *ProtocolNameHandler) WriteTag(device *Device, tag *Tag, value interface{}) error {
    // Implementation details
}

func (h *ProtocolNameHandler) ReadMultipleTags(device *Device, tags []*Tag) (map[string]interface{}, error) {
    // Implementation details
}

// Device discovery and information
func (h *ProtocolNameHandler) DiscoverDevices(ctx context.Context, networkRange string) ([]*Device, error) {
    // Implementation details
}

func (h *ProtocolNameHandler) GetDeviceInfo(device *Device) (*DeviceInfo, error) {
    // Implementation details
}

// Protocol-specific operations
func (h *ProtocolNameHandler) GetSupportedDataTypes() []string {
    return []string{"bool", "int16", "int32", "float32", "float64", "string"}
}

func (h *ProtocolNameHandler) ValidateTagAddress(address string) error {
    // Address validation logic
}

// Health and diagnostics
func (h *ProtocolNameHandler) Ping(device *Device) error {
    // Implementation details
}

func (h *ProtocolNameHandler) GetDiagnostics(device *Device) (*Diagnostics, error) {
    // Implementation details
}
```

## Protocol-Specific Implementation Examples

### EtherCAT Implementation

```go
// go-gateway/internal/protocols/ethercat.go
package protocols

/*
#cgo CFLAGS: -I${SRCDIR}/../../../third-party/pysoem/include
#cgo LDFLAGS: -L${SRCDIR}/../../../third-party/pysoem/lib -lpysoem
#include "pysoem_bridge.h"
*/
import "C"

type EtherCATHandler struct {
    logger      *zap.Logger
    connections sync.Map
    config      *EtherCATConfig
    master      *EtherCATMaster
}

type EtherCATConfig struct {
    DefaultTimeout    time.Duration `yaml:"default_timeout"`
    CycleTime         time.Duration `yaml:"cycle_time"`
    NetworkInterface  string        `yaml:"network_interface"`
    MaxSlaves         int           `yaml:"max_slaves"`
    EnableDC          bool          `yaml:"enable_dc"`
}

type EtherCATMaster struct {
    interfaceName string
    slaveCount    int
    slaves        []*EtherCATSlave
    ioMap         []byte
    dcEnabled     bool
    cycleTime     time.Duration
}

type EtherCATSlave struct {
    position      int
    slaveID       uint16
    vendorID      uint32
    productCode   uint32
    state         EtherCATState
    inputSize     int
    outputSize    int
    inputOffset   int
    outputOffset  int
}

type EtherCATState int

const (
    StateInit EtherCATState = iota
    StatePreOp
    StateSafeOp
    StateOp
)

func NewEtherCATHandler(logger *zap.Logger) ProtocolHandler {
    return &EtherCATHandler{
        logger: logger,
        config: &EtherCATConfig{
            DefaultTimeout:   5 * time.Second,
            CycleTime:        time.Millisecond,
            NetworkInterface: "eth0",
            MaxSlaves:        64,
            EnableDC:         true,
        },
    }
}

func (e *EtherCATHandler) Connect(device *Device) error {
    e.logger.Info("Connecting to EtherCAT network", 
        zap.String("interface", e.config.NetworkInterface))
    
    // Initialize EtherCAT master using SOEM library
    result := C.ec_init(C.CString(e.config.NetworkInterface))
    if result <= 0 {
        return fmt.Errorf("failed to initialize EtherCAT master")
    }
    
    // Discover slaves
    slaveCount := C.ec_config_init(0)
    if slaveCount <= 0 {
        return fmt.Errorf("no EtherCAT slaves found")
    }
    
    e.logger.Info("EtherCAT slaves discovered", 
        zap.Int("count", int(slaveCount)))
    
    // Configure slaves and map process data
    ioMapSize := C.ec_config_map(&e.master.ioMap[0])
    if ioMapSize <= 0 {
        return fmt.Errorf("failed to map EtherCAT process data")
    }
    
    // Transition slaves to SAFE_OP state
    C.ec_statecheck(0, C.EC_STATE_SAFE_OP, C.int(e.config.DefaultTimeout.Milliseconds()))
    
    // Start cyclic operation
    go e.cyclicTask()
    
    return nil
}

func (e *EtherCATHandler) cyclicTask() {
    ticker := time.NewTicker(e.config.CycleTime)
    defer ticker.Stop()
    
    for range ticker.C {
        // Send process data
        C.ec_send_processdata()
        
        // Receive process data
        C.ec_receive_processdata(C.EC_TIMEOUTRET)
        
        // Handle distributed clocks if enabled
        if e.config.EnableDC {
            C.ec_sync(C.ec_DCtime, e.config.CycleTime.Nanoseconds(), nil)
        }
    }
}

func (e *EtherCATHandler) ReadTag(device *Device, tag *Tag) (interface{}, error) {
    // Parse EtherCAT address format: "slave.index.subindex" or "slave.input.offset.type"
    addr, err := e.parseEtherCATAddress(tag.Address)
    if err != nil {
        return nil, err
    }
    
    // Read from process data image
    value, err := e.readProcessData(addr)
    if err != nil {
        return nil, err
    }
    
    return e.convertValue(value, tag.DataType), nil
}
```

### BACnet Implementation

```go
// go-gateway/internal/protocols/bacnet.go
package protocols

import (
    "github.com/bacnet/go-bacnet"
)

type BACnetHandler struct {
    logger      *zap.Logger
    connections sync.Map
    config      *BACnetConfig
    client      *bacnet.Client
}

type BACnetConfig struct {
    DefaultTimeout   time.Duration `yaml:"default_timeout"`
    DeviceID         uint32        `yaml:"device_id"`
    NetworkPort      int           `yaml:"network_port"`
    MaxAPDULength    int           `yaml:"max_apdu_length"`
    VendorID         uint16        `yaml:"vendor_id"`
    SegmentationSupported bool     `yaml:"segmentation_supported"`
}

type BACnetDevice struct {
    DeviceID     uint32
    InstanceID   uint32
    ObjectName   string
    VendorName   string
    ModelName    string
    Description  string
    Location     string
    Objects      map[bacnet.ObjectType][]bacnet.ObjectInstance
}

func NewBACnetHandler(logger *zap.Logger) ProtocolHandler {
    return &BACnetHandler{
        logger: logger,
        config: &BACnetConfig{
            DefaultTimeout:        10 * time.Second,
            DeviceID:              1001,
            NetworkPort:           47808,
            MaxAPDULength:         1476,
            VendorID:              999,
            SegmentationSupported: true,
        },
    }
}

func (b *BACnetHandler) Connect(device *Device) error {
    b.logger.Info("Connecting to BACnet device", 
        zap.String("address", device.Address),
        zap.Uint32("device_id", b.config.DeviceID))
    
    // Initialize BACnet client
    client, err := bacnet.NewClient(&bacnet.ClientConfig{
        Port:                  b.config.NetworkPort,
        DeviceID:              b.config.DeviceID,
        MaxAPDULength:         b.config.MaxAPDULength,
        SegmentationSupported: b.config.SegmentationSupported,
        VendorID:              b.config.VendorID,
    })
    if err != nil {
        return fmt.Errorf("failed to create BACnet client: %v", err)
    }
    
    b.client = client
    
    // Test connectivity by reading device object
    deviceInfo, err := b.readDeviceObject(device)
    if err != nil {
        return fmt.Errorf("failed to read device object: %v", err)
    }
    
    b.logger.Info("BACnet device connected", 
        zap.String("device_name", deviceInfo.ObjectName),
        zap.String("vendor", deviceInfo.VendorName))
    
    return nil
}

func (b *BACnetHandler) ReadTag(device *Device, tag *Tag) (interface{}, error) {
    // Parse BACnet address format: "object_type:instance:property[:array_index]"
    addr, err := b.parseBACnetAddress(tag.Address)
    if err != nil {
        return nil, err
    }
    
    // Create read property request
    request := &bacnet.ReadPropertyRequest{
        ObjectType:     addr.ObjectType,
        InstanceNumber: addr.InstanceNumber,
        PropertyID:     addr.PropertyID,
        ArrayIndex:     addr.ArrayIndex,
    }
    
    // Send request to device
    response, err := b.client.ReadProperty(device.Address, request)
    if err != nil {
        return nil, fmt.Errorf("BACnet read property failed: %v", err)
    }
    
    // Convert BACnet value to native type
    return b.convertBACnetValue(response.Value, tag.DataType), nil
}

func (b *BACnetHandler) DiscoverDevices(ctx context.Context, networkRange string) ([]*Device, error) {
    b.logger.Info("Starting BACnet device discovery", 
        zap.String("network_range", networkRange))
    
    devices := make([]*Device, 0)
    
    // Send Who-Is broadcast
    whoIsRequest := &bacnet.WhoIsRequest{
        LowLimit:  1,
        HighLimit: 4194303, // Maximum device instance number
    }
    
    responses, err := b.client.WhoIs(whoIsRequest)
    if err != nil {
        return nil, fmt.Errorf("BACnet Who-Is request failed: %v", err)
    }
    
    // Process I-Am responses
    for _, response := range responses {
        device := &Device{
            ID:       fmt.Sprintf("bacnet-%d", response.DeviceID),
            Name:     fmt.Sprintf("BACnet Device %d", response.DeviceID),
            Protocol: "bacnet",
            Address:  response.Address,
            Port:     b.config.NetworkPort,
            Config: map[string]interface{}{
                "device_id":      response.DeviceID,
                "max_apdu_length": response.MaxAPDULength,
                "segmentation":   response.SegmentationSupported,
                "vendor_id":      response.VendorID,
            },
        }
        
        devices = append(devices, device)
    }
    
    b.logger.Info("BACnet device discovery completed", 
        zap.Int("devices_found", len(devices)))
    
    return devices, nil
}
```

### ProfiNet Implementation

```go
// go-gateway/internal/protocols/profinet.go
package protocols

/*
#cgo CFLAGS: -I${SRCDIR}/../../../third-party/pnio-dcp/include
#cgo LDFLAGS: -L${SRCDIR}/../../../third-party/pnio-dcp/lib -lpnio_dcp
#include "pnio_dcp_bridge.h"
*/
import "C"

type ProfiNetHandler struct {
    logger      *zap.Logger
    connections sync.Map
    config      *ProfiNetConfig
    dcpClient   *DCPClient
    rtEngine    *ProfiNetRTEngine
}

type ProfiNetConfig struct {
    DefaultTimeout   time.Duration `yaml:"default_timeout"`
    NetworkInterface string        `yaml:"network_interface"`
    CycleTime        time.Duration `yaml:"cycle_time"`
    VendorID         uint16        `yaml:"vendor_id"`
    DeviceID         uint16        `yaml:"device_id"`
    MaxDevices       int           `yaml:"max_devices"`
}

type DCPClient struct {
    interfaceName string
    socket        int
}

type ProfiNetRTEngine struct {
    devices    map[string]*ProfiNetDevice
    cycleTime  time.Duration
    running    bool
    stopChan   chan struct{}
}

type ProfiNetDevice struct {
    Name         string
    TypeOfStation string
    Role         string
    IPAddress    string
    MACAddress   string
    VendorID     uint16
    DeviceID     uint16
    GSDML        *GSDMLData
    IOModules    []*IOModule
}

type GSDMLData struct {
    DeviceAccess *DeviceAccessData
    Modules      []*ModuleInfo
    Submodules   []*SubmoduleInfo
}

type IOModule struct {
    SlotNumber   uint16
    ModuleID     uint32
    InputSize    uint16
    OutputSize   uint16
    Submodules   []*IOSubmodule
}

func NewProfiNetHandler(logger *zap.Logger) ProtocolHandler {
    return &ProfiNetHandler{
        logger: logger,
        config: &ProfiNetConfig{
            DefaultTimeout:   10 * time.Second,
            NetworkInterface: "eth0",
            CycleTime:        time.Millisecond * 2,
            VendorID:         0x012A, // Example vendor ID
            DeviceID:         0x0001,
            MaxDevices:       32,
        },
    }
}

func (p *ProfiNetHandler) Connect(device *Device) error {
    p.logger.Info("Connecting to ProfiNet device", 
        zap.String("interface", p.config.NetworkInterface))
    
    // Initialize DCP client for device discovery
    dcpClient, err := p.initializeDCPClient()
    if err != nil {
        return fmt.Errorf("failed to initialize DCP client: %v", err)
    }
    p.dcpClient = dcpClient
    
    // Initialize ProfiNet RT engine
    rtEngine, err := p.initializeRTEngine()
    if err != nil {
        return fmt.Errorf("failed to initialize RT engine: %v", err)
    }
    p.rtEngine = rtEngine
    
    return nil
}

func (p *ProfiNetHandler) DiscoverDevices(ctx context.Context, networkRange string) ([]*Device, error) {
    p.logger.Info("Starting ProfiNet device discovery via DCP")
    
    devices := make([]*Device, 0)
    
    // Send DCP Identify All request using pnio-dcp library
    result := C.dcp_identify_all(C.CString(p.config.NetworkInterface))
    if result < 0 {
        return nil, fmt.Errorf("DCP identify request failed")
    }
    
    // Process DCP responses
    for i := 0; i < int(result); i++ {
        var dcpResponse C.dcp_response_t
        C.dcp_get_response(C.int(i), &dcpResponse)
        
        device := &Device{
            ID:       fmt.Sprintf("profinet-%s", C.GoString(dcpResponse.name_of_station)),
            Name:     C.GoString(dcpResponse.name_of_station),
            Protocol: "profinet",
            Address:  C.GoString(dcpResponse.ip_address),
            Port:     0, // ProfiNet uses Ethernet frames, not UDP/TCP ports
            Config: map[string]interface{}{
                "name_of_station": C.GoString(dcpResponse.name_of_station),
                "type_of_station": C.GoString(dcpResponse.type_of_station),
                "role":            C.GoString(dcpResponse.role),
                "mac_address":     C.GoString(dcpResponse.mac_address),
                "vendor_id":       uint16(dcpResponse.vendor_id),
                "device_id":       uint16(dcpResponse.device_id),
            },
        }
        
        devices = append(devices, device)
    }
    
    p.logger.Info("ProfiNet device discovery completed", 
        zap.Int("devices_found", len(devices)))
    
    return devices, nil
}

func (p *ProfiNetHandler) ReadTag(device *Device, tag *Tag) (interface{}, error) {
    // Parse ProfiNet address format: "slot.subslot.offset.type"
    addr, err := p.parseProfiNetAddress(tag.Address)
    if err != nil {
        return nil, err
    }
    
    // Read from RT process image
    value, err := p.rtEngine.ReadProcessData(device.ID, addr)
    if err != nil {
        return nil, err
    }
    
    return p.convertValue(value, tag.DataType), nil
}
```

## Testing Framework

### Unit Test Template

```go
// internal/protocols/protocol_test.go
func TestProtocolNameHandler(t *testing.T) {
    logger := zap.NewNop()
    handler := NewProtocolNameHandler(logger)
    
    t.Run("Handler Creation", func(t *testing.T) {
        assert.NotNil(t, handler)
        assert.Equal(t, []string{"bool", "int16", "int32", "float32", "float64", "string"}, 
                     handler.GetSupportedDataTypes())
    })
    
    t.Run("Address Validation", func(t *testing.T) {
        validAddresses := []string{
            "valid.address.1",
            "another.valid.address",
        }
        
        for _, addr := range validAddresses {
            err := handler.ValidateTagAddress(addr)
            assert.NoError(t, err, "Address %s should be valid", addr)
        }
        
        invalidAddresses := []string{
            "",
            "invalid..address",
            "toolong.address.with.too.many.parts.for.protocol",
        }
        
        for _, addr := range invalidAddresses {
            err := handler.ValidateTagAddress(addr)
            assert.Error(t, err, "Address %s should be invalid", addr)
        }
    })
}
```

### Integration Test Template

```go
// internal/protocols/protocol_integration_test.go
// +build integration

func TestProtocolNameIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    logger := zap.NewNop()
    handler := NewProtocolNameHandler(logger)
    
    // Test device configuration
    device := &Device{
        ID:       "test-device-1",
        Name:     "Test Device",
        Protocol: "protocolname",
        Address:  "192.168.1.100",
        Port:     502,
        Config: map[string]interface{}{
            "timeout": 5000,
        },
    }
    
    t.Run("Device Connection", func(t *testing.T) {
        err := handler.Connect(device)
        assert.NoError(t, err)
        assert.True(t, handler.IsConnected(device))
        
        defer func() {
            err := handler.Disconnect(device)
            assert.NoError(t, err)
        }()
    })
    
    t.Run("Tag Operations", func(t *testing.T) {
        err := handler.Connect(device)
        require.NoError(t, err)
        defer handler.Disconnect(device)
        
        tag := &Tag{
            ID:       "test-tag",
            Name:     "Test Tag",
            Address:  "test.address.1",
            DataType: "int16",
        }
        
        // Test read operation
        value, err := handler.ReadTag(device, tag)
        assert.NoError(t, err)
        assert.NotNil(t, value)
        
        // Test write operation
        err = handler.WriteTag(device, tag, int16(42))
        assert.NoError(t, err)
        
        // Verify write
        value, err = handler.ReadTag(device, tag)
        assert.NoError(t, err)
        assert.Equal(t, int16(42), value)
    })
}
```

## Performance Benchmarking

### Benchmark Template

```go
// internal/protocols/protocol_benchmark_test.go
func BenchmarkProtocolNameOperations(b *testing.B) {
    logger := zap.NewNop()
    handler := NewProtocolNameHandler(logger)
    
    device := &Device{
        ID:       "benchmark-device",
        Protocol: "protocolname",
        Address:  "192.168.1.100",
        Port:     502,
    }
    
    err := handler.Connect(device)
    if err != nil {
        b.Fatalf("Failed to connect: %v", err)
    }
    defer handler.Disconnect(device)
    
    tag := &Tag{
        ID:       "benchmark-tag",
        Address:  "benchmark.address",
        DataType: "int16",
    }
    
    b.Run("ReadTag", func(b *testing.B) {
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            _, err := handler.ReadTag(device, tag)
            if err != nil {
                b.Fatalf("Read failed: %v", err)
            }
        }
    })
    
    b.Run("WriteTag", func(b *testing.B) {
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            err := handler.WriteTag(device, tag, int16(i))
            if err != nil {
                b.Fatalf("Write failed: %v", err)
            }
        }
    })
    
    b.Run("ReadMultipleTags", func(b *testing.B) {
        tags := make([]*Tag, 10)
        for i := range tags {
            tags[i] = &Tag{
                ID:       fmt.Sprintf("tag-%d", i),
                Address:  fmt.Sprintf("address.%d", i),
                DataType: "int16",
            }
        }
        
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            _, err := handler.ReadMultipleTags(device, tags)
            if err != nil {
                b.Fatalf("Multi-read failed: %v", err)
            }
        }
    })
}
```

## Documentation Guidelines

### API Documentation

Each protocol handler should include comprehensive documentation:

```go
// Package documentation
/*
Package protocols provides industrial protocol implementations for the Bifrost gateway.

This package contains protocol handlers that implement the unified ProtocolHandler 
interface, enabling consistent access to various industrial communication protocols.

Supported Protocols:
- Modbus TCP/RTU (production ready)
- EtherNet/IP (CIP) (production ready)
- EtherCAT (implementation in progress)
- BACnet (implementation in progress)
- ProfiNet (implementation in progress)

Usage Example:
    logger := zap.NewDevelopment()
    handler := protocols.NewBACnetHandler(logger)
    
    device := &protocols.Device{
        ID:       "bacnet-device-1",
        Protocol: "bacnet",
        Address:  "192.168.1.100",
        Port:     47808,
    }
    
    err := handler.Connect(device)
    if err != nil {
        log.Fatal(err)
    }
    defer handler.Disconnect(device)
    
    tag := &protocols.Tag{
        Address:  "analog-input:1:present-value",
        DataType: "float32",
    }
    
    value, err := handler.ReadTag(device, tag)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Tag value: %v\n", value)
*/
```

### README Template

Each protocol should have a dedicated README file:

```markdown
# Protocol Name Implementation

## Overview

Brief description of the protocol and its use cases in industrial automation.

## Features

- List of supported features
- Performance characteristics
- Limitations and known issues

## Configuration

### Device Configuration
```yaml
device:
  protocol: protocolname
  address: "192.168.1.100"
  port: 502
  config:
    timeout: 5000
    protocol_specific_setting: value
```

### Tag Addressing

Describe the tag addressing format and provide examples:

- Simple tag: `tag.name`
- Complex tag: `device.module.tag[index]`

## Examples

### Basic Usage
```go
// Go code example
```

### Advanced Features
```go
// Advanced usage examples
```

## Troubleshooting

Common issues and their solutions.

## Performance

Benchmark results and optimization tips.

## Contributing

Guidelines for contributing to this protocol implementation.
```

This guide provides the foundation for implementing new fieldbus protocols in a consistent, maintainable, and high-performance manner while following established patterns in the Bifrost codebase.