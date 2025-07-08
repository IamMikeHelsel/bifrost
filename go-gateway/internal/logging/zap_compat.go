//go:build !zap
// +build !zap

// Package zap provides a minimal compatibility layer over log/slog
package zap

import (
	"log/slog"
	"os"
	"time"
)

// Logger provides zap-compatible logging over slog
type Logger struct {
	logger *slog.Logger
}

// Config represents logger configuration
type Config struct {
	Level            string
	Development      bool
	DisableCaller    bool
	DisableStacktrace bool
	OutputPaths      []string
	ErrorOutputPaths []string
}

// NewDevelopment creates a development logger
func NewDevelopment() (*Logger, error) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	return &Logger{logger: slog.New(handler)}, nil
}

// NewProduction creates a production logger  
func NewProduction() (*Logger, error) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return &Logger{logger: slog.New(handler)}, nil
}

// New creates a logger from config
func (cfg Config) Build() (*Logger, error) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if cfg.Development {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return &Logger{logger: slog.New(handler)}, nil
}

// Logging methods
func (l *Logger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, fieldsToArgs(fields)...)
}

func (l *Logger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, fieldsToArgs(fields)...)
}

func (l *Logger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, fieldsToArgs(fields)...)
}

func (l *Logger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, fieldsToArgs(fields)...)
}

func (l *Logger) Fatal(msg string, fields ...Field) {
	l.logger.Error(msg, fieldsToArgs(fields)...)
	os.Exit(1)
}

func (l *Logger) Panic(msg string, fields ...Field) {
	l.logger.Error(msg, fieldsToArgs(fields)...)
	panic(msg)
}

// Utility methods
func (l *Logger) Sync() error { return nil }
func (l *Logger) Named(name string) *Logger {
	return &Logger{logger: l.logger.With("component", name)}
}
func (l *Logger) With(fields ...Field) *Logger {
	return &Logger{logger: l.logger.With(fieldsToArgs(fields)...)}
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value interface{}
}

// Field constructors
func String(key, val string) Field { return Field{key, val} }
func Int(key string, val int) Field { return Field{key, val} }
func Int64(key string, val int64) Field { return Field{key, val} }
func Float64(key string, val float64) Field { return Field{key, val} }
func Bool(key string, val bool) Field { return Field{key, val} }
func Duration(key string, val time.Duration) Field { return Field{key, val} }
func Time(key string, val time.Time) Field { return Field{key, val} }
func Any(key string, val interface{}) Field { return Field{key, val} }
func Strings(key string, val []string) Field { return Field{key, val} }
func Error(err error) Field { return Field{"error", err} }

func fieldsToArgs(fields []Field) []interface{} {
	args := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		args = append(args, f.Key, f.Value)
	}
	return args
}

// zapcore compatibility stubs
type Level int8
const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel  
	ErrorLevel
	FatalLevel
	PanicLevel
)

func NewAtomicLevel() AtomicLevel { return AtomicLevel{} }
type AtomicLevel struct{}
func (a AtomicLevel) Level() Level { return InfoLevel }
func (a AtomicLevel) SetLevel(Level) {}

// Core interface stub
type Core interface{}
type WriteSyncer interface{}
type Encoder interface{}

func NewCore(Encoder, WriteSyncer, Level) Core { return nil }
func NewTee(...Core) Core { return nil }
func New(Core, ...Option) *Logger { 
	l, _ := NewDevelopment()
	return l
}

type Option func(*Logger)
func AddCaller() Option { return func(*Logger) {} }
func AddStacktrace(Level) Option { return func(*Logger) {} }
func Development() Option { return func(*Logger) {} }