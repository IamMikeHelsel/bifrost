package hardware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"bifrost-gateway/internal/protocols"
)

// TestScenario defines a specific test scenario
type TestScenario struct {
	Name        string                 `yaml:"name" json:"name"`
	Description string                 `yaml:"description" json:"description"`
	Protocol    string                 `yaml:"protocol" json:"protocol"`
	Steps       []TestStep             `yaml:"steps" json:"steps"`
	Timeout     time.Duration          `yaml:"timeout" json:"timeout"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// TestStep represents a single step in a test scenario
type TestStep struct {
	Name        string                 `yaml:"name" json:"name"`
	Type        string                 `yaml:"type" json:"type"` // connect, read, write, ping, disconnect
	Address     string                 `yaml:"address,omitempty" json:"address,omitempty"`
	Value       interface{}            `yaml:"value,omitempty" json:"value,omitempty"`
	Expected    interface{}            `yaml:"expected,omitempty" json:"expected,omitempty"`
	Timeout     time.Duration          `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	RetryCount  int                    `yaml:"retry_count,omitempty" json:"retry_count,omitempty"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// TestExecution represents an ongoing test execution
type TestExecution struct {
	ID          string                 `json:"id"`
	DeviceID    string                 `json:"device_id"`
	Scenario    TestScenario           `json:"scenario"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Status      string                 `json:"status"` // running, completed, failed, cancelled
	CurrentStep int                    `json:"current_step"`
	Steps       []TestStepResult       `json:"steps"`
	Metrics     map[string]interface{} `json:"metrics"`
	Error       string                 `json:"error,omitempty"`
}

// TestStepResult represents the result of a test step
type TestStepResult struct {
	Step        TestStep               `json:"step"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Duration    time.Duration          `json:"duration"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	ActualValue interface{}            `json:"actual_value,omitempty"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
}

// HardwareTestExecutor manages the execution of hardware tests
type HardwareTestExecutor struct {
	logger          *zap.Logger
	registry        *DeviceRegistry
	protocolHandlers map[string]protocols.ProtocolHandler
	mutex           sync.RWMutex
	executions      map[string]*TestExecution
	maxConcurrent   int
	running         int
}

// NewHardwareTestExecutor creates a new test executor
func NewHardwareTestExecutor(logger *zap.Logger, registry *DeviceRegistry, maxConcurrent int) *HardwareTestExecutor {
	return &HardwareTestExecutor{
		logger:           logger,
		registry:         registry,
		protocolHandlers: make(map[string]protocols.ProtocolHandler),
		executions:       make(map[string]*TestExecution),
		maxConcurrent:    maxConcurrent,
	}
}

// RegisterProtocolHandler registers a protocol handler for testing
func (hte *HardwareTestExecutor) RegisterProtocolHandler(protocol string, handler protocols.ProtocolHandler) {
	hte.mutex.Lock()
	defer hte.mutex.Unlock()
	
	hte.protocolHandlers[protocol] = handler
	hte.logger.Info("Registered protocol handler for hardware testing",
		zap.String("protocol", protocol))
}

// ExecuteTest executes a test scenario against a hardware device
func (hte *HardwareTestExecutor) ExecuteTest(ctx context.Context, deviceID string, scenario TestScenario) (*TestExecution, error) {
	hte.mutex.Lock()
	if hte.running >= hte.maxConcurrent {
		hte.mutex.Unlock()
		return nil, fmt.Errorf("maximum concurrent tests (%d) reached", hte.maxConcurrent)
	}
	hte.running++
	hte.mutex.Unlock()

	defer func() {
		hte.mutex.Lock()
		hte.running--
		hte.mutex.Unlock()
	}()

	// Get device from registry
	device, err := hte.registry.GetDevice(deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Check if device is available
	if device.Status != "available" {
		return nil, fmt.Errorf("device %s is not available (status: %s)", deviceID, device.Status)
	}

	// Update device status to testing
	if err := hte.registry.UpdateDeviceStatus(deviceID, "testing"); err != nil {
		return nil, fmt.Errorf("failed to update device status: %w", err)
	}

	// Create test execution
	execution := &TestExecution{
		ID:          generateTestID(),
		DeviceID:    deviceID,
		Scenario:    scenario,
		StartTime:   time.Now(),
		Status:      "running",
		CurrentStep: 0,
		Steps:       make([]TestStepResult, 0, len(scenario.Steps)),
		Metrics:     make(map[string]interface{}),
	}

	// Store execution
	hte.mutex.Lock()
	hte.executions[execution.ID] = execution
	hte.mutex.Unlock()

	// Execute the test
	go hte.executeTestSteps(ctx, execution, device)

	return execution, nil
}

// executeTestSteps executes all test steps
func (hte *HardwareTestExecutor) executeTestSteps(ctx context.Context, execution *TestExecution, device *HardwareDevice) {
	defer func() {
		// Restore device status
		hte.registry.UpdateDeviceStatus(device.DeviceID, "available")
		
		// Update execution status
		hte.mutex.Lock()
		now := time.Now()
		execution.EndTime = &now
		if execution.Status == "running" {
			execution.Status = "completed"
		}
		hte.mutex.Unlock()

		// Add result to device registry
		result := TestResult{
			TestID:    execution.ID,
			DeviceID:  execution.DeviceID,
			Scenario:  execution.Scenario.Name,
			StartTime: execution.StartTime,
			EndTime:   now,
			Duration:  now.Sub(execution.StartTime),
			Success:   execution.Status == "completed",
			Metrics:   execution.Metrics,
		}
		if execution.Error != "" {
			result.Error = execution.Error
		}

		hte.registry.AddTestResult(device.DeviceID, result)
	}()

	// Get protocol handler
	handler, exists := hte.protocolHandlers[execution.Scenario.Protocol]
	if !exists {
		hte.updateExecutionError(execution, fmt.Sprintf("no handler for protocol %s", execution.Scenario.Protocol))
		return
	}

	// Convert to protocol device
	protocolDevice := device.ConvertToProtocolDevice()
	protocolDevice.Protocol = execution.Scenario.Protocol

	// Execute each step
	var connected bool
	for i, step := range execution.Scenario.Steps {
		execution.CurrentStep = i

		stepResult := hte.executeTestStep(ctx, handler, protocolDevice, step, &connected)
		
		hte.mutex.Lock()
		execution.Steps = append(execution.Steps, stepResult)
		hte.mutex.Unlock()

		if !stepResult.Success {
			hte.updateExecutionError(execution, stepResult.Error)
			break
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			hte.updateExecutionError(execution, "test cancelled")
			return
		default:
		}
	}

	// Ensure disconnection
	if connected {
		handler.Disconnect(protocolDevice)
	}
}

// executeTestStep executes a single test step
func (hte *HardwareTestExecutor) executeTestStep(ctx context.Context, handler protocols.ProtocolHandler, device *protocols.Device, step TestStep, connected *bool) TestStepResult {
	startTime := time.Now()
	result := TestStepResult{
		Step:      step,
		StartTime: startTime,
		Metrics:   make(map[string]interface{}),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	// Set timeout context if needed
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	var err error
	
	switch step.Type {
	case "connect":
		err = handler.Connect(device)
		if err == nil {
			*connected = true
		}

	case "disconnect":
		err = handler.Disconnect(device)
		if err == nil {
			*connected = false
		}

	case "ping":
		err = handler.Ping(device)

	case "read":
		if step.Address == "" {
			err = fmt.Errorf("address required for read operation")
			break
		}
		
		tag := &protocols.Tag{
			Name:    step.Address,
			Address: step.Address,
		}
		
		value, readErr := handler.ReadTag(device, tag)
		if readErr != nil {
			err = readErr
		} else {
			result.ActualValue = value
			// Check expected value if provided
			if step.Expected != nil && value != step.Expected {
				err = fmt.Errorf("expected %v, got %v", step.Expected, value)
			}
		}

	case "write":
		if step.Address == "" {
			err = fmt.Errorf("address required for write operation")
			break
		}
		if step.Value == nil {
			err = fmt.Errorf("value required for write operation")
			break
		}
		
		tag := &protocols.Tag{
			Name:    step.Address,
			Address: step.Address,
		}
		
		err = handler.WriteTag(device, tag, step.Value)

	case "device_info":
		info, infoErr := handler.GetDeviceInfo(device)
		if infoErr != nil {
			err = infoErr
		} else {
			result.ActualValue = info
		}

	case "diagnostics":
		diagnostics, diagErr := handler.GetDiagnostics(device)
		if diagErr != nil {
			err = diagErr
		} else {
			result.ActualValue = diagnostics
		}

	default:
		err = fmt.Errorf("unknown test step type: %s", step.Type)
	}

	// Handle retry logic
	retryCount := step.RetryCount
	for err != nil && retryCount > 0 {
		hte.logger.Debug("Retrying test step",
			zap.String("step", step.Name),
			zap.Int("retries_left", retryCount),
			zap.Error(err))
		
		time.Sleep(100 * time.Millisecond) // Brief delay between retries
		retryCount--
		
		// Retry the operation (simplified - would need full retry logic)
		switch step.Type {
		case "ping":
			err = handler.Ping(device)
		case "read":
			tag := &protocols.Tag{Name: step.Address, Address: step.Address}
			value, readErr := handler.ReadTag(device, tag)
			if readErr != nil {
				err = readErr
			} else {
				result.ActualValue = value
				err = nil
			}
		}
	}

	result.Success = err == nil
	if err != nil {
		result.Error = err.Error()
	}

	// Collect performance metrics
	result.Metrics["execution_time_ms"] = result.Duration.Milliseconds()
	
	return result
}

// updateExecutionError updates an execution with an error
func (hte *HardwareTestExecutor) updateExecutionError(execution *TestExecution, errMsg string) {
	hte.mutex.Lock()
	defer hte.mutex.Unlock()
	
	execution.Status = "failed"
	execution.Error = errMsg
}

// GetExecution retrieves a test execution by ID
func (hte *HardwareTestExecutor) GetExecution(executionID string) (*TestExecution, error) {
	hte.mutex.RLock()
	defer hte.mutex.RUnlock()
	
	execution, exists := hte.executions[executionID]
	if !exists {
		return nil, fmt.Errorf("execution %s not found", executionID)
	}
	
	return execution, nil
}

// GetActiveExecutions returns all currently running executions
func (hte *HardwareTestExecutor) GetActiveExecutions() []*TestExecution {
	hte.mutex.RLock()
	defer hte.mutex.RUnlock()
	
	var active []*TestExecution
	for _, execution := range hte.executions {
		if execution.Status == "running" {
			active = append(active, execution)
		}
	}
	
	return active
}

// CancelExecution cancels a running test execution
func (hte *HardwareTestExecutor) CancelExecution(executionID string) error {
	hte.mutex.Lock()
	defer hte.mutex.Unlock()
	
	execution, exists := hte.executions[executionID]
	if !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}
	
	if execution.Status != "running" {
		return fmt.Errorf("execution %s is not running", executionID)
	}
	
	execution.Status = "cancelled"
	return nil
}

// generateTestID generates a unique test ID
func generateTestID() string {
	return fmt.Sprintf("hw-test-%d", time.Now().UnixNano())
}