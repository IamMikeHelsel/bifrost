package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// FileAuditLogger implements audit logging to files with rotation
type FileAuditLogger struct {
	logger *zap.Logger
	file   *os.File
	mu     sync.Mutex
	config *SecurityConfig
}

// NewFileAuditLogger creates a new file-based audit logger
func NewFileAuditLogger(config *SecurityConfig) (*FileAuditLogger, error) {
	// Ensure the log directory exists
	logDir := filepath.Dir(config.Audit.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	
	// Create audit logger configuration
	auditConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(getLogLevel(config.Audit.LogLevel)),
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
		OutputPaths:      []string{config.Audit.LogFile},
		ErrorOutputPaths: []string{"stderr"},
	}
	
	logger, err := auditConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build audit logger: %w", err)
	}
	
	return &FileAuditLogger{
		logger: logger,
		config: config,
	}, nil
}

// LogEvent logs an audit event
func (f *FileAuditLogger) LogEvent(event *AuditEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Convert event to structured fields
	fields := []zap.Field{
		zap.Time("audit_timestamp", event.Timestamp),
		zap.String("event_type", event.EventType),
		zap.String("action", event.Action),
		zap.String("result", event.Result),
	}
	
	if event.UserID != "" {
		fields = append(fields, zap.String("user_id", event.UserID))
	}
	if event.DeviceID != "" {
		fields = append(fields, zap.String("device_id", event.DeviceID))
	}
	if event.Resource != "" {
		fields = append(fields, zap.String("resource", event.Resource))
	}
	if event.RemoteAddr != "" {
		fields = append(fields, zap.String("remote_addr", event.RemoteAddr))
	}
	if event.UserAgent != "" {
		fields = append(fields, zap.String("user_agent", event.UserAgent))
	}
	if event.SessionID != "" {
		fields = append(fields, zap.String("session_id", event.SessionID))
	}
	if event.Details != nil {
		if detailsJSON, err := json.Marshal(event.Details); err == nil {
			fields = append(fields, zap.String("details", string(detailsJSON)))
		}
	}
	
	// Log at appropriate level based on result
	switch event.Result {
	case "failure", "error":
		f.logger.Error("Security audit event", fields...)
	case "success":
		f.logger.Info("Security audit event", fields...)
	default:
		f.logger.Warn("Security audit event", fields...)
	}
	
	return nil
}

// Close closes the audit logger
func (f *FileAuditLogger) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.logger != nil {
		_ = f.logger.Sync()
	}
	
	if f.file != nil {
		return f.file.Close()
	}
	
	return nil
}

// getLogLevel converts string log level to zap level
func getLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// SecurityEventTypes defines standard security event types
var SecurityEventTypes = struct {
	Authentication  string
	Authorization   string
	DataAccess      string
	DataModification string
	SystemAccess    string
	ConfigChange    string
	KeyRotation     string
	CertificateOp   string
	ProtocolSecurity string
	AuditFailure    string
}{
	Authentication:   "authentication",
	Authorization:    "authorization",
	DataAccess:       "data_access",
	DataModification: "data_modification",
	SystemAccess:     "system_access",
	ConfigChange:     "config_change",
	KeyRotation:      "key_rotation",
	CertificateOp:    "certificate_operation",
	ProtocolSecurity: "protocol_security",
	AuditFailure:     "audit_failure",
}

// SecurityActions defines standard security actions
var SecurityActions = struct {
	Login      string
	Logout     string
	Read       string
	Write      string
	Delete     string
	Create     string
	Update     string
	Connect    string
	Disconnect string
	Subscribe  string
	Publish    string
	Encrypt    string
	Decrypt    string
	Rotate     string
	Revoke     string
}{
	Login:      "login",
	Logout:     "logout",
	Read:       "read",
	Write:      "write",
	Delete:     "delete",
	Create:     "create",
	Update:     "update",
	Connect:    "connect",
	Disconnect: "disconnect",
	Subscribe:  "subscribe",
	Publish:    "publish",
	Encrypt:    "encrypt",
	Decrypt:    "decrypt",
	Rotate:     "rotate",
	Revoke:     "revoke",
}

// SecurityResults defines standard security results
var SecurityResults = struct {
	Success string
	Failure string
	Error   string
	Denied  string
}{
	Success: "success",
	Failure: "failure",
	Error:   "error",
	Denied:  "denied",
}

// Helper functions for creating common audit events

// NewAuthenticationEvent creates an authentication audit event
func NewAuthenticationEvent(userID, action, result, remoteAddr string, details map[string]interface{}) *AuditEvent {
	return &AuditEvent{
		EventType:  SecurityEventTypes.Authentication,
		UserID:     userID,
		Action:     action,
		Result:     result,
		RemoteAddr: remoteAddr,
		Details:    details,
	}
}

// NewDataAccessEvent creates a data access audit event
func NewDataAccessEvent(userID, deviceID, action, resource, result string, details map[string]interface{}) *AuditEvent {
	return &AuditEvent{
		EventType: SecurityEventTypes.DataAccess,
		UserID:    userID,
		DeviceID:  deviceID,
		Action:    action,
		Resource:  resource,
		Result:    result,
		Details:   details,
	}
}

// NewProtocolSecurityEvent creates a protocol security audit event
func NewProtocolSecurityEvent(deviceID, action, result string, details map[string]interface{}) *AuditEvent {
	return &AuditEvent{
		EventType: SecurityEventTypes.ProtocolSecurity,
		DeviceID:  deviceID,
		Action:    action,
		Result:    result,
		Details:   details,
	}
}