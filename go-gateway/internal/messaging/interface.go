package messaging

import (
	"context"
	"time"

	"bifrost-gateway/internal/protocols"
)

// MessagingLayer defines the interface for all messaging implementations
type MessagingLayer interface {
	// Lifecycle management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool

	// Publishing operations
	PublishDeviceData(deviceID string, data *protocols.Tag) error
	PublishDeviceEvent(deviceID string, event *DeviceEvent) error
	PublishAlarm(deviceID string, alarm *AlarmEvent) error

	// Subscription operations
	Subscribe(subject string, handler MessageHandler) error
	Unsubscribe(subject string) error

	// Request-reply patterns (for NATS)
	RequestReply(subject string, data []byte, timeout time.Duration) ([]byte, error)

	// Health and diagnostics
	GetConnectionStatus() *ConnectionStatus
	GetMetrics() *MessagingMetrics
}

// MessageHandler defines the callback function for incoming messages
type MessageHandler func(subject string, data []byte) error

// DeviceEvent represents a device-level event
type DeviceEvent struct {
	DeviceID   string                 `json:"device_id"`
	EventType  string                 `json:"event_type"`
	Timestamp  time.Time              `json:"timestamp"`
	Data       map[string]interface{} `json:"data"`
	Severity   string                 `json:"severity"`
	Message    string                 `json:"message"`
}

// AlarmEvent represents an alarm condition
type AlarmEvent struct {
	DeviceID    string    `json:"device_id"`
	AlarmID     string    `json:"alarm_id"`
	AlarmType   string    `json:"alarm_type"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	Message     string    `json:"message"`
	Acknowledged bool     `json:"acknowledged"`
	Value       interface{} `json:"value,omitempty"`
	Threshold   interface{} `json:"threshold,omitempty"`
}

// ConnectionStatus provides status information about the messaging connection
type ConnectionStatus struct {
	Connected     bool      `json:"connected"`
	LastConnected time.Time `json:"last_connected"`
	LastError     string    `json:"last_error,omitempty"`
	ReconnectCount int      `json:"reconnect_count"`
}

// MessagingMetrics provides performance metrics for messaging operations
type MessagingMetrics struct {
	MessagesPublished uint64 `json:"messages_published"`
	MessagesReceived  uint64 `json:"messages_received"`
	PublishErrors     uint64 `json:"publish_errors"`
	ReceiveErrors     uint64 `json:"receive_errors"`
	AverageLatency    time.Duration `json:"average_latency"`
	LastUpdate        time.Time `json:"last_update"`
}

// TopicBuilder helps construct standardized topic names
type TopicBuilder interface {
	TelemetryTopic(siteID, deviceID, tagName string) string
	CommandTopic(siteID, deviceID, commandType string) string
	AlarmTopic(siteID, deviceID, alarmLevel string) string
	EventTopic(siteID, deviceID, eventType string) string
	DiagnosticTopic(siteID, deviceID, metricName string) string
}