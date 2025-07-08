package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	"github.com/bifrost/go-gateway/internal/gateway"
)

// Config represents the gateway configuration
type Config struct {
	Gateway struct {
		Port           int           `yaml:"port"`
		GRPCPort       int           `yaml:"grpc_port"`
		MaxConnections int           `yaml:"max_connections"`
		DataBufferSize int           `yaml:"data_buffer_size"`
		UpdateInterval time.Duration `yaml:"update_interval"`
		EnableMetrics  bool          `yaml:"enable_metrics"`
		LogLevel       string        `yaml:"log_level"`
	} `yaml:"gateway"`

	Protocols struct {
		Modbus struct {
			DefaultTimeout    time.Duration `yaml:"default_timeout"`
			DefaultUnitID     int           `yaml:"default_unit_id"`
			MaxConnections    int           `yaml:"max_connections"`
			ConnectionTimeout time.Duration `yaml:"connection_timeout"`
			ReadTimeout       time.Duration `yaml:"read_timeout"`
			WriteTimeout      time.Duration `yaml:"write_timeout"`
			EnableKeepAlive   bool          `yaml:"enable_keep_alive"`
		} `yaml:"modbus"`
	} `yaml:"protocols"`
}

func main() {
	// Parse command-line flags
	var (
		configFile  = flag.String("config", "gateway.yaml", "Path to configuration file")
		logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		port        = flag.Int("port", 8080, "HTTP server port")
		grpcPort    = flag.Int("grpc-port", 9090, "gRPC server port")
		healthCheck = flag.Bool("health-check", false, "Perform health check and exit")
	)
	flag.Parse()

	// Handle health check
	if *healthCheck {
		os.Exit(performHealthCheck())
	}

	// Load configuration
	config, err := loadConfig(*configFile)
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Override config with command-line flags
	if *port != 8080 {
		config.Gateway.Port = *port
	}
	if *grpcPort != 9090 {
		config.Gateway.GRPCPort = *grpcPort
	}
	if *logLevel != "info" {
		config.Gateway.LogLevel = *logLevel
	}

	// Set up logging
	logger := setupLogger(config.Gateway.LogLevel)
	defer logger.Sync()

	logger.Info("Starting Bifrost Industrial Gateway",
		zap.String("version", "1.0.0"),
		zap.Int("port", config.Gateway.Port),
		zap.Int("grpc_port", config.Gateway.GRPCPort),
		zap.String("log_level", config.Gateway.LogLevel),
	)

	// Create gateway configuration
	gatewayConfig := &gateway.Config{
		Port:           config.Gateway.Port,
		GRPCPort:       config.Gateway.GRPCPort,
		MaxConnections: config.Gateway.MaxConnections,
		DataBufferSize: config.Gateway.DataBufferSize,
		UpdateInterval: config.Gateway.UpdateInterval,
		EnableMetrics:  config.Gateway.EnableMetrics,
		LogLevel:       config.Gateway.LogLevel,
	}

	// Create and start the gateway
	gw := gateway.NewIndustrialGateway(gatewayConfig, logger)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, shutting down gracefully...")
		cancel()
	}()

	// Start the gateway
	if err := gw.Start(ctx); err != nil {
		logger.Error("Gateway startup failed", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Gateway shutdown complete")
}

func loadConfig(filename string) (*Config, error) {
	// Set default configuration
	config := &Config{}

	// Set defaults
	config.Gateway.Port = 8080
	config.Gateway.GRPCPort = 9090
	config.Gateway.MaxConnections = 1000
	config.Gateway.DataBufferSize = 10000
	config.Gateway.UpdateInterval = 1 * time.Second
	config.Gateway.EnableMetrics = true
	config.Gateway.LogLevel = "info"

	config.Protocols.Modbus.DefaultTimeout = 5 * time.Second
	config.Protocols.Modbus.DefaultUnitID = 1
	config.Protocols.Modbus.MaxConnections = 100
	config.Protocols.Modbus.ConnectionTimeout = 10 * time.Second
	config.Protocols.Modbus.ReadTimeout = 5 * time.Second
	config.Protocols.Modbus.WriteTimeout = 5 * time.Second
	config.Protocols.Modbus.EnableKeepAlive = true

	// Try to load from file
	if data, err := os.ReadFile(filename); err == nil {
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func setupLogger(level string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapLevel),
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
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	return logger
}

func performHealthCheck() int {
	// Simple health check - try to connect to the health endpoint
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://localhost:8080/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return 0
	}

	return 1
}
