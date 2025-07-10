package protocols

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	
	"go.uber.org/zap"
)

func TestNewEtherNetIPHandler(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger)

	assert.NotNil(t, handler)

	ethernetIPHandler, ok := handler.(*EtherNetIPHandler)
	assert.True(t, ok)
	assert.NotNil(t, ethernetIPHandler.logger)
	assert.NotNil(t, ethernetIPHandler.config)
}

func TestEtherNetIPAddressParsing(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger).(*EtherNetIPHandler)

	tests := []struct {
		name        string
		address     string
		expected    *EtherNetIPAddress
		expectError bool
	}{
		{
			name:    "Simple tag name",
			address: "MyTag",
			expected: &EtherNetIPAddress{
				TagName:     "MyTag",
				IsSymbolic:  true,
				AttributeID: 1,
				DataType:    CIPDataTypeDint,
			},
		},
		{
			name:    "Tag with array index",
			address: "MyArray[5]",
			expected: &EtherNetIPAddress{
				TagName:      "MyArray",
				IsSymbolic:   true,
				IsArray:      true,
				ElementIndex: 5,
				AttributeID:  1,
				DataType:     CIPDataTypeDint,
			},
		},
		{
			name:    "Instance-based addressing",
			address: "Symbol@100.1",
			expected: &EtherNetIPAddress{
				InstanceID:  100,
				AttributeID: 1,
				IsSymbolic:  false,
				DataType:    CIPDataTypeDint,
			},
		},
		{
			name:        "Empty address",
			address:     "",
			expectError: true,
		},
		{
			name:        "Invalid instance format",
			address:     "Symbol@invalid",
			expectError: true,
		},
		{
			name:        "Invalid array format",
			address:     "MyArray[invalid]",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.parseAddress(tt.address)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.TagName, result.TagName)
				assert.Equal(t, tt.expected.IsSymbolic, result.IsSymbolic)
				assert.Equal(t, tt.expected.IsArray, result.IsArray)
				assert.Equal(t, tt.expected.ElementIndex, result.ElementIndex)
				assert.Equal(t, tt.expected.InstanceID, result.InstanceID)
				assert.Equal(t, tt.expected.AttributeID, result.AttributeID)
			}
		})
	}
}

func TestEtherNetIPPathBuilding(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger).(*EtherNetIPHandler)

	t.Run("Symbolic path", func(t *testing.T) {
		path := handler.buildSymbolicPath("TestTag")

		assert.NotNil(t, path)
		assert.Equal(t, byte(0x91), path[0]) // ANSI Extended Symbolic Segment
		assert.Equal(t, byte(7), path[1])    // Length of "TestTag"
		assert.Equal(t, []byte("TestTag"), path[2:9])
	})

	t.Run("Instance path", func(t *testing.T) {
		path := handler.buildInstancePath(CIPClassSymbol, 100, 1)

		assert.NotNil(t, path)
		// Should contain class, instance, and attribute segments
		assert.Contains(t, path, byte(0x20)) // 8-bit class segment
		assert.Contains(t, path, byte(0x24)) // 8-bit instance segment
		assert.Contains(t, path, byte(0x30)) // 8-bit attribute segment
	})
}

func TestCIPDataConversion(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger).(*EtherNetIPHandler)

	t.Run("Boolean conversion", func(t *testing.T) {
		// Test true
		data, err := handler.convertToCIP(true, CIPDataTypeBool)
		assert.NoError(t, err)
		assert.Equal(t, []byte{0x01}, data)

		value, err := handler.convertFromCIP(data, CIPDataTypeBool)
		assert.NoError(t, err)
		assert.Equal(t, true, value)

		// Test false
		data, err = handler.convertToCIP(false, CIPDataTypeBool)
		assert.NoError(t, err)
		assert.Equal(t, []byte{0x00}, data)

		value, err = handler.convertFromCIP(data, CIPDataTypeBool)
		assert.NoError(t, err)
		assert.Equal(t, false, value)
	})

	t.Run("Integer conversion", func(t *testing.T) {
		testValue := int32(12345)

		data, err := handler.convertToCIP(testValue, CIPDataTypeDint)
		assert.NoError(t, err)
		assert.Len(t, data, 4)

		value, err := handler.convertFromCIP(data, CIPDataTypeDint)
		assert.NoError(t, err)
		assert.Equal(t, testValue, value)
	})

	t.Run("String conversion", func(t *testing.T) {
		testValue := "Hello World"

		data, err := handler.convertToCIP(testValue, CIPDataTypeString)
		assert.NoError(t, err)
		assert.True(t, len(data) >= len(testValue)+2) // 2 bytes for length + string

		value, err := handler.convertFromCIP(data, CIPDataTypeString)
		assert.NoError(t, err)
		assert.Equal(t, testValue, value)
	})

	t.Run("Unsupported data type", func(t *testing.T) {
		_, err := handler.convertToCIP(123, 0xFF) // Invalid data type
		assert.Error(t, err)

		_, err = handler.convertFromCIP([]byte{0x01}, 0xFF) // Invalid data type
		assert.Error(t, err)
	})
}

func TestEtherNetIPHandler_GetSupportedDataTypes(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger)

	dataTypes := handler.GetSupportedDataTypes()

	assert.NotNil(t, dataTypes)
	assert.Contains(t, dataTypes, string(DataTypeBool))
	assert.Contains(t, dataTypes, string(DataTypeInt16))
	assert.Contains(t, dataTypes, string(DataTypeInt32))
	assert.Contains(t, dataTypes, string(DataTypeFloat32))
	assert.Contains(t, dataTypes, string(DataTypeString))
}

func TestEtherNetIPHandler_ValidateTagAddress(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger)

	tests := []struct {
		name        string
		address     string
		expectError bool
	}{
		{"Valid tag name", "MyTag", false},
		{"Valid array tag", "MyArray[0]", false},
		{"Valid instance address", "Symbol@100.1", false},
		{"Empty address", "", true},
		{"Invalid array format", "MyArray[invalid]", true},
		{"Invalid instance format", "Symbol@invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.ValidateTagAddress(tt.address)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEtherNetIPHandler_DiscoverDevices(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with invalid network range
	devices, err := handler.DiscoverDevices(ctx, "invalid-range")
	assert.Error(t, err)
	assert.Nil(t, devices)

	// Test with valid network range (should complete without error even if no devices found)
	devices, err = handler.DiscoverDevices(ctx, "127.0.0.1/32")
	assert.NoError(t, err)
	assert.NotNil(t, devices)
	// Note: devices slice might be empty if no EtherNet/IP devices are running locally
}

func TestEtherNetIPHandler_ConnectionManagement(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger)

	device := &Device{
		ID:       "test-device",
		Name:     "Test Device",
		Protocol: "ethernet-ip",
		Address:  "127.0.0.1",
		Port:     44818,
		Config:   make(map[string]interface{}),
	}

	// Test connection with no server (should fail)
	err := handler.Connect(device)
	assert.Error(t, err)

	// Test IsConnected on unconnected device
	connected := handler.IsConnected(device)
	assert.False(t, connected)

	// Test disconnect on unconnected device (should not error)
	err = handler.Disconnect(device)
	assert.NoError(t, err)
}

func TestEtherNetIPHandler_ReadWriteOperations(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger)

	device := &Device{
		ID:       "test-device",
		Name:     "Test Device",
		Protocol: "ethernet-ip",
		Address:  "127.0.0.1",
		Port:     44818,
		Config:   make(map[string]interface{}),
	}

	tag := &Tag{
		ID:       "test-tag",
		Name:     "Test Tag",
		Address:  "TestTag",
		DataType: string(DataTypeInt32),
		Writable: true,
	}

	// Test read operation on unconnected device
	_, err := handler.ReadTag(device, tag)
	assert.Error(t, err)

	// Test write operation on unconnected device
	err = handler.WriteTag(device, tag, int32(123))
	assert.Error(t, err)

	// Test read multiple tags on unconnected device
	tags := []*Tag{tag}
	_, err = handler.ReadMultipleTags(device, tags)
	assert.Error(t, err)

	// Test write to non-writable tag
	tag.Writable = false
	err = handler.WriteTag(device, tag, int32(123))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not writable")
}

func TestEtherNetIPHandler_DiagnosticsAndPing(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger)

	device := &Device{
		ID:       "test-device",
		Name:     "Test Device",
		Protocol: "ethernet-ip",
		Address:  "127.0.0.1",
		Port:     44818,
		Config:   make(map[string]interface{}),
	}

	// Test ping on unconnected device
	err := handler.Ping(device)
	assert.Error(t, err)

	// Test diagnostics on unconnected device
	_, err = handler.GetDiagnostics(device)
	assert.Error(t, err)

	// Test GetDeviceInfo on unconnected device
	_, err = handler.GetDeviceInfo(device)
	assert.Error(t, err)
}

func TestVendorNameMapping(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger).(*EtherNetIPHandler)

	tests := []struct {
		vendorID     uint16
		expectedName string
	}{
		{0x0001, "Rockwell Automation/Allen-Bradley"},
		{0x0002, "Schneider Electric"},
		{0x0003, "Siemens"},
		{0x0004, "GE Fanuc"},
		{0x0005, "Omron"},
		{0x0006, "Mitsubishi Electric"},
		{0x0007, "Honeywell"},
		{0x0008, "Yokogawa"},
		{0x0009, "Emerson"},
		{0x000A, "ABB"},
		{0x9999, "Unknown (0x9999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedName, func(t *testing.T) {
			result := handler.getVendorName(tt.vendorID)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}

func TestCIPConstants(t *testing.T) {
	// Test that all CIP constants are defined correctly
	assert.Equal(t, uint16(0x0000), CIPCommandNOP)
	assert.Equal(t, uint16(0x0065), CIPCommandRegisterSession)
	assert.Equal(t, uint16(0x0066), CIPCommandUnregisterSession)
	assert.Equal(t, uint16(0x006F), CIPCommandSendRRData)

	assert.Equal(t, uint8(0x0E), CIPServiceGetAttributeSingle)
	assert.Equal(t, uint8(0x10), CIPServiceSetAttributeSingle)

	assert.Equal(t, uint16(0x01), CIPClassIdentity)
	assert.Equal(t, uint16(0x6B), CIPClassSymbol)

	assert.Equal(t, uint8(0xC1), CIPDataTypeBool)
	assert.Equal(t, uint8(0xC4), CIPDataTypeDint)
	assert.Equal(t, uint8(0xCA), CIPDataTypeReal)
	assert.Equal(t, uint8(0xD0), CIPDataTypeString)

	assert.Equal(t, 44818, DefaultTCPPort)
	assert.Equal(t, 2222, DefaultUDPPort)
}

func TestEtherNetIPEncapsulationHeader(t *testing.T) {
	header := CIPEncapsulationHeader{
		Command:       CIPCommandRegisterSession,
		Length:        4,
		SessionHandle: 0x12345678,
		Status:        0,
		Options:       0,
	}

	assert.Equal(t, uint16(0x0065), header.Command)
	assert.Equal(t, uint16(4), header.Length)
	assert.Equal(t, uint32(0x12345678), header.SessionHandle)
	assert.Equal(t, uint32(0), header.Status)
	assert.Equal(t, uint32(0), header.Options)
}

func TestCIPIdentityObjectParsing(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger).(*EtherNetIPHandler)

	// Create minimal identity object data
	data := make([]byte, 24)
	// Vendor ID = 0x0001 (Rockwell Automation)
	data[0] = 0x01
	data[1] = 0x00
	// Device Type = 0x000E (Communications Device)
	data[2] = 0x0E
	data[3] = 0x00
	// Product Code = 0x0043 (1756-ENBT)
	data[4] = 0x43
	data[5] = 0x00
	// Revision = 0x0201 (2.1)
	data[6] = 0x01
	data[7] = 0x02
	// Status = 0x0000
	data[8] = 0x00
	data[9] = 0x00
	// Serial Number = 0x12345678
	data[10] = 0x78
	data[11] = 0x56
	data[12] = 0x34
	data[13] = 0x12
	// Product Name Length = 8
	data[14] = 0x08
	// Product Name = "Test PLC"
	copy(data[15:23], []byte("Test PLC"))
	// State = 0x03
	data[23] = 0x03

	identity, err := handler.parseIdentityObject(data)
	assert.NoError(t, err)
	assert.NotNil(t, identity)

	assert.Equal(t, uint16(0x0001), identity.VendorID)
	assert.Equal(t, uint16(0x000E), identity.DeviceType)
	assert.Equal(t, uint16(0x0043), identity.ProductCode)
	assert.Equal(t, uint16(0x0201), identity.Revision)
	assert.Equal(t, uint32(0x12345678), identity.SerialNumber)
	assert.Equal(t, "Test PLC", identity.ProductName)
	assert.Equal(t, uint8(0x03), identity.State)
}

func TestGroupTagsForBatchRead(t *testing.T) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger).(*EtherNetIPHandler)

	// Create test tags
	tags := make([]*Tag, 10)
	for i := 0; i < 10; i++ {
		tags[i] = &Tag{
			ID:      fmt.Sprintf("tag-%d", i),
			Name:    fmt.Sprintf("Tag %d", i),
			Address: fmt.Sprintf("TestTag%d", i),
		}
	}

	// Test with batch size of 3
	batches := handler.groupTagsForBatchRead(tags, 3)

	assert.Len(t, batches, 4) // Should create 4 batches (3,3,3,1)
	assert.Len(t, batches[0], 3)
	assert.Len(t, batches[1], 3)
	assert.Len(t, batches[2], 3)
	assert.Len(t, batches[3], 1)
}

// Benchmark tests
func BenchmarkEtherNetIPAddressParsing(b *testing.B) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger).(*EtherNetIPHandler)

	addresses := []string{
		"SimpleTag",
		"ArrayTag[100]",
		"Symbol@500.1",
		"ComplexTag.SubTag",
		"LongTagNameForBenchmarking",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		address := addresses[i%len(addresses)]
		_, err := handler.parseAddress(address)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCIPDataConversion(b *testing.B) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger).(*EtherNetIPHandler)

	testValue := int32(12345)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := handler.convertToCIP(testValue, CIPDataTypeDint)
		if err != nil {
			b.Fatal(err)
		}

		_, err = handler.convertFromCIP(data, CIPDataTypeDint)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSymbolicPathBuilding(b *testing.B) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger).(*EtherNetIPHandler)

	tagName := "BenchmarkTag"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := handler.buildSymbolicPath(tagName)
		if len(path) == 0 {
			b.Fatal("Empty path returned")
		}
	}
}
