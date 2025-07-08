package cloud

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestManager(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultManagerConfig()
	config.DefaultConnector = "test" // Set default connector
	
	manager, err := NewManager(logger, config)
	require.NoError(t, err)
	require.NotNil(t, manager)
	
	// Test connector registration
	mockConnector := &MockConnector{}
	err = manager.RegisterConnector("test", mockConnector)
	assert.NoError(t, err)
	
	// Test data sending
	data := &CloudData{
		ID:        "test-1",
		DeviceID:  "PLC001",
		TagName:   "temperature",
		Value:     25.5,
		Quality:   "GOOD",
		Timestamp: time.Now(),
	}
	
	err = manager.SendData(context.Background(), data)
	assert.NoError(t, err)
	
	// Test shutdown
	err = manager.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestBuffer(t *testing.T) {
	logger := zap.NewNop()
	config := &BufferConfig{
		MaxSize:        10,
		FlushInterval:  1 * time.Second,
		PersistentPath: "/tmp/test-buffer",
	}
	
	buffer := NewMemoryBuffer(logger, config)
	
	// Test adding messages
	message := &CloudMessage{
		ID:       "test-1",
		Type:     MessageTypeData,
		Priority: PriorityNormal,
		Created:  time.Now(),
		Payload:  "test data",
	}
	
	err := buffer.Add(message)
	assert.NoError(t, err)
	assert.Equal(t, 1, buffer.Size())
	
	// Test retrieving messages
	messages, err := buffer.Get(1)
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "test-1", messages[0].ID)
	
	// Test removing messages
	err = buffer.Remove([]string{"test-1"})
	assert.NoError(t, err)
	assert.Equal(t, 0, buffer.Size())
}

func TestRetryManager(t *testing.T) {
	logger := zap.NewNop()
	config := &RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Strategy:     RetryStrategyExponential,
		Jitter:       false,
	}
	
	retryManager := NewRetryManager(logger, config)
	
	// Test successful operation
	attempts := 0
	operation := func(ctx context.Context) error {
		attempts++
		return nil
	}
	
	err := retryManager.Execute(context.Background(), operation, "test-operation")
	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
	
	// Test operation that succeeds after retries
	attempts = 0
	operation = func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return NewCloudError("test_error", "test error", "test", "test_op", true)
		}
		return nil
	}
	
	err = retryManager.Execute(context.Background(), operation, "test-operation-retry")
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
	
	// Test operation that fails with max retries
	attempts = 0
	operation = func(ctx context.Context) error {
		attempts++
		return NewCloudError("test_error", "persistent error", "test", "test_op", true)
	}
	
	err = retryManager.Execute(context.Background(), operation, "test-operation-fail")
	assert.Error(t, err)
	assert.Equal(t, 4, attempts) // Initial attempt + 3 retries
}

// MockConnector for testing
type MockConnector struct {
	connected bool
	data      []*CloudData
	health    *HealthStatus
	metrics   *ConnectorMetrics
	config    *ConnectorConfig
}

func (m *MockConnector) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *MockConnector) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *MockConnector) IsConnected() bool {
	return m.connected
}

func (m *MockConnector) SendData(ctx context.Context, data *CloudData) error {
	m.data = append(m.data, data)
	return nil
}

func (m *MockConnector) SendBatch(ctx context.Context, batch []*CloudData) error {
	m.data = append(m.data, batch...)
	return nil
}

func (m *MockConnector) Ping(ctx context.Context) error {
	if !m.connected {
		return NewCloudError("not_connected", "not connected", "mock", "ping", true)
	}
	return nil
}

func (m *MockConnector) GetHealth() *HealthStatus {
	if m.health == nil {
		m.health = &HealthStatus{
			IsHealthy:   m.connected,
			SuccessRate: 1.0,
		}
	}
	return m.health
}

func (m *MockConnector) GetMetrics() *ConnectorMetrics {
	if m.metrics == nil {
		m.metrics = &ConnectorMetrics{
			DataPointsSent: uint64(len(m.data)),
		}
	}
	return m.metrics
}

func (m *MockConnector) GetConfig() *ConnectorConfig {
	if m.config == nil {
		m.config = &ConnectorConfig{
			Name:    "mock",
			Type:    "mock",
			Enabled: true,
		}
	}
	return m.config
}

func (m *MockConnector) ValidateConfig() error {
	return nil
}