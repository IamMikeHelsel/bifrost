package gateway

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// FuzzRESTAPIEndpoints tests REST API endpoint handlers for security vulnerabilities
func FuzzRESTAPIEndpoints(f *testing.F) {
	// Add sample valid JSON payloads
	f.Add([]byte(`{"device_id": "plc-001", "tag_ids": ["temp1", "pressure"]}`))
	f.Add([]byte(`{"device_id": "controller-002", "operation": "read"}`))
	f.Add([]byte(`{"devices": [{"id": "test", "protocol": "modbus"}]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))

	logger, _ := zap.NewDevelopment()
	config := &Config{
		Port:           8080,
		GRPCPort:       9090,
		MaxConnections: 100,
		DataBufferSize: 1000,
		UpdateInterval: time.Second,
		EnableMetrics:  false, // Disable for testing
		LogLevel:       "info",
	}

	gateway := NewIndustrialGateway(config, logger)

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("API handler panicked with input %v: %v", string(data), r)
			}
		}()

		// Test different API endpoints with fuzzed data
		endpoints := []struct {
			path   string
			method string
		}{
			{"/api/devices", "GET"},
			{"/api/devices", "POST"},
			{"/api/discovery", "GET"},
			{"/api/tags/read", "POST"},
			{"/api/tags/write", "POST"},
		}

		for _, endpoint := range endpoints {
			req := httptest.NewRequest(endpoint.method, endpoint.path, bytes.NewReader(data))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			// Route to appropriate handler
			switch endpoint.path {
			case "/api/devices":
				gateway.handleDevices(w, req)
			case "/api/discovery":
				gateway.handleDiscovery(w, req)
			case "/api/tags/read":
				gateway.handleTagRead(w, req)
			case "/api/tags/write":
				gateway.handleTagWrite(w, req)
			}

			// Validate response doesn't contain sensitive information
			if w.Code >= 400 {
				body := w.Body.String()
				if strings.Contains(strings.ToLower(body), "internal") ||
					strings.Contains(strings.ToLower(body), "panic") ||
					strings.Contains(strings.ToLower(body), "stack") {
					t.Errorf("Error response contains sensitive information: %s", body)
				}
			}

			// Validate response headers
			contentType := w.Header().Get("Content-Type")
			if contentType != "" && !strings.Contains(contentType, "application/json") {
				t.Errorf("Unexpected content type: %s", contentType)
			}
		}
	})
}

// FuzzWebSocketConnection tests WebSocket connection handling for security issues
func FuzzWebSocketConnection(f *testing.F) {
	// Add sample WebSocket message payloads
	f.Add([]byte(`{"type": "subscribe", "device_id": "plc-001"}`))
	f.Add([]byte(`{"type": "unsubscribe", "device_id": "plc-001"}`))
	f.Add([]byte(`{"type": "ping"}`))
	f.Add([]byte(`invalid json`))
	f.Add([]byte(`{}`))

	logger, _ := zap.NewDevelopment()
	config := &Config{
		Port:           8080,
		GRPCPort:       9090,
		MaxConnections: 100,
		DataBufferSize: 1000,
		UpdateInterval: time.Second,
		EnableMetrics:  false,
		LogLevel:       "info",
	}

	gateway := NewIndustrialGateway(config, logger)

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("WebSocket handler panicked with input %v: %v", string(data), r)
			}
		}()

		// Create WebSocket request with fuzzed data as query parameter
		req := httptest.NewRequest("GET", "/ws?data="+string(data), nil)
		req.Header.Set("Connection", "upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Key", "test-key")
		req.Header.Set("Sec-WebSocket-Version", "13")

		w := httptest.NewRecorder()

		// Test WebSocket handler (note: this won't complete the upgrade in test, but shouldn't crash)
		gateway.handleWebSocket(w, req)

		// Validate no sensitive information is leaked in response
		if w.Code >= 400 {
			body := w.Body.String()
			if strings.Contains(strings.ToLower(body), "internal") {
				t.Errorf("WebSocket error response contains internal information: %s", body)
			}
		}
	})
}

// FuzzConfigurationParsing tests configuration file parsing for security vulnerabilities
func FuzzConfigurationParsing(f *testing.F) {
	validConfig := `
gateway:
  port: 8080
  grpc_port: 9090
  max_connections: 1000
  data_buffer_size: 10000
  update_interval: 1s
  enable_metrics: true
  log_level: info

protocols:
  modbus:
    default_timeout: 5s
    default_unit_id: 1
    max_connections: 100
`

	f.Add([]byte(validConfig))
	f.Add([]byte(`gateway: {port: 8080}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`invalid yaml: [[[`))
	f.Add([]byte(`gateway:\n  port: -1`))      // Negative port
	f.Add([]byte(`gateway:\n  port: 999999`))  // Very high port

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Config parser panicked with input %v: %v", string(data), r)
			}
		}()

		// Test YAML parsing
		type TestConfig struct {
			Gateway struct {
				Port           int           `yaml:"port"`
				GRPCPort       int           `yaml:"grpc_port"`
				MaxConnections int           `yaml:"max_connections"`
				DataBufferSize int           `yaml:"data_buffer_size"`
				UpdateInterval time.Duration `yaml:"update_interval"`
				EnableMetrics  bool          `yaml:"enable_metrics"`
				LogLevel       string        `yaml:"log_level"`
			} `yaml:"gateway"`

			Protocols struct {
				Modbus struct {
					DefaultTimeout    time.Duration `yaml:"default_timeout"`
					DefaultUnitID     int           `yaml:"default_unit_id"`
					MaxConnections    int           `yaml:"max_connections"`
					ConnectionTimeout time.Duration `yaml:"connection_timeout"`
					ReadTimeout       time.Duration `yaml:"read_timeout"`
					WriteTimeout      time.Duration `yaml:"write_timeout"`
					EnableKeepAlive   bool          `yaml:"enable_keep_alive"`
				} `yaml:"modbus"`
			} `yaml:"protocols"`
		}

		config := &TestConfig{}
		err := yaml.Unmarshal(data, config)
		if err != nil {
			// Invalid YAML is expected for many fuzz inputs
			return
		}

		// Validate configuration doesn't contain dangerous values
		if config.Gateway.Port < 0 || config.Gateway.Port > 65535 {
			// Invalid port range - this should be caught by validation
			return
		}

		if config.Gateway.MaxConnections < 0 || config.Gateway.MaxConnections > 100000 {
			// Invalid connection limit - should be validated
			return
		}

		if config.Gateway.DataBufferSize < 0 || config.Gateway.DataBufferSize > 1000000 {
			// Invalid buffer size - should be validated
			return
		}

		// Test that valid config can be used to create a gateway
		if config.Gateway.Port > 0 && config.Gateway.Port <= 65535 {
			logger, _ := zap.NewDevelopment()
			gatewayConfig := &Config{
				Port:           config.Gateway.Port,
				GRPCPort:       config.Gateway.GRPCPort,
				MaxConnections: config.Gateway.MaxConnections,
				DataBufferSize: config.Gateway.DataBufferSize,
				UpdateInterval: config.Gateway.UpdateInterval,
				EnableMetrics:  config.Gateway.EnableMetrics,
				LogLevel:       config.Gateway.LogLevel,
			}

			// Create gateway instance to test config validation
			gateway := NewIndustrialGateway(gatewayConfig, logger)
			if gateway == nil {
				t.Error("Failed to create gateway with valid config")
			}
		}
	})
}

// FuzzDeviceConfiguration tests device configuration parsing for security issues
func FuzzDeviceConfiguration(f *testing.F) {
	// Add sample device configurations
	f.Add(`{"id": "plc-001", "protocol": "modbus", "address": "192.168.1.100", "port": 502}`)
	f.Add(`{"id": "hmi-001", "protocol": "opcua", "address": "opc.tcp://localhost:4840"}`)
	f.Add(`{"config": {"unit_id": 1, "timeout": 5000}}`)
	f.Add(`{}`)
	f.Add(`{"id": ""}`)

	f.Fuzz(func(t *testing.T, configJSON string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Device config parsing panicked with input %q: %v", configJSON, r)
			}
		}()

		var deviceConfig map[string]interface{}
		err := json.Unmarshal([]byte(configJSON), &deviceConfig)
		if err != nil {
			// Invalid JSON is expected
			return
		}

		// Create device from fuzzed config
		device := &Device{
			ID:       getString(deviceConfig, "id", "test-device"),
			Name:     getString(deviceConfig, "name", "Test Device"),
			Protocol: getString(deviceConfig, "protocol", "modbus"),
			Address:  getString(deviceConfig, "address", "localhost"),
			Port:     getInt(deviceConfig, "port", 502),
			Config:   getMap(deviceConfig, "config", make(map[string]interface{})),
		}

		// Validate device configuration doesn't contain dangerous values
		if device.Port < 0 || device.Port > 65535 {
			return // Invalid port
		}

		if len(device.ID) > 100 || len(device.Name) > 100 {
			return // Excessive string lengths
		}

		// Test device creation doesn't crash
		if device.ID != "" && device.Protocol != "" {
			// Device appears valid, should not cause issues
		}
	})
}

// Helper functions for fuzzing tests
func getString(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
		if i, ok := v.(int); ok {
			return i
		}
	}
	return defaultValue
}

func getMap(m map[string]interface{}, key string, defaultValue map[string]interface{}) map[string]interface{} {
	if v, ok := m[key]; ok {
		if subMap, ok := v.(map[string]interface{}); ok {
			return subMap
		}
	}
	return defaultValue
}