package hardware

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"bifrost-gateway/internal/protocols"
)

// MockProtocolHandler implements ProtocolHandler for testing
type MockProtocolHandler struct {
	connected bool
	pingError error
}

func (m *MockProtocolHandler) Connect(device *protocols.Device) error {
	m.connected = true
	return nil
}

func (m *MockProtocolHandler) Disconnect(device *protocols.Device) error {
	m.connected = false
	return nil
}

func (m *MockProtocolHandler) IsConnected(device *protocols.Device) bool {
	return m.connected
}

func (m *MockProtocolHandler) ReadTag(device *protocols.Device, tag *protocols.Tag) (interface{}, error) {
	if tag.Address == "40001" {
		return 100, nil
	}
	if tag.Address == "40010" {
		return 12345, nil
	}
	return 0, nil
}

func (m *MockProtocolHandler) WriteTag(device *protocols.Device, tag *protocols.Tag, value interface{}) error {
	return nil
}

func (m *MockProtocolHandler) ReadMultipleTags(device *protocols.Device, tags []*protocols.Tag) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for _, tag := range tags {
		val, _ := m.ReadTag(device, tag)
		result[tag.Name] = val
	}
	return result, nil
}

func (m *MockProtocolHandler) DiscoverDevices(ctx context.Context, networkRange string) ([]*protocols.Device, error) {
	return []*protocols.Device{}, nil
}

func (m *MockProtocolHandler) GetDeviceInfo(device *protocols.Device) (*protocols.DeviceInfo, error) {
	return &protocols.DeviceInfo{
		Vendor:          "Mock Manufacturer",
		Model:           "Mock Model",
		SerialNumber:    "12345",
		FirmwareVersion: "1.0.0",
		Capabilities:    []string{"read", "write"},
		MaxConnections:  10,
		SupportedRates:  []int{1000, 2000},
		CustomInfo:      map[string]string{"type": "mock"},
	}, nil
}

func (m *MockProtocolHandler) GetSupportedDataTypes() []string {
	return []string{"INT", "REAL", "BOOL"}
}

func (m *MockProtocolHandler) ValidateTagAddress(address string) error {
	return nil
}

func (m *MockProtocolHandler) Ping(device *protocols.Device) error {
	return m.pingError
}

func (m *MockProtocolHandler) GetDiagnostics(device *protocols.Device) (*protocols.Diagnostics, error) {
	return &protocols.Diagnostics{
		IsHealthy:         true,
		LastCommunication: time.Now(),
		ResponseTime:      50 * time.Millisecond,
		ErrorCount:        0,
		SuccessRate:       100.0,
		ConnectionUptime:  time.Hour,
		Errors:            []protocols.DiagnosticError{},
	}, nil
}

func createTestConfig(t *testing.T) string {
	config := `config:
  name: "Test Lab"
  location: "Test Location"
  network:
    base_subnet: "192.168.1.0/24"
    vlans: ["test_vlan"]
    gateway: "192.168.1.1"
  scheduling:
    max_concurrent_tests: 2
    default_timeout: "30s"
    retry_attempts: 1
    retry_delay: "1s"
  reporting:
    result_retention: "1h"
    export_formats: ["json"]

devices:
  - device_id: "test_device_001"
    manufacturer: "Test Manufacturer"
    model: "Test Model"
    firmware: "1.0.0"
    protocols: ["modbus_tcp"]
    network:
      ip: "192.168.1.100"
      port: 502
      subnet: "test_vlan"
    test_schedule:
      frequency: "daily"
      scenarios: ["basic_io"]
      enabled: true
      priority: 1
    metadata:
      location: "Test Rack"
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)
	
	return configPath
}

func createTestScenarios(t *testing.T) string {
	scenarios := `scenarios:
  - name: "basic_io"
    description: "Basic I/O test"
    protocol: "modbus_tcp"
    timeout: "1m"
    steps:
      - name: "Connect"
        type: "connect"
        timeout: "10s"
      - name: "Ping"
        type: "ping"
        timeout: "5s"
      - name: "Read register"
        type: "read"
        address: "40001"
        timeout: "5s"
      - name: "Write register"
        type: "write"
        address: "40010"
        value: 12345
        timeout: "5s"
      - name: "Read back"
        type: "read"
        address: "40010"
        expected: 12345
        timeout: "5s"
      - name: "Disconnect"
        type: "disconnect"
        timeout: "5s"
`

	tmpDir := t.TempDir()
	scenariosPath := filepath.Join(tmpDir, "test_scenarios.yaml")
	err := os.WriteFile(scenariosPath, []byte(scenarios), 0644)
	require.NoError(t, err)
	
	return scenariosPath
}

func TestDeviceRegistry(t *testing.T) {
	logger := zap.NewNop()
	configPath := createTestConfig(t)
	
	registry := NewDeviceRegistry(logger, configPath)
	
	// Test loading configuration
	err := registry.LoadConfiguration()
	assert.NoError(t, err)
	
	// Test getting all devices
	devices := registry.GetAllDevices()
	assert.Len(t, devices, 1)
	assert.Equal(t, "test_device_001", devices[0].DeviceID)
	
	// Test getting device by ID
	device, err := registry.GetDevice("test_device_001")
	assert.NoError(t, err)
	assert.Equal(t, "Test Manufacturer", device.Manufacturer)
	
	// Test getting non-existent device
	_, err = registry.GetDevice("non_existent")
	assert.Error(t, err)
	
	// Test updating device status
	err = registry.UpdateDeviceStatus("test_device_001", "testing")
	assert.NoError(t, err)
	
	device, _ = registry.GetDevice("test_device_001")
	assert.Equal(t, "testing", device.Status)
	
	// Test getting devices by protocol
	modbusDevices := registry.GetDevicesByProtocol("modbus_tcp")
	assert.Len(t, modbusDevices, 1)
	
	ethernetDevices := registry.GetDevicesByProtocol("ethernet_ip")
	assert.Len(t, ethernetDevices, 0)
}

func TestHardwareTestExecutor(t *testing.T) {
	logger := zap.NewNop()
	configPath := createTestConfig(t)
	
	registry := NewDeviceRegistry(logger, configPath)
	err := registry.LoadConfiguration()
	require.NoError(t, err)
	
	executor := NewHardwareTestExecutor(logger, registry, 2)
	
	// Register mock protocol handler
	mockHandler := &MockProtocolHandler{}
	executor.RegisterProtocolHandler("modbus_tcp", mockHandler)
	
	// Create test scenario
	scenario := TestScenario{
		Name:        "test_scenario",
		Description: "Test scenario",
		Protocol:    "modbus_tcp",
		Timeout:     30 * time.Second,
		Steps: []TestStep{
			{Name: "Connect", Type: "connect", Timeout: 10 * time.Second},
			{Name: "Ping", Type: "ping", Timeout: 5 * time.Second},
			{Name: "Read", Type: "read", Address: "40001", Timeout: 5 * time.Second},
			{Name: "Disconnect", Type: "disconnect", Timeout: 5 * time.Second},
		},
	}
	
	// Execute test
	ctx := context.Background()
	execution, err := executor.ExecuteTest(ctx, "test_device_001", scenario)
	assert.NoError(t, err)
	assert.NotNil(t, execution)
	assert.Equal(t, "test_device_001", execution.DeviceID)
	assert.Equal(t, "running", execution.Status)
	
	// Wait a bit for test to complete
	time.Sleep(100 * time.Millisecond)
	
	// Check execution status
	execution, err = executor.GetExecution(execution.ID)
	assert.NoError(t, err)
	// Status might still be running due to async execution
	assert.Contains(t, []string{"running", "completed"}, execution.Status)
}

func TestHardwareTestScheduler(t *testing.T) {
	logger := zap.NewNop()
	configPath := createTestConfig(t)
	
	registry := NewDeviceRegistry(logger, configPath)
	err := registry.LoadConfiguration()
	require.NoError(t, err)
	
	executor := NewHardwareTestExecutor(logger, registry, 2)
	scheduler := NewHardwareTestScheduler(logger, registry, executor)
	
	// Add test scenario
	scenario := TestScenario{
		Name:        "basic_io",
		Description: "Basic I/O test",
		Protocol:    "modbus_tcp",
		Timeout:     30 * time.Second,
		Steps: []TestStep{
			{Name: "Connect", Type: "connect"},
			{Name: "Ping", Type: "ping"},
		},
	}
	
	scheduler.AddTestScenario(scenario)
	
	// Schedule device tests
	err = scheduler.ScheduleDeviceTests()
	assert.NoError(t, err)
	
	// Get scheduled tasks
	tasks := scheduler.GetScheduledTasks()
	assert.Len(t, tasks, 1)
	assert.Equal(t, "test_device_001-basic_io", tasks[0].ID)
	assert.True(t, tasks[0].Enabled)
}

func TestHardwareTestManager(t *testing.T) {
	logger := zap.NewNop()
	configPath := createTestConfig(t)
	scenariosPath := createTestScenarios(t)
	
	manager := NewHardwareTestManager(logger, configPath)
	
	// Initialize manager
	err := manager.Initialize()
	assert.NoError(t, err)
	
	// Load test scenarios
	err = manager.LoadTestScenarios(scenariosPath)
	assert.NoError(t, err)
	
	// Register mock protocol handler
	mockHandler := &MockProtocolHandler{}
	manager.RegisterProtocolHandler("modbus_tcp", mockHandler)
	
	// Get status before starting
	status := manager.GetStatus()
	assert.False(t, status["running"].(bool))
	assert.Equal(t, 1, status["total_devices"].(int))
	
	// Start manager
	err = manager.Start()
	assert.NoError(t, err)
	
	// Get status after starting
	status = manager.GetStatus()
	assert.True(t, status["running"].(bool))
	
	// Stop manager
	err = manager.Stop()
	assert.NoError(t, err)
	
	// Get status after stopping
	status = manager.GetStatus()
	assert.False(t, status["running"].(bool))
}

func TestHardwareDeviceConversion(t *testing.T) {
	device := &HardwareDevice{
		DeviceID:     "test_001",
		Manufacturer: "Test Manufacturer",
		Model:        "Test Model",
		Firmware:     "1.0.0",
		Protocols:    []string{"modbus_tcp", "ethernet_ip"},
		Network: NetworkConfig{
			IP:     "192.168.1.100",
			Port:   502,
			Subnet: "test_subnet",
		},
	}
	
	protocolDevice := device.ConvertToProtocolDevice()
	
	assert.Equal(t, "test_001", protocolDevice.ID)
	assert.Equal(t, "Test Manufacturer Test Model", protocolDevice.Name)
	assert.Equal(t, "modbus_tcp", protocolDevice.Protocol)
	assert.Equal(t, "192.168.1.100", protocolDevice.Address)
	assert.Equal(t, 502, protocolDevice.Port)
	assert.Equal(t, "Test Manufacturer", protocolDevice.Config["manufacturer"])
}

func TestTestStepExecution(t *testing.T) {
	logger := zap.NewNop()
	configPath := createTestConfig(t)
	
	registry := NewDeviceRegistry(logger, configPath)
	err := registry.LoadConfiguration()
	require.NoError(t, err)
	
	executor := NewHardwareTestExecutor(logger, registry, 1)
	mockHandler := &MockProtocolHandler{}
	
	// Test successful ping
	device := &protocols.Device{ID: "test", Address: "192.168.1.100", Port: 502}
	step := TestStep{Name: "Ping", Type: "ping", Timeout: 5 * time.Second}
	connected := false
	
	result := executor.executeTestStep(context.Background(), mockHandler, device, step, &connected)
	assert.True(t, result.Success)
	assert.Empty(t, result.Error)
	
	// Test failed ping
	mockHandler.pingError = assert.AnError
	result = executor.executeTestStep(context.Background(), mockHandler, device, step, &connected)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Error)
}