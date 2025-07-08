package cloud

import (
	"context"
	"time"
)

// CloudConnector defines the interface for all cloud platform integrations
type CloudConnector interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	
	// Data operations
	SendData(ctx context.Context, data *CloudData) error
	SendBatch(ctx context.Context, batch []*CloudData) error
	
	// Health and monitoring
	Ping(ctx context.Context) error
	GetHealth() *HealthStatus
	GetMetrics() *ConnectorMetrics
	
	// Configuration
	GetConfig() *ConnectorConfig
	ValidateConfig() error
}

// CloudData represents data being sent to cloud platforms
type CloudData struct {
	ID        string                 `json:"id"`
	DeviceID  string                 `json:"device_id"`
	TagName   string                 `json:"tag_name"`
	Value     interface{}            `json:"value"`
	Quality   string                 `json:"quality"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ConnectorConfig holds common configuration for cloud connectors
type ConnectorConfig struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Enabled  bool   `yaml:"enabled"`
	
	// Connection settings
	Endpoint    string        `yaml:"endpoint"`
	Timeout     time.Duration `yaml:"timeout"`
	RetryCount  int           `yaml:"retry_count"`
	RetryDelay  time.Duration `yaml:"retry_delay"`
	
	// Security settings
	TLSEnabled  bool   `yaml:"tls_enabled"`
	CertFile    string `yaml:"cert_file,omitempty"`
	KeyFile     string `yaml:"key_file,omitempty"`
	CAFile      string `yaml:"ca_file,omitempty"`
	
	// Buffering
	BufferSize     int           `yaml:"buffer_size"`
	FlushInterval  time.Duration `yaml:"flush_interval"`
	DiskPersistent bool          `yaml:"disk_persistent"`
	
	// Provider-specific configuration
	ProviderConfig map[string]interface{} `yaml:"provider_config,omitempty"`
}

// HealthStatus represents the health of a cloud connector
type HealthStatus struct {
	IsHealthy         bool          `json:"is_healthy"`
	LastCommunication time.Time     `json:"last_communication"`
	ResponseTime      time.Duration `json:"response_time"`
	ErrorCount        uint64        `json:"error_count"`
	SuccessRate       float64       `json:"success_rate"`
	ConnectionUptime  time.Duration `json:"connection_uptime"`
	LastError         string        `json:"last_error,omitempty"`
}

// ConnectorMetrics holds performance metrics for cloud connectors
type ConnectorMetrics struct {
	DataPointsSent      uint64        `json:"data_points_sent"`
	DataPointsFailed    uint64        `json:"data_points_failed"`
	BatchesSent         uint64        `json:"batches_sent"`
	BatchesFailed       uint64        `json:"batches_failed"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	TotalUptime         time.Duration `json:"total_uptime"`
	ConnectionAttempts  uint64        `json:"connection_attempts"`
	ReconnectCount      uint64        `json:"reconnect_count"`
}

// CloudMessage represents a message in the cloud connector system
type CloudMessage struct {
	ID       string      `json:"id"`
	Type     MessageType `json:"type"`
	Payload  interface{} `json:"payload"`
	Priority Priority    `json:"priority"`
	Created  time.Time   `json:"created"`
	Expires  time.Time   `json:"expires,omitempty"`
	Retries  int         `json:"retries"`
}

// MessageType defines the type of cloud message
type MessageType string

const (
	MessageTypeData    MessageType = "data"
	MessageTypeEvent   MessageType = "event"
	MessageTypeAlarm   MessageType = "alarm"
	MessageTypeCommand MessageType = "command"
)

// Priority defines message priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// CloudError represents errors from cloud operations
type CloudError struct {
	Code       string    `json:"code"`
	Message    string    `json:"message"`
	Connector  string    `json:"connector"`
	Operation  string    `json:"operation"`
	Timestamp  time.Time `json:"timestamp"`
	Retryable  bool      `json:"retryable"`
	StatusCode int       `json:"status_code,omitempty"`
}

func (e *CloudError) Error() string {
	return e.Message
}

// NewCloudError creates a new cloud error
func NewCloudError(code, message, connector, operation string, retryable bool) *CloudError {
	return &CloudError{
		Code:      code,
		Message:   message,
		Connector: connector,
		Operation: operation,
		Timestamp: time.Now(),
		Retryable: retryable,
	}
}