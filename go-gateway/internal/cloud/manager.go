package cloud

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager manages multiple cloud connectors
type Manager struct {
	logger     *zap.Logger
	connectors map[string]CloudConnector
	config     *ManagerConfig
	mutex      sync.RWMutex
	
	// Data routing
	buffer      Buffer
	batchBuffer []*CloudData
	batchMutex  sync.Mutex
	batchTimer  *time.Timer
	
	// Health monitoring
	healthChecker *HealthChecker
	
	// Graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ManagerConfig holds configuration for the cloud manager
type ManagerConfig struct {
	DefaultConnector  string                           `yaml:"default_connector"`
	BatchSize         int                              `yaml:"batch_size"`
	BatchTimeout      time.Duration                    `yaml:"batch_timeout"`
	HealthCheckInterval time.Duration                  `yaml:"health_check_interval"`
	BufferConfig      *BufferConfig                    `yaml:"buffer"`
	Connectors        map[string]*ConnectorConfig      `yaml:"connectors"`
	RoutingRules      []*RoutingRule                   `yaml:"routing_rules"`
}

// RoutingRule defines how data should be routed to connectors
type RoutingRule struct {
	Name        string            `yaml:"name"`
	Condition   string            `yaml:"condition"`  // Simple condition like "device_id=PLC001" or "tag_name=temperature"
	Connectors  []string          `yaml:"connectors"` // List of connector names
	Priority    int               `yaml:"priority"`   // Higher priority rules are evaluated first
	Transform   map[string]string `yaml:"transform"`  // Optional data transformation
}

// DefaultManagerConfig returns a default manager configuration
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		BatchSize:           100,
		BatchTimeout:        5 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		BufferConfig: &BufferConfig{
			MaxSize:        10000,
			FlushInterval:  10 * time.Second,
			PersistentPath: "/tmp/bifrost-cloud-buffer",
		},
		Connectors:   make(map[string]*ConnectorConfig),
		RoutingRules: make([]*RoutingRule, 0),
	}
}

// NewManager creates a new cloud connector manager
func NewManager(logger *zap.Logger, config *ManagerConfig) (*Manager, error) {
	if config == nil {
		config = DefaultManagerConfig()
	}
	
	// Create buffer
	var buffer Buffer
	var err error
	if config.BufferConfig != nil {
		buffer, err = NewDiskBuffer(logger, config.BufferConfig)
		if err != nil {
			logger.Warn("Failed to create disk buffer, using memory buffer", zap.Error(err))
			buffer = NewMemoryBuffer(logger, config.BufferConfig)
		}
	} else {
		buffer = NewMemoryBuffer(logger, DefaultManagerConfig().BufferConfig)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &Manager{
		logger:      logger,
		connectors:  make(map[string]CloudConnector),
		config:      config,
		buffer:      buffer,
		batchBuffer: make([]*CloudData, 0, config.BatchSize),
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// Initialize health checker
	manager.healthChecker = NewHealthChecker(logger, config.HealthCheckInterval)
	
	// Start batch processing
	manager.startBatchProcessing()
	
	return manager, nil
}

// RegisterConnector registers a cloud connector
func (m *Manager) RegisterConnector(name string, connector CloudConnector) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if _, exists := m.connectors[name]; exists {
		return fmt.Errorf("connector %s already registered", name)
	}
	
	// Validate connector configuration
	if err := connector.ValidateConfig(); err != nil {
		return fmt.Errorf("connector %s validation failed: %w", name, err)
	}
	
	m.connectors[name] = connector
	m.healthChecker.AddConnector(name, connector)
	
	m.logger.Info("Registered cloud connector", zap.String("name", name))
	
	return nil
}

// UnregisterConnector removes a cloud connector
func (m *Manager) UnregisterConnector(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	connector, exists := m.connectors[name]
	if !exists {
		return fmt.Errorf("connector %s not found", name)
	}
	
	// Disconnect if connected
	if connector.IsConnected() {
		if err := connector.Disconnect(m.ctx); err != nil {
			m.logger.Warn("Error disconnecting connector", zap.String("name", name), zap.Error(err))
		}
	}
	
	delete(m.connectors, name)
	m.healthChecker.RemoveConnector(name)
	
	m.logger.Info("Unregistered cloud connector", zap.String("name", name))
	
	return nil
}

// GetConnector returns a connector by name
func (m *Manager) GetConnector(name string) (CloudConnector, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	connector, exists := m.connectors[name]
	return connector, exists
}

// ListConnectors returns all registered connector names
func (m *Manager) ListConnectors() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	names := make([]string, 0, len(m.connectors))
	for name := range m.connectors {
		names = append(names, name)
	}
	
	return names
}

// ConnectAll connects all registered connectors
func (m *Manager) ConnectAll(ctx context.Context) error {
	m.mutex.RLock()
	connectors := make([]CloudConnector, 0, len(m.connectors))
	names := make([]string, 0, len(m.connectors))
	for name, connector := range m.connectors {
		connectors = append(connectors, connector)
		names = append(names, name)
	}
	m.mutex.RUnlock()
	
	// Connect in parallel
	var wg sync.WaitGroup
	errors := make([]error, len(connectors))
	
	for i, connector := range connectors {
		wg.Add(1)
		go func(idx int, conn CloudConnector, name string) {
			defer wg.Done()
			if err := conn.Connect(ctx); err != nil {
				errors[idx] = fmt.Errorf("failed to connect %s: %w", name, err)
				m.logger.Error("Failed to connect connector", zap.String("name", name), zap.Error(err))
			} else {
				m.logger.Info("Connected cloud connector", zap.String("name", name))
			}
		}(i, connector, names[i])
	}
	
	wg.Wait()
	
	// Check for errors
	var firstError error
	for _, err := range errors {
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	
	return firstError
}

// DisconnectAll disconnects all registered connectors
func (m *Manager) DisconnectAll(ctx context.Context) error {
	m.mutex.RLock()
	connectors := make([]CloudConnector, 0, len(m.connectors))
	names := make([]string, 0, len(m.connectors))
	for name, connector := range m.connectors {
		connectors = append(connectors, connector)
		names = append(names, name)
	}
	m.mutex.RUnlock()
	
	// Disconnect in parallel
	var wg sync.WaitGroup
	
	for i, connector := range connectors {
		wg.Add(1)
		go func(idx int, conn CloudConnector, name string) {
			defer wg.Done()
			if err := conn.Disconnect(ctx); err != nil {
				m.logger.Error("Failed to disconnect connector", zap.String("name", name), zap.Error(err))
			} else {
				m.logger.Info("Disconnected cloud connector", zap.String("name", name))
			}
		}(i, connector, names[i])
	}
	
	wg.Wait()
	
	return nil
}

// SendData sends data to appropriate connectors based on routing rules
func (m *Manager) SendData(ctx context.Context, data *CloudData) error {
	// Apply routing rules to determine which connectors to use
	targetConnectors := m.routeData(data)
	
	if len(targetConnectors) == 0 {
		// Use default connector if no rules match
		if m.config.DefaultConnector != "" {
			targetConnectors = []string{m.config.DefaultConnector}
		} else {
			return fmt.Errorf("no routing rules matched and no default connector configured")
		}
	}
	
	// Add to batch buffer
	m.addToBatch(data)
	
	// Send to target connectors
	var lastError error
	for _, connectorName := range targetConnectors {
		if connector, exists := m.GetConnector(connectorName); exists {
			if connector.IsConnected() {
				if err := connector.SendData(ctx, data); err != nil {
					m.logger.Error("Failed to send data to connector", 
						zap.String("connector", connectorName), zap.Error(err))
					lastError = err
				}
			} else {
				m.logger.Warn("Connector not connected, buffering data", 
					zap.String("connector", connectorName))
				// Add to buffer for retry when connection is restored
				m.bufferData(data)
			}
		}
	}
	
	return lastError
}

// SendBatch sends a batch of data to appropriate connectors
func (m *Manager) SendBatch(ctx context.Context, batch []*CloudData) error {
	if len(batch) == 0 {
		return nil
	}
	
	// Group data by routing rules
	routingGroups := make(map[string][]*CloudData)
	
	for _, data := range batch {
		targetConnectors := m.routeData(data)
		if len(targetConnectors) == 0 && m.config.DefaultConnector != "" {
			targetConnectors = []string{m.config.DefaultConnector}
		}
		
		for _, connectorName := range targetConnectors {
			routingGroups[connectorName] = append(routingGroups[connectorName], data)
		}
	}
	
	// Send to each connector
	var lastError error
	for connectorName, connectorData := range routingGroups {
		if connector, exists := m.GetConnector(connectorName); exists {
			if connector.IsConnected() {
				if err := connector.SendBatch(ctx, connectorData); err != nil {
					m.logger.Error("Failed to send batch to connector", 
						zap.String("connector", connectorName), 
						zap.Int("batchSize", len(connectorData)),
						zap.Error(err))
					lastError = err
				}
			} else {
				// Buffer all data for retry
				for _, data := range connectorData {
					m.bufferData(data)
				}
			}
		}
	}
	
	return lastError
}

// GetHealth returns health status of all connectors
func (m *Manager) GetHealth() map[string]*HealthStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	health := make(map[string]*HealthStatus)
	for name, connector := range m.connectors {
		health[name] = connector.GetHealth()
	}
	
	return health
}

// GetMetrics returns metrics for all connectors
func (m *Manager) GetMetrics() map[string]*ConnectorMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	metrics := make(map[string]*ConnectorMetrics)
	for name, connector := range m.connectors {
		metrics[name] = connector.GetMetrics()
	}
	
	return metrics
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down cloud manager")
	
	// Stop batch processing
	m.cancel()
	
	// Stop health checker
	if m.healthChecker != nil {
		m.healthChecker.Stop()
	}
	
	// Flush any remaining batch data
	m.flushBatch()
	
	// Disconnect all connectors
	if err := m.DisconnectAll(ctx); err != nil {
		m.logger.Error("Error disconnecting connectors during shutdown", zap.Error(err))
	}
	
	// Close buffer
	if err := m.buffer.Close(); err != nil {
		m.logger.Error("Error closing buffer during shutdown", zap.Error(err))
	}
	
	// Wait for goroutines to finish
	m.wg.Wait()
	
	m.logger.Info("Cloud manager shutdown complete")
	return nil
}

// routeData applies routing rules to determine target connectors
func (m *Manager) routeData(data *CloudData) []string {
	var targetConnectors []string
	
	// Apply routing rules in priority order
	for _, rule := range m.config.RoutingRules {
		if m.matchesCondition(data, rule.Condition) {
			targetConnectors = append(targetConnectors, rule.Connectors...)
			break // Use first matching rule
		}
	}
	
	return targetConnectors
}

// matchesCondition checks if data matches a routing condition
func (m *Manager) matchesCondition(data *CloudData, condition string) bool {
	// Simple condition matching - in a real implementation this would be more sophisticated
	if condition == "" {
		return true
	}
	
	// Example: "device_id=PLC001" or "tag_name=temperature"
	// This is a simplified implementation
	switch {
	case condition == fmt.Sprintf("device_id=%s", data.DeviceID):
		return true
	case condition == fmt.Sprintf("tag_name=%s", data.TagName):
		return true
	case condition == fmt.Sprintf("quality=%s", data.Quality):
		return true
	default:
		return false
	}
}

// addToBatch adds data to the batch buffer
func (m *Manager) addToBatch(data *CloudData) {
	m.batchMutex.Lock()
	defer m.batchMutex.Unlock()
	
	m.batchBuffer = append(m.batchBuffer, data)
	
	// Check if batch is full
	if len(m.batchBuffer) >= m.config.BatchSize {
		m.flushBatchLocked()
	}
}

// flushBatch flushes the current batch
func (m *Manager) flushBatch() {
	m.batchMutex.Lock()
	defer m.batchMutex.Unlock()
	m.flushBatchLocked()
}

// flushBatchLocked flushes the batch (caller must hold lock)
func (m *Manager) flushBatchLocked() {
	if len(m.batchBuffer) == 0 {
		return
	}
	
	// Process batch asynchronously
	batch := make([]*CloudData, len(m.batchBuffer))
	copy(batch, m.batchBuffer)
	m.batchBuffer = m.batchBuffer[:0] // Reset buffer
	
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := m.SendBatch(ctx, batch); err != nil {
			m.logger.Error("Failed to send batch", zap.Error(err))
		}
	}()
}

// startBatchProcessing starts the batch timeout processing
func (m *Manager) startBatchProcessing() {
	if m.config.BatchTimeout <= 0 {
		return
	}
	
	m.batchTimer = time.AfterFunc(m.config.BatchTimeout, func() {
		m.flushBatch()
		m.startBatchProcessing() // Reschedule
	})
}

// bufferData adds data to the buffer for retry
func (m *Manager) bufferData(data *CloudData) {
	message := &CloudMessage{
		ID:       data.ID,
		Type:     MessageTypeData,
		Payload:  data,
		Priority: PriorityNormal,
		Created:  time.Now(),
		Retries:  0,
	}
	
	if err := m.buffer.Add(message); err != nil {
		m.logger.Error("Failed to buffer data", zap.Error(err))
	}
}