package security

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// AuditLogger handles security event logging
type AuditLogger struct {
	logger *zap.Logger
	config AuditConfig
}

type AuditConfig struct {
	Enabled    bool   `yaml:"enabled"`
	LogFile    string `yaml:"log_file"`
	LogLevel   string `yaml:"log_level"`
	MaxSize    int    `yaml:"max_size"`    // MB
	MaxBackups int    `yaml:"max_backups"`
}

// SecurityEvent represents a security event for auditing
type SecurityEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	Severity    string                 `json:"severity"`
	Source      string                 `json:"source"`
	UserID      string                 `json:"user_id,omitempty"`
	DeviceID    string                 `json:"device_id,omitempty"`
	Action      string                 `json:"action"`
	Result      string                 `json:"result"` // success, failure, error
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
}

// Event types
const (
	EventTypeAuthentication = "authentication"
	EventTypeAuthorization  = "authorization"
	EventTypeDataAccess     = "data_access"
	EventTypeConfiguration  = "configuration"
	EventTypeConnection     = "connection"
	EventTypeCrypto         = "cryptography"
	EventTypeProtocol       = "protocol"
	EventTypeAudit          = "audit"
)

// Severity levels
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityError    = "error"
	SeverityCritical = "critical"
)

// Results
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
	ResultError   = "error"
)

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config AuditConfig) (*AuditLogger, error) {
	if !config.Enabled {
		return &AuditLogger{
			logger: zap.NewNop(),
			config: config,
		}, nil
	}

	// Configure audit logging
	zapConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(parseLogLevel(config.LogLevel)),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{config.LogFile},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}

	return &AuditLogger{
		logger: logger,
		config: config,
	}, nil
}

// LogEvent logs a security event
func (al *AuditLogger) LogEvent(event SecurityEvent) {
	if !al.config.Enabled {
		return
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Convert event to JSON for structured logging
	eventJSON, err := json.Marshal(event)
	if err != nil {
		al.logger.Error("Failed to marshal security event", zap.Error(err))
		return
	}

	// Log at appropriate level based on severity
	switch event.Severity {
	case SeverityInfo:
		al.logger.Info("Security Event", zap.String("event", string(eventJSON)))
	case SeverityWarning:
		al.logger.Warn("Security Event", zap.String("event", string(eventJSON)))
	case SeverityError:
		al.logger.Error("Security Event", zap.String("event", string(eventJSON)))
	case SeverityCritical:
		al.logger.Error("Critical Security Event", zap.String("event", string(eventJSON)))
	default:
		al.logger.Info("Security Event", zap.String("event", string(eventJSON)))
	}
}

// Convenience methods for common events

// LogAuthentication logs authentication events
func (al *AuditLogger) LogAuthentication(userID, source string, success bool, details map[string]interface{}) {
	result := ResultSuccess
	severity := SeverityInfo
	if !success {
		result = ResultFailure
		severity = SeverityWarning
	}

	event := SecurityEvent{
		EventType: EventTypeAuthentication,
		Severity:  severity,
		Source:    source,
		UserID:    userID,
		Action:    "login",
		Result:    result,
		Message:   fmt.Sprintf("User %s authentication %s", userID, result),
		Details:   details,
	}
	al.LogEvent(event)
}

// LogDataAccess logs data access events
func (al *AuditLogger) LogDataAccess(userID, deviceID, action string, success bool, details map[string]interface{}) {
	result := ResultSuccess
	severity := SeverityInfo
	if !success {
		result = ResultFailure
		severity = SeverityWarning
	}

	event := SecurityEvent{
		EventType: EventTypeDataAccess,
		Severity:  severity,
		Source:    "gateway",
		UserID:    userID,
		DeviceID:  deviceID,
		Action:    action,
		Result:    result,
		Message:   fmt.Sprintf("Data %s on device %s: %s", action, deviceID, result),
		Details:   details,
	}
	al.LogEvent(event)
}

// LogProtocolSecurity logs protocol security events
func (al *AuditLogger) LogProtocolSecurity(protocol, deviceID, action string, success bool, details map[string]interface{}) {
	result := ResultSuccess
	severity := SeverityInfo
	if !success {
		result = ResultFailure
		severity = SeverityError
	}

	event := SecurityEvent{
		EventType: EventTypeProtocol,
		Severity:  severity,
		Source:    protocol,
		DeviceID:  deviceID,
		Action:    action,
		Result:    result,
		Message:   fmt.Sprintf("Protocol %s security %s for device %s: %s", protocol, action, deviceID, result),
		Details:   details,
	}
	al.LogEvent(event)
}

// LogConfigurationChange logs configuration changes
func (al *AuditLogger) LogConfigurationChange(userID, component, action string, details map[string]interface{}) {
	event := SecurityEvent{
		EventType: EventTypeConfiguration,
		Severity:  SeverityInfo,
		Source:    "gateway",
		UserID:    userID,
		Action:    action,
		Result:    ResultSuccess,
		Message:   fmt.Sprintf("Configuration %s for %s", action, component),
		Details:   details,
	}
	al.LogEvent(event)
}

// LogCryptoOperation logs cryptographic operations
func (al *AuditLogger) LogCryptoOperation(operation string, success bool, details map[string]interface{}) {
	result := ResultSuccess
	severity := SeverityInfo
	if !success {
		result = ResultFailure
		severity = SeverityError
	}

	event := SecurityEvent{
		EventType: EventTypeCrypto,
		Severity:  severity,
		Source:    "crypto",
		Action:    operation,
		Result:    result,
		Message:   fmt.Sprintf("Cryptographic operation %s: %s", operation, result),
		Details:   details,
	}
	al.LogEvent(event)
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	if al.logger != nil {
		return al.logger.Sync()
	}
	return nil
}

func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}