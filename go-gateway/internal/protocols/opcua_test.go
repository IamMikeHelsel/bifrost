package protocols

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestOPCUAHandler_Creation(t *testing.T) {
	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.clients)
	assert.NotNil(t, handler.connections)
	assert.Equal(t, 100, handler.maxConcurrentReads)
	assert.Equal(t, 1000, handler.batchSize)
	assert.Equal(t, time.Second*5, handler.readTimeout)
}

func TestOPCUAHandler_ValidateTagAddress(t *testing.T) {
	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)

	tests := []struct {
		name     string
		address  string
		wantErr  bool
	}{
		{
			name:    "Valid numeric NodeID",
			address: "ns=1;i=1001",
			wantErr: false,
		},
		{
			name:    "Valid string NodeID",
			address: "ns=2;s=Temperature",
			wantErr: false,
		},
		{
			name:    "Valid GUID NodeID",
			address: "ns=3;g=550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "Empty address",
			address: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.ValidateTagAddress(tt.address)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOPCUAHandler_GetSupportedDataTypes(t *testing.T) {
	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)

	dataTypes := handler.GetSupportedDataTypes()
	
	assert.NotEmpty(t, dataTypes)
	assert.Contains(t, dataTypes, "bool")
	assert.Contains(t, dataTypes, "int32")
	assert.Contains(t, dataTypes, "float")
	assert.Contains(t, dataTypes, "string")
	assert.Contains(t, dataTypes, "datetime")
}

func TestOPCUAHandler_ParseConfig(t *testing.T) {
	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)

	tests := []struct {
		name   string
		config map[string]interface{}
		want   *OPCUAConfig
	}{
		{
			name:   "Empty config",
			config: make(map[string]interface{}),
			want: &OPCUAConfig{
				SecurityPolicy:     "None",
				SecurityMode:       "None",
				AuthPolicy:         "Anonymous",
				SessionTimeout:     time.Minute * 30,
				MaxConcurrentReads: 100,
				BatchSize:          1000,
				ReadTimeout:        time.Second * 5,
			},
		},
		{
			name: "Custom config",
			config: map[string]interface{}{
				"security_policy": "Basic256Sha256",
				"security_mode":   "SignAndEncrypt",
				"auth_policy":     "Username",
				"username":        "testuser",
				"password":        "testpass",
			},
			want: &OPCUAConfig{
				SecurityPolicy:     "Basic256Sha256",
				SecurityMode:       "SignAndEncrypt",
				AuthPolicy:         "Username",
				Username:           "testuser",
				Password:           "testpass",
				SessionTimeout:     time.Minute * 30,
				MaxConcurrentReads: 100,
				BatchSize:          1000,
				ReadTimeout:        time.Second * 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := handler.parseOPCUAConfig(tt.config)
			assert.Equal(t, tt.want.SecurityPolicy, config.SecurityPolicy)
			assert.Equal(t, tt.want.SecurityMode, config.SecurityMode)
			assert.Equal(t, tt.want.AuthPolicy, config.AuthPolicy)
			assert.Equal(t, tt.want.Username, config.Username)
			assert.Equal(t, tt.want.Password, config.Password)
		})
	}
}

func TestOPCUAHandler_BuildEndpointURL(t *testing.T) {
	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)

	tests := []struct {
		name    string
		address string
		port    int
		want    string
	}{
		{
			name:    "IP address with port",
			address: "192.168.1.100",
			port:    4840,
			want:    "opc.tcp://192.168.1.100:4840",
		},
		{
			name:    "Hostname with custom port",
			address: "plc-server",
			port:    4841,
			want:    "opc.tcp://plc-server:4841",
		},
		{
			name:    "Address with default port",
			address: "localhost",
			port:    0,
			want:    "opc.tcp://localhost:4840",
		},
		{
			name:    "Full OPC UA URL",
			address: "opc.tcp://plc.example.com:4840",
			port:    4841, // Should be ignored
			want:    "opc.tcp://plc.example.com:4840",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.buildEndpointURL(tt.address, tt.port)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestOPCUAHandler_ConvertToVariant(t *testing.T) {
	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)

	tests := []struct {
		name     string
		value    interface{}
		dataType string
		wantErr  bool
	}{
		{
			name:     "Boolean value",
			value:    true,
			dataType: "bool",
			wantErr:  false,
		},
		{
			name:     "Integer value",
			value:    int32(42),
			dataType: "int32",
			wantErr:  false,
		},
		{
			name:     "Float value",
			value:    float32(3.14),
			dataType: "float32",
			wantErr:  false,
		},
		{
			name:     "String value",
			value:    "test string",
			dataType: "string",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variant, err := handler.convertToVariant(tt.value, tt.dataType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, variant)
			}
		})
	}
}

func TestOPCUAHandler_DiscoverDevices(t *testing.T) {
	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// This test will work even without a running OPC UA server
	// as it just tests the discovery mechanism
	devices, err := handler.DiscoverDevices(ctx, "192.168.1.0/24")
	
	assert.NoError(t, err)
	// Discovery returns an empty slice when no servers are found, which is expected
	assert.NotNil(t, devices)
	assert.GreaterOrEqual(t, len(devices), 0)
}

func TestOPCUAHandler_ConnectionLifecycle(t *testing.T) {
	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)

	device := &Device{
		ID:       "test-device",
		Name:     "Test OPC UA Device",
		Protocol: "opcua",
		Address:  "localhost",
		Port:     4840,
		Config:   make(map[string]interface{}),
	}

	// Test that device is not connected initially
	assert.False(t, handler.IsConnected(device))

	// Note: Connection test would require a running OPC UA server
	// In a real test environment, you would:
	// 1. Start the virtual OPC UA server
	// 2. Test connection
	// 3. Test read/write operations
	// 4. Test disconnection
}

func TestOPCUAHandler_QualityMapping(t *testing.T) {
	tests := []struct {
		name string
		want Quality
	}{
		{
			name: "Good quality",
			want: QualityGood,
		},
		{
			name: "Bad quality",
			want: QualityBad,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that quality constants are properly defined
			assert.NotEmpty(t, string(tt.want))
		})
	}
}

// Integration tests would require the virtual OPC UA server to be running
func TestOPCUAHandler_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)

	// This would test against the virtual OPC UA server in virtual-devices/opcua-sim/
	device := &Device{
		ID:       "virtual-server",
		Name:     "Virtual OPC UA Server",
		Protocol: "opcua",
		Address:  "localhost",
		Port:     4840,
		Config: map[string]interface{}{
			"security_policy": "None",
			"security_mode":   "None",
			"auth_policy":     "Anonymous",
		},
	}

	// Try to connect (will fail if server is not running)
	err := handler.Connect(device)
	if err != nil {
		t.Logf("Could not connect to virtual OPC UA server: %v", err)
		t.Skip("Virtual OPC UA server not available for integration test")
		return
	}

	defer handler.Disconnect(device)

	// Test basic operations
	assert.True(t, handler.IsConnected(device))

	// Test ping
	err = handler.Ping(device)
	assert.NoError(t, err)

	// Test device info
	info, err := handler.GetDeviceInfo(device)
	assert.NoError(t, err)
	assert.NotNil(t, info)

	// Test diagnostics
	diag, err := handler.GetDiagnostics(device)
	assert.NoError(t, err)
	assert.NotNil(t, diag)
}

// Performance benchmarks
func BenchmarkOPCUA_ValidateTagAddress(b *testing.B) {
	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)
	address := "ns=1;i=1001"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.ValidateTagAddress(address)
	}
}

func BenchmarkOPCUA_ConvertToVariant(b *testing.B) {
	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)
	value := int32(42)
	dataType := "int32"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.convertToVariant(value, dataType)
	}
}

// Performance test for bulk operations
func TestOPCUAHandler_PerformanceTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	logger := zap.NewNop()
	handler := NewOPCUAHandler(logger)

	// Test address validation performance - should handle many addresses quickly
	addresses := make([]string, 10000)
	for i := 0; i < 10000; i++ {
		addresses[i] = "ns=1;i=" + fmt.Sprintf("%d", i+1000)
	}

	start := time.Now()
	for _, addr := range addresses {
		err := handler.ValidateTagAddress(addr)
		require.NoError(t, err)
	}
	duration := time.Since(start)

	// Should validate 10,000 addresses in well under 1 second
	assert.Less(t, duration, time.Second, "Address validation should be fast")

	t.Logf("Validated %d addresses in %v (%.0f addr/sec)", 
		len(addresses), duration, float64(len(addresses))/duration.Seconds())
}