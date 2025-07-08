package logging

import (
	"context"
	"log/slog"
	"os"
)

// Logger provides a simple logging interface compatible with zap
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new logger instance
func NewLogger(level string) *Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{logger: logger}
}

// NewDevelopment creates a development logger (compatible with zap.NewDevelopment)
func NewDevelopment() (*Logger, error) {
	return NewLogger("debug"), nil
}

// Info logs at info level
func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Debug logs at debug level
func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Warn logs at warn level
func (l *Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error logs at error level
func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// Fatal logs at error level and exits (compatible with zap)
func (l *Logger) Fatal(msg string, args ...any) {
	l.logger.Error(msg, args...)
	os.Exit(1)
}

// With creates a child logger with additional fields
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		logger: l.logger.With(args...),
	}
}

// Named creates a named logger (compatible with zap)
func (l *Logger) Named(name string) *Logger {
	return &Logger{
		logger: l.logger.With("component", name),
	}
}

// Sync flushes any buffered log entries (compatible with zap, no-op for slog)
func (l *Logger) Sync() error {
	return nil
}

// Sugar returns the logger (compatible with zap, returns self)
func (l *Logger) Sugar() *Logger {
	return l
}

// WithContext adds context to logger (placeholder)
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return l
}

// Field helpers to maintain zap compatibility
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a bool field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field
func Duration(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field
func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Strings creates a strings field
func Strings(key string, values []string) Field {
	return Field{Key: key, Value: values}
}

// Helper method to convert fields to slog args
func fieldsToArgs(fields []Field) []any {
	args := make([]any, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	return args
}

// Logging methods with fields
func (l *Logger) InfoF(msg string, fields ...Field) {
	l.logger.Info(msg, fieldsToArgs(fields)...)
}

func (l *Logger) DebugF(msg string, fields ...Field) {
	l.logger.Debug(msg, fieldsToArgs(fields)...)
}

func (l *Logger) WarnF(msg string, fields ...Field) {
	l.logger.Warn(msg, fieldsToArgs(fields)...)
}

func (l *Logger) ErrorF(msg string, fields ...Field) {
	l.logger.Error(msg, fieldsToArgs(fields)...)
}