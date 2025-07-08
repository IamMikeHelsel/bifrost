package protocols

import (
	"testing"

	"go.uber.org/zap"
)

// FuzzOPCUANodeIDParsing tests OPC-UA Node ID parsing for security vulnerabilities
func FuzzOPCUANodeIDParsing(f *testing.F) {
	// Add valid OPC-UA Node ID examples
	f.Add("ns=2;i=1001")           // Numeric identifier
	f.Add("ns=1;s=Temperature")    // String identifier
	f.Add("ns=0;i=2253")          // Standard OPC-UA node
	f.Add("i=85")                 // Default namespace
	f.Add("s=MyVariable")         // String in default namespace

	logger, _ := zap.NewDevelopment()
	handler := NewOPCUAHandler(logger)

	f.Fuzz(func(t *testing.T, nodeID string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("OPC-UA node ID parsing panicked with input %q: %v", nodeID, r)
			}
		}()

		// Create a test device and tag
		device := &Device{
			ID:       "test-opcua-device",
			Name:     "Test OPC-UA Device",
			Protocol: "opcua",
			Address:  "opc.tcp://localhost:4840",
			Port:     4840,
			Config:   make(map[string]interface{}),
		}

		tag := &Tag{
			ID:       "test-tag",
			Name:     "Test Tag",
			Address:  nodeID,
			DataType: "variant",
			Writable: false,
		}

		// Test that node ID processing doesn't crash
		// Note: Since OPC-UA implementation is minimal, we mainly test for panics
		_, err := handler.ReadTag(device, tag)
		
		// Invalid node IDs are expected to return errors, not crash
		_ = err
	})
}

// FuzzOPCUAConnectionString tests OPC-UA connection string parsing
func FuzzOPCUAConnectionString(f *testing.F) {
	// Add valid OPC-UA connection string examples  
	f.Add("opc.tcp://localhost:4840")
	f.Add("opc.tcp://192.168.1.100:4840/freeopcua/server/")
	f.Add("opc.tcp://server.example.com:4840")
	f.Add("opc.tcp://[::1]:4840")

	logger, _ := zap.NewDevelopment()
	handler := NewOPCUAHandler(logger)

	f.Fuzz(func(t *testing.T, connectionString string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("OPC-UA connection parsing panicked with input %q: %v", connectionString, r)
			}
		}()

		// Create test device with fuzzed connection string
		device := &Device{
			ID:       "test-device",
			Name:     "Test Device",
			Protocol: "opcua", 
			Address:  connectionString,
			Port:     4840,
			Config:   make(map[string]interface{}),
		}

		// Test connection attempt (should not crash)
		err := handler.Connect(device)
		if err != nil {
			// Connection failures are expected for invalid URLs
			return
		}

		// Test disconnection if connect succeeded
		_ = handler.Disconnect(device)
	})
}

// FuzzOPCUAConfigParsing tests OPC-UA configuration parsing for security issues
func FuzzOPCUAConfigParsing(f *testing.F) {
	// Add configuration values that could be problematic
	f.Add("none")           // security_mode
	f.Add("basic256sha256") // security_policy
	f.Add("anonymous")      // auth method
	f.Add("")               // empty string
	f.Add("very_long_string_that_might_cause_buffer_overflow_issues_if_not_handled_properly")

	logger, _ := zap.NewDevelopment()

	f.Fuzz(func(t *testing.T, configValue string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("OPC-UA config processing panicked with input %q: %v", configValue, r)
			}
		}()

		// Create test device with fuzzed config value
		config := make(map[string]interface{})
		config["security_mode"] = configValue
		config["security_policy"] = configValue
		config["endpoint_url"] = "opc.tcp://localhost:4840"

		device := &Device{
			ID:       "test-device",
			Name:     "Test Device",
			Protocol: "opcua",
			Address:  "opc.tcp://localhost:4840", 
			Port:     4840,
			Config:   config,
		}

		handler := NewOPCUAHandler(logger)

		// Test that config processing doesn't crash during connection attempt
		err := handler.Connect(device)
		_ = err // Connection may fail, that's expected

		// Test IsConnected with the device
		connected := handler.IsConnected(device)
		_ = connected // Result doesn't matter, just shouldn't crash
	})
}

// FuzzOPCUABinaryProtocol tests OPC-UA binary protocol parsing (placeholder for future implementation)
func FuzzOPCUABinaryProtocol(f *testing.F) {
	// Standard OPC-UA Hello message structure
	f.Add([]byte{0x48, 0x45, 0x4c, 0x4f}) // "HELO" message type
	f.Add([]byte{0x41, 0x43, 0x4b, 0x4e}) // "ACKN" message type
	f.Add([]byte{0x45, 0x52, 0x52, 0x4f}) // "ERRO" message type
	f.Add([]byte{})                        // Empty message
	f.Add([]byte{0x00, 0x01, 0x02, 0x03}) // Random bytes

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("OPC-UA binary protocol parsing panicked with input %v: %v", data, r)
			}
		}()

		// This is a placeholder for future OPC-UA binary protocol implementation
		// For now, just ensure the test framework can handle binary data
		
		// Test basic validation that would be expected in a real parser
		if len(data) >= 4 {
			// Check for standard OPC-UA message types
			msgType := string(data[:4])
			validTypes := []string{"HELO", "ACKN", "ERRO", "OPNF", "CLOF", "MSGF"}
			
			for _, validType := range validTypes {
				if msgType == validType {
					// Found valid message type, would continue parsing in real implementation
					break
				}
			}
		}

		// In a real implementation, this would call a proper OPC-UA binary parser
		// For now, this validates that fuzzing infrastructure works
	})
}