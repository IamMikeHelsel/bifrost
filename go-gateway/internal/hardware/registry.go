package hardware

import (
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"bifrost-gateway/internal/protocols"
)

// HardwareDevice represents a real physical device in the test lab
type HardwareDevice struct {
	DeviceID     string            `yaml:"device_id" json:"device_id"`
	Manufacturer string            `yaml:"manufacturer" json:"manufacturer"`
	Model        string            `yaml:"model" json:"model"`
	Firmware     string            `yaml:"firmware" json:"firmware"`
	Protocols    []string          `yaml:"protocols" json:"protocols"`
	Network      NetworkConfig     `yaml:"network" json:"network"`
	TestSchedule TestScheduleConfig `yaml:"test_schedule" json:"test_schedule"`
	Metadata     map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	
	// Runtime state
	LastTested   time.Time `yaml:"-" json:"last_tested"`
	Status       string    `yaml:"-" json:"status"` // available, testing, offline, error
	TestResults  []TestResult `yaml:"-" json:"test_results,omitempty"`
}

// NetworkConfig defines the network configuration for a device
type NetworkConfig struct {
	IP     string `yaml:"ip" json:"ip"`
	Port   int    `yaml:"port,omitempty" json:"port,omitempty"`
	Subnet string `yaml:"subnet" json:"subnet"`
	VLAN   string `yaml:"vlan,omitempty" json:"vlan,omitempty"`
}

// TestScheduleConfig defines when and how to test a device
type TestScheduleConfig struct {
	Frequency string   `yaml:"frequency" json:"frequency"` // daily, weekly, monthly, on-demand
	Scenarios []string `yaml:"scenarios" json:"scenarios"`
	Enabled   bool     `yaml:"enabled" json:"enabled"`
	Priority  int      `yaml:"priority,omitempty" json:"priority,omitempty"`
}

// TestResult represents the result of a hardware test
type TestResult struct {
	TestID      string                 `json:"test_id"`
	DeviceID    string                 `json:"device_id"`
	Scenario    string                 `json:"scenario"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Duration    time.Duration          `json:"duration"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// HardwareTestLab represents the entire test lab configuration
type HardwareTestLab struct {
	Devices []HardwareDevice `yaml:"devices" json:"devices"`
	Config  LabConfig        `yaml:"config" json:"config"`
}

// LabConfig contains lab-wide configuration
type LabConfig struct {
	Name        string            `yaml:"name" json:"name"`
	Location    string            `yaml:"location" json:"location"`
	Network     NetworkLabConfig  `yaml:"network" json:"network"`
	Scheduling  SchedulingConfig  `yaml:"scheduling" json:"scheduling"`
	Reporting   ReportingConfig   `yaml:"reporting" json:"reporting"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// NetworkLabConfig defines lab network configuration
type NetworkLabConfig struct {
	BaseSubnet   string   `yaml:"base_subnet" json:"base_subnet"`
	VLANs        []string `yaml:"vlans" json:"vlans"`
	Gateway      string   `yaml:"gateway" json:"gateway"`
	DNSServers   []string `yaml:"dns_servers,omitempty" json:"dns_servers,omitempty"`
}

// SchedulingConfig defines test scheduling behavior
type SchedulingConfig struct {
	MaxConcurrentTests int    `yaml:"max_concurrent_tests" json:"max_concurrent_tests"`
	DefaultTimeout     string `yaml:"default_timeout" json:"default_timeout"`
	RetryAttempts      int    `yaml:"retry_attempts" json:"retry_attempts"`
	RetryDelay         string `yaml:"retry_delay" json:"retry_delay"`
}

// ReportingConfig defines result reporting behavior
type ReportingConfig struct {
	ResultRetention string   `yaml:"result_retention" json:"result_retention"`
	ExportFormats   []string `yaml:"export_formats" json:"export_formats"`
	NotificationURL string   `yaml:"notification_url,omitempty" json:"notification_url,omitempty"`
}

// DeviceRegistry manages the hardware device registry
type DeviceRegistry struct {
	logger    *zap.Logger
	mutex     sync.RWMutex
	devices   map[string]*HardwareDevice
	labConfig LabConfig
	configPath string
}

// NewDeviceRegistry creates a new device registry
func NewDeviceRegistry(logger *zap.Logger, configPath string) *DeviceRegistry {
	return &DeviceRegistry{
		logger:     logger,
		devices:    make(map[string]*HardwareDevice),
		configPath: configPath,
	}
}

// LoadConfiguration loads the hardware test lab configuration from YAML
func (dr *DeviceRegistry) LoadConfiguration() error {
	dr.mutex.Lock()
	defer dr.mutex.Unlock()

	data, err := ioutil.ReadFile(dr.configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	var lab HardwareTestLab
	if err := yaml.Unmarshal(data, &lab); err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Clear existing devices and reload
	dr.devices = make(map[string]*HardwareDevice)
	for i := range lab.Devices {
		device := &lab.Devices[i]
		device.Status = "available"
		dr.devices[device.DeviceID] = device
	}

	dr.labConfig = lab.Config
	dr.logger.Info("Loaded hardware test lab configuration",
		zap.Int("device_count", len(dr.devices)),
		zap.String("lab_name", dr.labConfig.Name))

	return nil
}

// SaveConfiguration saves the current configuration to YAML
func (dr *DeviceRegistry) SaveConfiguration() error {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()

	devices := make([]HardwareDevice, 0, len(dr.devices))
	for _, device := range dr.devices {
		devices = append(devices, *device)
	}

	lab := HardwareTestLab{
		Devices: devices,
		Config:  dr.labConfig,
	}

	data, err := yaml.Marshal(&lab)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if err := ioutil.WriteFile(dr.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// GetDevice retrieves a device by ID
func (dr *DeviceRegistry) GetDevice(deviceID string) (*HardwareDevice, error) {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()

	device, exists := dr.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device %s not found", deviceID)
	}

	return device, nil
}

// GetAllDevices returns all registered devices
func (dr *DeviceRegistry) GetAllDevices() []*HardwareDevice {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()

	devices := make([]*HardwareDevice, 0, len(dr.devices))
	for _, device := range dr.devices {
		devices = append(devices, device)
	}

	return devices
}

// GetDevicesByProtocol returns devices that support a specific protocol
func (dr *DeviceRegistry) GetDevicesByProtocol(protocol string) []*HardwareDevice {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()

	var devices []*HardwareDevice
	for _, device := range dr.devices {
		for _, p := range device.Protocols {
			if p == protocol {
				devices = append(devices, device)
				break
			}
		}
	}

	return devices
}

// GetAvailableDevices returns devices that are currently available for testing
func (dr *DeviceRegistry) GetAvailableDevices() []*HardwareDevice {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()

	var devices []*HardwareDevice
	for _, device := range dr.devices {
		if device.Status == "available" {
			devices = append(devices, device)
		}
	}

	return devices
}

// UpdateDeviceStatus updates the status of a device
func (dr *DeviceRegistry) UpdateDeviceStatus(deviceID, status string) error {
	dr.mutex.Lock()
	defer dr.mutex.Unlock()

	device, exists := dr.devices[deviceID]
	if !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	device.Status = status
	dr.logger.Debug("Updated device status",
		zap.String("device_id", deviceID),
		zap.String("status", status))

	return nil
}

// AddTestResult adds a test result to a device
func (dr *DeviceRegistry) AddTestResult(deviceID string, result TestResult) error {
	dr.mutex.Lock()
	defer dr.mutex.Unlock()

	device, exists := dr.devices[deviceID]
	if !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	device.TestResults = append(device.TestResults, result)
	device.LastTested = result.EndTime

	// Limit result history to prevent memory issues
	maxResults := 100
	if len(device.TestResults) > maxResults {
		device.TestResults = device.TestResults[len(device.TestResults)-maxResults:]
	}

	return nil
}

// ConvertToProtocolDevice converts a HardwareDevice to a protocols.Device
func (hd *HardwareDevice) ConvertToProtocolDevice() *protocols.Device {
	return &protocols.Device{
		ID:       hd.DeviceID,
		Name:     fmt.Sprintf("%s %s", hd.Manufacturer, hd.Model),
		Protocol: hd.Protocols[0], // Use first protocol as default
		Address:  hd.Network.IP,
		Port:     hd.Network.Port,
		Config: map[string]interface{}{
			"manufacturer": hd.Manufacturer,
			"model":        hd.Model,
			"firmware":     hd.Firmware,
			"subnet":       hd.Network.Subnet,
			"protocols":    hd.Protocols,
		},
	}
}