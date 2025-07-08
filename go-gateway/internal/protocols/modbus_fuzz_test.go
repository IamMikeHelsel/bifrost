package protocols

import (
	"testing"

	"go.uber.org/zap"
)

// FuzzModbusAddressParsing tests the Modbus address parsing functionality for security vulnerabilities
func FuzzModbusAddressParsing(f *testing.F) {
	// Add valid Modbus address seeds
	f.Add("40001")    // Holding Register
	f.Add("30001")    // Input Register
	f.Add("00001")    // Coil
	f.Add("10001")    // Discrete Input
	f.Add("40100")    // Valid holding register
	f.Add("49999")    // Max holding register

	logger, _ := zap.NewDevelopment()
	handler := NewModbusHandler(logger).(*ModbusHandler)

	f.Fuzz(func(t *testing.T, address string) {
		// Ensure parsing doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Modbus address parsing panicked with input %q: %v", address, r)
			}
		}()

		// Test address parsing
		addr, err := handler.parseAddress(address)
		if err != nil {
			// Invalid input is expected for many fuzz inputs
			return
		}

		// Validate parsed address structure for security issues
		if addr == nil {
			t.Error("parseAddress returned nil address without error")
			return
		}

		// Validate function code is within expected range
		if addr.FunctionCode < 1 || addr.FunctionCode > 16 {
			t.Errorf("Invalid function code %d for address %q", addr.FunctionCode, address)
		}

		// Validate address ranges don't cause integer overflow
		if addr.Address > 65535 {
			t.Errorf("Address overflow: %d for input %q", addr.Address, address)
		}

		// Validate count is reasonable
		if addr.Count == 0 || addr.Count > 2000 {
			t.Errorf("Invalid count %d for address %q", addr.Count, address)
		}
	})
}

// FuzzModbusAddressValidation tests the address validation functionality
func FuzzModbusAddressValidation(f *testing.F) {
	// Add valid tag addresses that should pass validation
	f.Add("40001")
	f.Add("30001") 
	f.Add("00001")
	f.Add("10001")

	logger, _ := zap.NewDevelopment()
	handler := NewModbusHandler(logger).(*ModbusHandler)

	f.Fuzz(func(t *testing.T, address string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Modbus address validation panicked with input %q: %v", address, r)
			}
		}()

		// Test address validation
		err := handler.ValidateTagAddress(address)
		
		// Validation should never panic, only return error for invalid addresses
		// No specific validation needed here as invalid addresses are expected
		_ = err
	})
}

// FuzzModbusDataConversion tests the data conversion functions for security issues
func FuzzModbusDataConversion(f *testing.F) {
	// Add sample binary data that represents Modbus register responses
	f.Add([]byte{0x00, 0x01})              // Single register
	f.Add([]byte{0x12, 0x34, 0x56, 0x78})  // Two registers  
	f.Add([]byte{0xFF, 0xFF})              // Max values
	f.Add([]byte{0x00, 0x00})              // Zero values
	f.Add([]byte{})                        // Empty data

	logger, _ := zap.NewDevelopment()
	handler := NewModbusHandler(logger).(*ModbusHandler)

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Modbus data conversion panicked with input %v: %v", data, r)
			}
		}()

		// Test conversion for different data types
		dataTypes := []string{"uint16", "int16", "uint32", "int32", "float32", "bool"}
		functionCodes := []ModbusFunctionCode{ReadHoldingRegisters, ReadInputRegisters, ReadCoils}

		for _, dataType := range dataTypes {
			for _, funcCode := range functionCodes {
				result, err := handler.convertFromModbus(data, dataType, funcCode)
				if err != nil {
					// Invalid conversions are expected
					continue
				}

				// Validate result is not nil for successful conversions
				if result == nil {
					t.Errorf("convertFromModbus returned nil result without error for data type %s", dataType)
				}
			}
		}
	})
}

// FuzzModbusConfigParsing tests configuration parsing for security vulnerabilities
func FuzzModbusConfigParsing(f *testing.F) {
	// Add valid configuration unit IDs
	f.Add(1)
	f.Add(255)
	f.Add(0)
	f.Add(-1)
	f.Add(300)

	logger, _ := zap.NewDevelopment()

	f.Fuzz(func(t *testing.T, unitID int) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Modbus config processing panicked with unit ID %d: %v", unitID, r)
			}
		}()

		// Create a test device with the fuzzed unit ID
		config := make(map[string]interface{})
		config["unit_id"] = unitID

		device := &Device{
			ID:       "test-device",
			Name:     "Test Device", 
			Protocol: "modbus-tcp",
			Address:  "127.0.0.1",
			Port:     502,
			Config:   config,
		}

		handler := NewModbusHandler(logger).(*ModbusHandler)

		// Test that config processing doesn't crash
		processedUnitID := handler.getUnitID(device)
		
		// Validate unit ID is within valid range (should be clamped)
		if processedUnitID > 255 {
			t.Errorf("Unit ID not properly validated: %d", processedUnitID)
		}
	})
}