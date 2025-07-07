package protocols

import (
	"context"
	"time"
)

// ProtocolHandler defines the interface for all industrial protocol implementations
type ProtocolHandler interface {
	// Connection management
	Connect(device *Device) error
	Disconnect(device *Device) error
	IsConnected(device *Device) bool

	// Data operations
	ReadTag(device *Device, tag *Tag) (interface{}, error)
	WriteTag(device *Device, tag *Tag, value interface{}) error
	ReadMultipleTags(device *Device, tags []*Tag) (map[string]interface{}, error)

	// Device discovery and information
	DiscoverDevices(ctx context.Context, networkRange string) ([]*Device, error)
	GetDeviceInfo(device *Device) (*DeviceInfo, error)

	// Protocol-specific operations
	GetSupportedDataTypes() []string
	ValidateTagAddress(address string) error

	// Health and diagnostics
	Ping(device *Device) error
	GetDiagnostics(device *Device) (*Diagnostics, error)
}

// Device represents an industrial device
type Device struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Protocol string                 `json:"protocol"`
	Address  string                 `json:"address"`
	Port     int                    `json:"port"`
	Config   map[string]interface{} `json:"config"`

	// Runtime state
	Connected    bool      `json:"connected"`
	LastSeen     time.Time `json:"last_seen"`
	ConnectionID string    `json:"-"` // Internal connection identifier

	// Protocol-specific data
	ProtocolData interface{} `json:"-"`
}

// Tag represents a data point on an industrial device
type Tag struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Address     string      `json:"address"`
	DataType    string      `json:"data_type"`
	Value       interface{} `json:"value"`
	Quality     Quality     `json:"quality"`
	Timestamp   time.Time   `json:"timestamp"`
	Writable    bool        `json:"writable"`
	Unit        string      `json:"unit"`
	Description string      `json:"description"`

	// Scaling and conversion
	ScaleFactor float64 `json:"scale_factor,omitempty"`
	Offset      float64 `json:"offset,omitempty"`

	// Protocol-specific addressing
	ProtocolConfig map[string]interface{} `json:"protocol_config,omitempty"`
}

// Quality represents the quality of a tag value
type Quality string

const (
	QualityGood      Quality = "GOOD"
	QualityBad       Quality = "BAD"
	QualityUncertain Quality = "UNCERTAIN"
	QualityStale     Quality = "STALE"
)

// DeviceInfo contains detailed device information
type DeviceInfo struct {
	Vendor          string            `json:"vendor"`
	Model           string            `json:"model"`
	SerialNumber    string            `json:"serial_number"`
	FirmwareVersion string            `json:"firmware_version"`
	Capabilities    []string          `json:"capabilities"`
	MaxConnections  int               `json:"max_connections"`
	SupportedRates  []int             `json:"supported_rates"`
	CustomInfo      map[string]string `json:"custom_info"`
}

// Diagnostics contains device health and performance information
type Diagnostics struct {
	IsHealthy         bool              `json:"is_healthy"`
	LastCommunication time.Time         `json:"last_communication"`
	ResponseTime      time.Duration     `json:"response_time"`
	ErrorCount        uint64            `json:"error_count"`
	SuccessRate       float64           `json:"success_rate"`
	ConnectionUptime  time.Duration     `json:"connection_uptime"`
	Errors            []DiagnosticError `json:"recent_errors"`

	// Protocol-specific diagnostics
	ProtocolDiagnostics interface{} `json:"protocol_diagnostics,omitempty"`
}

// DiagnosticError represents a communication or protocol error
type DiagnosticError struct {
	Timestamp   time.Time `json:"timestamp"`
	ErrorCode   string    `json:"error_code"`
	Description string    `json:"description"`
	Operation   string    `json:"operation"`
	Address     string    `json:"address,omitempty"`
}

// ConnectionConfig holds common connection parameters
type ConnectionConfig struct {
	Timeout        time.Duration `json:"timeout"`
	RetryCount     int           `json:"retry_count"`
	RetryDelay     time.Duration `json:"retry_delay"`
	KeepAlive      bool          `json:"keep_alive"`
	KeepAliveDelay time.Duration `json:"keep_alive_delay"`
	MaxConcurrent  int           `json:"max_concurrent"`
}

// DataType represents supported data types across protocols
type DataType string

const (
	DataTypeBool    DataType = "bool"
	DataTypeInt16   DataType = "int16"
	DataTypeUInt16  DataType = "uint16"
	DataTypeInt32   DataType = "int32"
	DataTypeUInt32  DataType = "uint32"
	DataTypeInt64   DataType = "int64"
	DataTypeUInt64  DataType = "uint64"
	DataTypeFloat32 DataType = "float32"
	DataTypeFloat64 DataType = "float64"
	DataTypeString  DataType = "string"
	DataTypeBytes   DataType = "bytes"
)

// ProtocolError represents protocol-specific errors
type ProtocolError struct {
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Operation   string    `json:"operation"`
	Address     string    `json:"address,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Recoverable bool      `json:"recoverable"`
}

func (e *ProtocolError) Error() string {
	return e.Message
}

// NewProtocolError creates a new protocol error
func NewProtocolError(code, message, operation string) *ProtocolError {
	return &ProtocolError{
		Code:      code,
		Message:   message,
		Operation: operation,
		Timestamp: time.Now(),
	}
}

// BatchOperation represents a batch read/write operation
type BatchOperation struct {
	Tags      []*Tag                 `json:"tags"`
	Operation string                 `json:"operation"`        // "read" or "write"
	Values    map[string]interface{} `json:"values,omitempty"` // for write operations
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	Success   bool                   `json:"success"`
	Results   map[string]interface{} `json:"results"`
	Errors    map[string]error       `json:"errors"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}
