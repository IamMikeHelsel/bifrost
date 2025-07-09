package hardware

import (
	"fmt"
	"io/ioutil"
	"sync"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"bifrost-gateway/internal/protocols"
)

// TestScenariosConfig represents the test scenarios configuration file
type TestScenariosConfig struct {
	Scenarios []TestScenario `yaml:"scenarios"`
}

// HardwareTestManager manages the entire hardware testing framework
type HardwareTestManager struct {
	logger    *zap.Logger
	registry  *DeviceRegistry
	executor  *HardwareTestExecutor
	scheduler *HardwareTestScheduler
	mutex     sync.RWMutex
	running   bool
}

// NewHardwareTestManager creates a new hardware test manager
func NewHardwareTestManager(logger *zap.Logger, configPath string) *HardwareTestManager {
	registry := NewDeviceRegistry(logger, configPath)
	executor := NewHardwareTestExecutor(logger, registry, 5) // Max 5 concurrent tests
	scheduler := NewHardwareTestScheduler(logger, registry, executor)

	return &HardwareTestManager{
		logger:    logger,
		registry:  registry,
		executor:  executor,
		scheduler: scheduler,
	}
}

// Initialize initializes the hardware test manager
func (htm *HardwareTestManager) Initialize() error {
	htm.mutex.Lock()
	defer htm.mutex.Unlock()

	// Load device registry configuration
	if err := htm.registry.LoadConfiguration(); err != nil {
		return fmt.Errorf("failed to load device registry: %w", err)
	}

	htm.logger.Info("Hardware test manager initialized successfully")
	return nil
}

// LoadTestScenarios loads test scenarios from a configuration file
func (htm *HardwareTestManager) LoadTestScenarios(scenariosPath string) error {
	data, err := ioutil.ReadFile(scenariosPath)
	if err != nil {
		return fmt.Errorf("failed to read scenarios file: %w", err)
	}

	var config TestScenariosConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse scenarios: %w", err)
	}

	htm.scheduler.LoadScenarios(config.Scenarios)
	
	htm.logger.Info("Loaded test scenarios",
		zap.Int("scenario_count", len(config.Scenarios)),
		zap.String("file", scenariosPath))

	return nil
}

// RegisterProtocolHandler registers a protocol handler for testing
func (htm *HardwareTestManager) RegisterProtocolHandler(protocol string, handler protocols.ProtocolHandler) {
	htm.executor.RegisterProtocolHandler(protocol, handler)
}

// Start starts the hardware testing framework
func (htm *HardwareTestManager) Start() error {
	htm.mutex.Lock()
	defer htm.mutex.Unlock()

	if htm.running {
		return fmt.Errorf("hardware test manager is already running")
	}

	// Start the scheduler
	if err := htm.scheduler.Start(); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	htm.running = true
	htm.logger.Info("Hardware test manager started")
	return nil
}

// Stop stops the hardware testing framework
func (htm *HardwareTestManager) Stop() error {
	htm.mutex.Lock()
	defer htm.mutex.Unlock()

	if !htm.running {
		return fmt.Errorf("hardware test manager is not running")
	}

	// Stop the scheduler
	if err := htm.scheduler.Stop(); err != nil {
		return fmt.Errorf("failed to stop scheduler: %w", err)
	}

	htm.running = false
	htm.logger.Info("Hardware test manager stopped")
	return nil
}

// GetRegistry returns the device registry
func (htm *HardwareTestManager) GetRegistry() *DeviceRegistry {
	return htm.registry
}

// GetExecutor returns the test executor
func (htm *HardwareTestManager) GetExecutor() *HardwareTestExecutor {
	return htm.executor
}

// GetScheduler returns the test scheduler
func (htm *HardwareTestManager) GetScheduler() *HardwareTestScheduler {
	return htm.scheduler
}

// GetStatus returns the current status of the hardware test manager
func (htm *HardwareTestManager) GetStatus() map[string]interface{} {
	htm.mutex.RLock()
	defer htm.mutex.RUnlock()

	devices := htm.registry.GetAllDevices()
	activeExecutions := htm.executor.GetActiveExecutions()
	scheduledTasks := htm.scheduler.GetScheduledTasks()

	deviceStats := make(map[string]int)
	for _, device := range devices {
		deviceStats[device.Status]++
	}

	taskStats := make(map[string]int)
	for _, task := range scheduledTasks {
		if task.Enabled {
			taskStats["enabled"]++
		} else {
			taskStats["disabled"]++
		}
	}

	return map[string]interface{}{
		"running":            htm.running,
		"total_devices":      len(devices),
		"device_status":      deviceStats,
		"active_executions":  len(activeExecutions),
		"scheduled_tasks":    len(scheduledTasks),
		"task_stats":         taskStats,
	}
}