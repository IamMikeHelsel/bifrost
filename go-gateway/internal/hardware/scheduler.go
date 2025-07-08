package hardware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ScheduledTask represents a scheduled test task
type ScheduledTask struct {
	ID          string        `json:"id"`
	DeviceID    string        `json:"device_id"`
	Scenario    TestScenario  `json:"scenario"`
	Schedule    string        `json:"schedule"` // cron-like schedule
	NextRun     time.Time     `json:"next_run"`
	LastRun     *time.Time    `json:"last_run,omitempty"`
	Enabled     bool          `json:"enabled"`
	Priority    int           `json:"priority"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// TaskQueue represents a priority queue for test tasks
type TaskQueue struct {
	tasks    []*ScheduledTask
	mutex    sync.RWMutex
}

// HardwareTestScheduler manages scheduled testing of hardware devices
type HardwareTestScheduler struct {
	logger        *zap.Logger
	registry      *DeviceRegistry
	executor      *HardwareTestExecutor
	scenarios     map[string]TestScenario
	taskQueue     *TaskQueue
	scheduledTasks map[string]*ScheduledTask
	mutex         sync.RWMutex
	running       bool
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewHardwareTestScheduler creates a new test scheduler
func NewHardwareTestScheduler(logger *zap.Logger, registry *DeviceRegistry, executor *HardwareTestExecutor) *HardwareTestScheduler {
	return &HardwareTestScheduler{
		logger:         logger,
		registry:       registry,
		executor:       executor,
		scenarios:      make(map[string]TestScenario),
		taskQueue:      &TaskQueue{tasks: make([]*ScheduledTask, 0)},
		scheduledTasks: make(map[string]*ScheduledTask),
		stopCh:         make(chan struct{}),
	}
}

// LoadScenarios loads test scenarios from configuration
func (hts *HardwareTestScheduler) LoadScenarios(scenarios []TestScenario) {
	hts.mutex.Lock()
	defer hts.mutex.Unlock()

	for _, scenario := range scenarios {
		hts.scenarios[scenario.Name] = scenario
	}

	hts.logger.Info("Loaded test scenarios",
		zap.Int("scenario_count", len(scenarios)))
}

// AddTestScenario adds a new test scenario
func (hts *HardwareTestScheduler) AddTestScenario(scenario TestScenario) {
	hts.mutex.Lock()
	defer hts.mutex.Unlock()

	hts.scenarios[scenario.Name] = scenario
	hts.logger.Info("Added test scenario",
		zap.String("scenario", scenario.Name),
		zap.String("protocol", scenario.Protocol))
}

// ScheduleDeviceTests schedules tests for all devices based on their configuration
func (hts *HardwareTestScheduler) ScheduleDeviceTests() error {
	devices := hts.registry.GetAllDevices()

	for _, device := range devices {
		if !device.TestSchedule.Enabled {
			continue
		}

		for _, scenarioName := range device.TestSchedule.Scenarios {
			scenario, exists := hts.scenarios[scenarioName]
			if !exists {
				hts.logger.Warn("Scenario not found for device",
					zap.String("device_id", device.DeviceID),
					zap.String("scenario", scenarioName))
				continue
			}

			taskID := fmt.Sprintf("%s-%s", device.DeviceID, scenarioName)
			nextRun := hts.calculateNextRun(device.TestSchedule.Frequency)

			task := &ScheduledTask{
				ID:        taskID,
				DeviceID:  device.DeviceID,
				Scenario:  scenario,
				Schedule:  device.TestSchedule.Frequency,
				NextRun:   nextRun,
				Enabled:   true,
				Priority:  device.TestSchedule.Priority,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			hts.addScheduledTask(task)
		}
	}

	return nil
}

// Start starts the scheduler
func (hts *HardwareTestScheduler) Start() error {
	hts.mutex.Lock()
	if hts.running {
		hts.mutex.Unlock()
		return fmt.Errorf("scheduler is already running")
	}
	hts.running = true
	hts.mutex.Unlock()

	hts.logger.Info("Starting hardware test scheduler")

	// Schedule device tests
	if err := hts.ScheduleDeviceTests(); err != nil {
		return fmt.Errorf("failed to schedule device tests: %w", err)
	}

	// Start scheduler goroutine
	hts.wg.Add(1)
	go hts.run()

	return nil
}

// Stop stops the scheduler
func (hts *HardwareTestScheduler) Stop() error {
	hts.mutex.Lock()
	if !hts.running {
		hts.mutex.Unlock()
		return fmt.Errorf("scheduler is not running")
	}
	hts.running = false
	hts.mutex.Unlock()

	hts.logger.Info("Stopping hardware test scheduler")

	close(hts.stopCh)
	hts.wg.Wait()

	hts.logger.Info("Hardware test scheduler stopped")
	return nil
}

// run is the main scheduler loop
func (hts *HardwareTestScheduler) run() {
	defer hts.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-hts.stopCh:
			return
		case <-ticker.C:
			hts.processPendingTasks()
		}
	}
}

// processPendingTasks processes tasks that are due to run
func (hts *HardwareTestScheduler) processPendingTasks() {
	now := time.Now()
	tasks := hts.getTasksDue(now)

	for _, task := range tasks {
		// Check if device is available
		device, err := hts.registry.GetDevice(task.DeviceID)
		if err != nil {
			hts.logger.Error("Failed to get device for scheduled task",
				zap.String("task_id", task.ID),
				zap.String("device_id", task.DeviceID),
				zap.Error(err))
			continue
		}

		if device.Status != "available" {
			hts.logger.Debug("Device not available for scheduled task",
				zap.String("task_id", task.ID),
				zap.String("device_id", task.DeviceID),
				zap.String("status", device.Status))
			continue
		}

		// Execute the test
		go hts.executeScheduledTask(task)
	}
}

// executeScheduledTask executes a scheduled task
func (hts *HardwareTestScheduler) executeScheduledTask(task *ScheduledTask) {
	hts.logger.Info("Executing scheduled test",
		zap.String("task_id", task.ID),
		zap.String("device_id", task.DeviceID),
		zap.String("scenario", task.Scenario.Name))

	ctx, cancel := context.WithTimeout(context.Background(), task.Scenario.Timeout)
	defer cancel()

	execution, err := hts.executor.ExecuteTest(ctx, task.DeviceID, task.Scenario)
	if err != nil {
		hts.logger.Error("Failed to execute scheduled test",
			zap.String("task_id", task.ID),
			zap.Error(err))
		return
	}

	// Update task last run time and schedule next run
	now := time.Now()
	task.LastRun = &now
	task.NextRun = hts.calculateNextRun(task.Schedule)
	task.UpdatedAt = now

	hts.logger.Info("Scheduled test started",
		zap.String("task_id", task.ID),
		zap.String("execution_id", execution.ID),
		zap.Time("next_run", task.NextRun))
}

// getTasksDue returns tasks that are due to run
func (hts *HardwareTestScheduler) getTasksDue(now time.Time) []*ScheduledTask {
	hts.mutex.RLock()
	defer hts.mutex.RUnlock()

	var dueTasks []*ScheduledTask
	for _, task := range hts.scheduledTasks {
		if task.Enabled && now.After(task.NextRun) {
			dueTasks = append(dueTasks, task)
		}
	}

	return dueTasks
}

// addScheduledTask adds a task to the scheduler
func (hts *HardwareTestScheduler) addScheduledTask(task *ScheduledTask) {
	hts.mutex.Lock()
	defer hts.mutex.Unlock()

	hts.scheduledTasks[task.ID] = task
	hts.logger.Debug("Added scheduled task",
		zap.String("task_id", task.ID),
		zap.String("device_id", task.DeviceID),
		zap.Time("next_run", task.NextRun))
}

// calculateNextRun calculates the next run time based on frequency
func (hts *HardwareTestScheduler) calculateNextRun(frequency string) time.Time {
	now := time.Now()

	switch frequency {
	case "hourly":
		return now.Add(1 * time.Hour)
	case "daily":
		return now.Add(24 * time.Hour)
	case "weekly":
		return now.Add(7 * 24 * time.Hour)
	case "monthly":
		return now.AddDate(0, 1, 0)
	default:
		// Default to daily
		return now.Add(24 * time.Hour)
	}
}

// GetScheduledTasks returns all scheduled tasks
func (hts *HardwareTestScheduler) GetScheduledTasks() []*ScheduledTask {
	hts.mutex.RLock()
	defer hts.mutex.RUnlock()

	tasks := make([]*ScheduledTask, 0, len(hts.scheduledTasks))
	for _, task := range hts.scheduledTasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// GetTask returns a specific scheduled task
func (hts *HardwareTestScheduler) GetTask(taskID string) (*ScheduledTask, error) {
	hts.mutex.RLock()
	defer hts.mutex.RUnlock()

	task, exists := hts.scheduledTasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	return task, nil
}

// EnableTask enables a scheduled task
func (hts *HardwareTestScheduler) EnableTask(taskID string) error {
	hts.mutex.Lock()
	defer hts.mutex.Unlock()

	task, exists := hts.scheduledTasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Enabled = true
	task.UpdatedAt = time.Now()

	hts.logger.Info("Enabled scheduled task", zap.String("task_id", taskID))
	return nil
}

// DisableTask disables a scheduled task
func (hts *HardwareTestScheduler) DisableTask(taskID string) error {
	hts.mutex.Lock()
	defer hts.mutex.Unlock()

	task, exists := hts.scheduledTasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Enabled = false
	task.UpdatedAt = time.Now()

	hts.logger.Info("Disabled scheduled task", zap.String("task_id", taskID))
	return nil
}

// RemoveTask removes a scheduled task
func (hts *HardwareTestScheduler) RemoveTask(taskID string) error {
	hts.mutex.Lock()
	defer hts.mutex.Unlock()

	if _, exists := hts.scheduledTasks[taskID]; !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	delete(hts.scheduledTasks, taskID)

	hts.logger.Info("Removed scheduled task", zap.String("task_id", taskID))
	return nil
}

// RunTaskNow immediately executes a scheduled task
func (hts *HardwareTestScheduler) RunTaskNow(taskID string) (*TestExecution, error) {
	task, err := hts.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), task.Scenario.Timeout)
	defer cancel()

	execution, err := hts.executor.ExecuteTest(ctx, task.DeviceID, task.Scenario)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task: %w", err)
	}

	hts.logger.Info("Executed task on demand",
		zap.String("task_id", taskID),
		zap.String("execution_id", execution.ID))

	return execution, nil
}