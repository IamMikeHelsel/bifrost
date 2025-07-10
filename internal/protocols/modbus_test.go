package protocols

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestModbusHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewModbusHandler(logger)

	// Test supported data types
	dataTypes := handler.GetSupportedDataTypes()
	if len(dataTypes) == 0 {
		t.Error("Expected non-empty supported data types")
	}

	// Test address validation
	testAddresses := []struct {
		address string
		valid   bool
	}{
		{"40001", true},
		{"40100", true},
		{"30001", true},
		{"00001", true},
		{"10001", true},
		{"invalid", false},
		{"", false},
		{"50001", false}, // Out of range
	}

	for _, test := range testAddresses {
		err := handler.ValidateTagAddress(test.address)
		if test.valid && err != nil {
			t.Errorf("Expected address %s to be valid, but got error: %v", test.address, err)
		}
		if !test.valid && err == nil {
			t.Errorf("Expected address %s to be invalid, but got no error", test.address)
		}
	}
}

func TestModbusAddressParsing(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewModbusHandler(logger).(*ModbusHandler)

	tests := []struct {
		address   string
		expected  *ModbusAddress
		expectErr bool
	}{
		{
			address: "40001",
			expected: &ModbusAddress{
				FunctionCode: ReadHoldingRegisters,
				Address:      0,
				Count:        1,
				UnitID:       1,
			},
			expectErr: false,
		},
		{
			address: "40100",
			expected: &ModbusAddress{
				FunctionCode: ReadHoldingRegisters,
				Address:      99,
				Count:        1,
				UnitID:       1,
			},
			expectErr: false,
		},
		{
			address: "30001",
			expected: &ModbusAddress{
				FunctionCode: ReadInputRegisters,
				Address:      0,
				Count:        1,
				UnitID:       1,
			},
			expectErr: false,
		},
		{
			address: "00001",
			expected: &ModbusAddress{
				FunctionCode: ReadCoils,
				Address:      0,
				Count:        1,
				UnitID:       1,
			},
			expectErr: false,
		},
		{
			address: "10001",
			expected: &ModbusAddress{
				FunctionCode: ReadDiscreteInputs,
				Address:      0,
				Count:        1,
				UnitID:       1,
			},
			expectErr: false,
		},
		{
			address:   "invalid",
			expectErr: true,
		},
	}

	for _, test := range tests {
		result, err := handler.parseAddress(test.address)

		if test.expectErr {
			if err == nil {
				t.Errorf("Expected error for address %s, but got none", test.address)
			}
			continue
		}

		if err != nil {
			t.Errorf("Unexpected error for address %s: %v", test.address, err)
			continue
		}

		if result.FunctionCode != test.expected.FunctionCode {
			t.Errorf("Wrong function code for %s: expected %d, got %d",
				test.address, test.expected.FunctionCode, result.FunctionCode)
		}
		if result.Address != test.expected.Address {
			t.Errorf("Wrong address for %s: expected %d, got %d",
				test.address, test.expected.Address, result.Address)
		}
	}
}

func TestModbusDataConversion(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewModbusHandler(logger).(*ModbusHandler)

	// Test bool conversion
	boolData := []byte{0x01}
	result, err := handler.convertFromModbus(boolData, "bool", ReadCoils)
	if err != nil {
		t.Errorf("Error converting bool: %v", err)
	}
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}

	// Test uint16 conversion
	uint16Data := []byte{0x01, 0x00} // Big endian 256
	result, err = handler.convertFromModbus(uint16Data, "uint16", ReadHoldingRegisters)
	if err != nil {
		t.Errorf("Error converting uint16: %v", err)
	}
	if result != uint16(256) {
		t.Errorf("Expected 256, got %v", result)
	}

	// Test int16 conversion
	int16Data := []byte{0xFF, 0xFF} // Big endian -1
	result, err = handler.convertFromModbus(int16Data, "int16", ReadHoldingRegisters)
	if err != nil {
		t.Errorf("Error converting int16: %v", err)
	}
	if result != int16(-1) {
		t.Errorf("Expected -1, got %v", result)
	}
}

func BenchmarkModbusAddressParsing(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	handler := NewModbusHandler(logger).(*ModbusHandler)

	addresses := []string{"40001", "40100", "30001", "00001", "10001"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, addr := range addresses {
			handler.parseAddress(addr)
		}
	}
}

func BenchmarkModbusDataConversion(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	handler := NewModbusHandler(logger).(*ModbusHandler)

	testData := []byte{0x01, 0x00}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.convertFromModbus(testData, "uint16", ReadHoldingRegisters)
	}
}

func TestModbusDeviceDiscovery(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewModbusHandler(logger)

	// Test with invalid network range
	ctx := context.Background()
	_, err := handler.DiscoverDevices(ctx, "invalid-range")
	if err == nil {
		t.Error("Expected error for invalid network range")
	}

	// Test with valid but unreachable network range
	// This should not error, but return empty results
	// Use a very small network range and short timeout to avoid long test times
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	devices, err := handler.DiscoverDevices(ctx, "192.168.99.0/30")

	// We expect either no error or a context timeout error
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Unexpected error for valid network range: %v", err)
	}

	// Should return non-nil devices slice
	if devices == nil {
		t.Error("Expected non-nil devices slice")
	}
}

func TestModbusConnectionManagement(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewModbusHandler(logger)

	// Create a test device
	device := &Device{
		ID:       "test-device",
		Name:     "Test Modbus Device",
		Protocol: "modbus-tcp",
		Address:  "127.0.0.1",
		Port:     502,
		Config:   make(map[string]interface{}),
	}

	// Test initial connection state
	if handler.IsConnected(device) {
		t.Error("Device should not be connected initially")
	}

	// Test connection to non-existent device (should fail)
	err := handler.Connect(device)
	if err == nil {
		t.Error("Expected connection to fail for non-existent device")
	}

	// Test disconnection of non-connected device (should not error)
	err = handler.Disconnect(device)
	if err != nil {
		t.Errorf("Unexpected error disconnecting non-connected device: %v", err)
	}
}

func TestModbusDeviceInfo(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := NewModbusHandler(logger)

	device := &Device{
		ID:       "test-device",
		Protocol: "modbus-tcp",
		Address:  "127.0.0.1",
		Port:     502,
	}

	info, err := handler.GetDeviceInfo(device)
	if err != nil {
		t.Errorf("Unexpected error getting device info: %v", err)
	}

	if info == nil {
		t.Error("Expected non-nil device info")
	}

	if len(info.Capabilities) == 0 {
		t.Error("Expected non-empty capabilities")
	}
}
