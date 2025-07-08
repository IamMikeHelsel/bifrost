package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/bifrost/go-gateway/internal/protocols"
)

// EtherNet/IP Demo Application
// This example demonstrates the usage of the EtherNet/IP protocol handler
// for communicating with Allen-Bradley PLCs and other EtherNet/IP devices.

var (
	deviceAddress = flag.String("address", "192.168.1.100", "PLC IP address")
	devicePort    = flag.Int("port", 44818, "PLC port (default: 44818)")
	tagName       = flag.String("tag", "TestTag", "Tag name to read/write")
	networkRange  = flag.String("scan", "", "Network range to scan for devices (e.g., 192.168.1.0/24)")
	logLevel      = flag.String("log", "info", "Log level (debug, info, warn, error)")
	demo          = flag.String("demo", "basic", "Demo type (basic, discovery, performance, diagnostics)")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger := initLogger(*logLevel)
	defer logger.Sync()

	logger.Info("Starting EtherNet/IP Demo Application",
		zap.String("device_address", *deviceAddress),
		zap.Int("device_port", *devicePort),
		zap.String("demo_type", *demo),
	)

	// Create EtherNet/IP handler
	handler := protocols.NewEtherNetIPHandler(logger)
	ethernetIPHandler := handler.(*protocols.EtherNetIPHandler)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// Run the selected demo
	switch *demo {
	case "basic":
		runBasicDemo(ctx, ethernetIPHandler, logger)
	case "discovery":
		runDiscoveryDemo(ctx, ethernetIPHandler, logger)
	case "performance":
		runPerformanceDemo(ctx, ethernetIPHandler, logger)
	case "diagnostics":
		runDiagnosticsDemo(ctx, ethernetIPHandler, logger)
	default:
		logger.Error("Unknown demo type", zap.String("demo", *demo))
		os.Exit(1)
	}

	logger.Info("Demo completed")
}

// initLogger initializes the logger with the specified level
func initLogger(level string) *zap.Logger {
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
		Level:             zap.NewAtomicLevelAt(zapLevel),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	return logger
}

// runBasicDemo demonstrates basic EtherNet/IP operations
func runBasicDemo(ctx context.Context, handler *protocols.EtherNetIPHandler, logger *zap.Logger) {
	logger.Info("Running Basic EtherNet/IP Demo")

	// Create device configuration
	device := &protocols.Device{
		ID:       "demo-plc",
		Name:     "Demo Allen-Bradley PLC",
		Protocol: "ethernet-ip",
		Address:  *deviceAddress,
		Port:     *devicePort,
		Config:   make(map[string]interface{}),
	}

	// Test connection
	logger.Info("Connecting to device...")
	if err := handler.Connect(device); err != nil {
		logger.Error("Failed to connect to device", zap.Error(err))
		return
	}
	defer handler.Disconnect(device)

	logger.Info("Successfully connected to device")

	// Get device information
	if deviceInfo, err := handler.GetDeviceInfo(device); err == nil {
		logger.Info("Device Information",
			zap.String("vendor", deviceInfo.Vendor),
			zap.String("model", deviceInfo.Model),
			zap.String("serial_number", deviceInfo.SerialNumber),
			zap.String("firmware_version", deviceInfo.FirmwareVersion),
			zap.Strings("capabilities", deviceInfo.Capabilities),
		)
	}

	// Create test tags
	tags := []*protocols.Tag{
		{
			ID:       "tag1",
			Name:     "Boolean Tag",
			Address:  "BooleanTag",
			DataType: string(protocols.DataTypeBool),
			Writable: true,
		},
		{
			ID:       "tag2",
			Name:     "Integer Tag",
			Address:  "IntegerTag",
			DataType: string(protocols.DataTypeInt32),
			Writable: true,
		},
		{
			ID:       "tag3",
			Name:     "Float Tag",
			Address:  "FloatTag",
			DataType: string(protocols.DataTypeFloat32),
			Writable: true,
		},
		{
			ID:       "tag4",
			Name:     "String Tag",
			Address:  "StringTag",
			DataType: string(protocols.DataTypeString),
			Writable: true,
		},
		{
			ID:       "tag5",
			Name:     "Array Tag",
			Address:  "ArrayTag[0]",
			DataType: string(protocols.DataTypeInt32),
			Writable: false,
		},
	}

	// Test individual tag reads
	logger.Info("Testing individual tag reads...")
	for _, tag := range tags {
		if value, err := handler.ReadTag(device, tag); err == nil {
			logger.Info("Tag read successful",
				zap.String("tag_id", tag.ID),
				zap.String("tag_name", tag.Name),
				zap.Any("value", value),
			)
		} else {
			logger.Warn("Tag read failed",
				zap.String("tag_id", tag.ID),
				zap.Error(err),
			)
		}
	}

	// Test batch read
	logger.Info("Testing batch tag read...")
	if results, err := handler.ReadMultipleTags(device, tags); err == nil {
		logger.Info("Batch read successful", zap.Int("tags_read", len(results)))
		for tagID, value := range results {
			logger.Debug("Batch read result",
				zap.String("tag_id", tagID),
				zap.Any("value", value),
			)
		}
	} else {
		logger.Error("Batch read failed", zap.Error(err))
	}

	// Test tag writes
	logger.Info("Testing tag writes...")
	writeTests := []struct {
		tag   *protocols.Tag
		value interface{}
	}{
		{tags[0], true},
		{tags[1], int32(12345)},
		{tags[2], float32(3.14159)},
		{tags[3], "Hello EtherNet/IP"},
	}

	for _, test := range writeTests {
		if test.tag.Writable {
			if err := handler.WriteTag(device, test.tag, test.value); err == nil {
				logger.Info("Tag write successful",
					zap.String("tag_id", test.tag.ID),
				zap.Any("value", test.value),
				)

				// Verify write by reading back
				if readValue, err := handler.ReadTag(device, test.tag); err == nil {
					logger.Info("Write verification successful",
						zap.String("tag_id", test.tag.ID),
						zap.Any("written_value", test.value),
						zap.Any("read_value", readValue),
					)
				}
			} else {
				logger.Warn("Tag write failed",
					zap.String("tag_id", test.tag.ID),
					zap.Error(err),
				)
			}
		}
	}

	// Test ping
	logger.Info("Testing device ping...")
	if err := handler.Ping(device); err == nil {
		logger.Info("Ping successful")
	} else {
		logger.Error("Ping failed", zap.Error(err))
	}

	// Get diagnostics
	logger.Info("Getting device diagnostics...")
	if diag, err := handler.GetDiagnostics(device); err == nil {
		logger.Info("Diagnostics retrieved",
			zap.Bool("is_healthy", diag.IsHealthy),
			zap.Time("last_communication", diag.LastCommunication),
			zap.Duration("connection_uptime", diag.ConnectionUptime),
			zap.Float64("success_rate", diag.SuccessRate),
		)
	} else {
		logger.Error("Failed to get diagnostics", zap.Error(err))
	}
}

// runDiscoveryDemo demonstrates device discovery capabilities
func runDiscoveryDemo(ctx context.Context, handler *protocols.EtherNetIPHandler, logger *zap.Logger) {
	logger.Info("Running EtherNet/IP Discovery Demo")

	scanRange := *networkRange
	if scanRange == "" {
		scanRange = "192.168.1.0/24"
		logger.Info("No network range specified, using default", zap.String("range", scanRange))
	}

	logger.Info("Scanning for EtherNet/IP devices...", zap.String("range", scanRange))

	// Set a timeout for discovery
	discoveryCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	devices, err := handler.DiscoverDevices(discoveryCtx, scanRange)
	if err != nil {
		logger.Error("Device discovery failed", zap.Error(err))
		return
	}

	logger.Info("Device discovery completed", zap.Int("devices_found", len(devices)))

	if len(devices) == 0 {
		logger.Info("No EtherNet/IP devices found in the specified range")
		return
	}

	// Display discovered devices
	for i, device := range devices {
		logger.Info("Discovered device",
			zap.Int("index", i+1),
			zap.String("device_id", device.ID),
			zap.String("name", device.Name),
			zap.String("address", device.Address),
			zap.Int("port", device.Port),
		)

		// Try to get detailed information for each device
		if err := handler.Connect(device); err == nil {
			if deviceInfo, err := handler.GetDeviceInfo(device); err == nil {
				logger.Info("Device details",
					zap.String("vendor", deviceInfo.Vendor),
					zap.String("model", deviceInfo.Model),
					zap.String("serial_number", deviceInfo.SerialNumber),
					zap.String("firmware_version", deviceInfo.FirmwareVersion),
				)
			}
			handler.Disconnect(device)
		}
	}
}

// runPerformanceDemo demonstrates performance optimization features
func runPerformanceDemo(ctx context.Context, handler *protocols.EtherNetIPHandler, logger *zap.Logger) {
	logger.Info("Running EtherNet/IP Performance Demo")

	// Create device configuration
	device := &protocols.Device{
		ID:       "perf-test-plc",
		Name:     "Performance Test PLC",
		Protocol: "ethernet-ip",
		Address:  *deviceAddress,
		Port:     *devicePort,
		Config:   make(map[string]interface{}),
	}

	// Connect to device
	if err := handler.Connect(device); err != nil {
		logger.Error("Failed to connect to device", zap.Error(err))
		return
	}
	defer handler.Disconnect(device)

	// Create a large number of test tags
	var tags []*protocols.Tag
	for i := 0; i < 100; i++ {
		tags = append(tags, &protocols.Tag{
			ID:       fmt.Sprintf("perf_tag_%d", i),
			Name:     fmt.Sprintf("Performance Tag %d", i),
			Address:  fmt.Sprintf("PerfTag%d", i),
			DataType: string(protocols.DataTypeInt32),
			Writable: false,
		})
	}

	// Benchmark individual reads
	logger.Info("Benchmarking individual tag reads...")
	startTime := time.Now()
	successCount := 0

	for _, tag := range tags {
		if _, err := handler.ReadTag(device, tag); err == nil {
			successCount++
		}
	}

	individualDuration := time.Since(startTime)
	individualRate := float64(successCount) / individualDuration.Seconds()

	logger.Info("Individual read benchmark completed",
		zap.Int("total_tags", len(tags)),
		zap.Int("successful_reads", successCount),
		zap.Duration("total_time", individualDuration),
		zap.Float64("reads_per_second", individualRate),
	)

	// Benchmark batch reads
	logger.Info("Benchmarking batch tag reads...")
	startTime = time.Now()

	results, err := handler.ReadMultipleTags(device, tags)
	batchDuration := time.Since(startTime)

	if err == nil {
		batchRate := float64(len(results)) / batchDuration.Seconds()
		logger.Info("Batch read benchmark completed",
			zap.Int("total_tags", len(tags)),
			zap.Int("successful_reads", len(results)),
			zap.Duration("total_time", batchDuration),
			zap.Float64("reads_per_second", batchRate),
			zap.Float64("performance_improvement", batchRate/individualRate),
		)
	} else {
		logger.Error("Batch read benchmark failed", zap.Error(err))
	}

	// Test concurrent access
	logger.Info("Testing concurrent access...")
	concurrentCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	concurrentResults := make(chan int, 10)

	for i := 0; i < 10; i++ {
		go func(workerID int) {
			count := 0
			for {
				select {
				case <-concurrentCtx.Done():
					concurrentResults <- count
					return
				default:
					if _, err := handler.ReadTag(device, tags[workerID%len(tags)]); err == nil {
						count++
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	time.Sleep(30 * time.Second)
	cancel()

	totalConcurrentReads := 0
	for i := 0; i < 10; i++ {
		totalConcurrentReads += <-concurrentResults
	}

	logger.Info("Concurrent access test completed",
		zap.Int("total_concurrent_reads", totalConcurrentReads),
		zap.Float64("concurrent_reads_per_second", float64(totalConcurrentReads)/30.0),
	)
}

// runDiagnosticsDemo demonstrates diagnostic and monitoring capabilities
func runDiagnosticsDemo(ctx context.Context, handler *protocols.EtherNetIPHandler, logger *zap.Logger) {
	logger.Info("Running EtherNet/IP Diagnostics Demo")

	// Create device configuration
	device := &protocols.Device{
		ID:       "diag-test-plc",
		Name:     "Diagnostics Test PLC",
		Protocol: "ethernet-ip",
		Address:  *deviceAddress,
		Port:     *devicePort,
		Config:   make(map[string]interface{}),
	}

	// Connect to device
	if err := handler.Connect(device); err != nil {
		logger.Error("Failed to connect to device", zap.Error(err))
		return
	}
	defer handler.Disconnect(device)

	// Perform some operations to generate diagnostic data
	testTag := &protocols.Tag{
		ID:       "diag_tag",
		Name:     "Diagnostic Tag",
		Address:  *tagName,
		DataType: string(protocols.DataTypeInt32),
		Writable: true,
	}

	logger.Info("Performing operations to generate diagnostic data...")
	for i := 0; i < 20; i++ {
		// Read operations
		if value, err := handler.ReadTag(device, testTag); err == nil {
			logger.Debug("Diagnostic read successful", zap.Any("value", value))
		} else {
			logger.Debug("Diagnostic read failed", zap.Error(err))
		}

		// Write operations
		if err := handler.WriteTag(device, testTag, int32(i)); err == nil {
			logger.Debug("Diagnostic write successful", zap.Int("value", i))
		} else {
			logger.Debug("Diagnostic write failed", zap.Error(err))
		}

		// Ping operations
		if err := handler.Ping(device); err != nil {
			logger.Debug("Diagnostic ping failed", zap.Error(err))
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Get comprehensive diagnostics
	logger.Info("Retrieving comprehensive diagnostics...")
	if diag, err := handler.GetDiagnostics(device); err == nil {
		logger.Info("Basic Diagnostics",
			zap.Bool("is_healthy", diag.IsHealthy),
			zap.Time("last_communication", diag.LastCommunication),
			zap.Duration("response_time", diag.ResponseTime),
			zap.Uint64("error_count", diag.ErrorCount),
			zap.Float64("success_rate", diag.SuccessRate),
			zap.Duration("connection_uptime", diag.ConnectionUptime),
		)

		if protocolDiag, ok := diag.ProtocolDiagnostics.(map[string]interface{}); ok {
			logger.Info("Protocol-specific Diagnostics",
				zap.Any("session_id", protocolDiag["session_id"]),
				zap.Any("connection_id", protocolDiag["connection_id"]),
				zap.Any("vendor_id", protocolDiag["vendor_id"]),
				zap.Any("device_type", protocolDiag["device_type"]),
				zap.Any("product_code", protocolDiag["product_code"]),
				zap.Any("product_name", protocolDiag["product_name"]),
			)
		}

		if len(diag.Errors) > 0 {
			logger.Info("Recent Errors")
			for i, err := range diag.Errors {
				logger.Info("Error",
					zap.Int("index", i),
					zap.Time("timestamp", err.Timestamp),
					zap.String("error_code", err.ErrorCode),
					zap.String("description", err.Description),
					zap.String("operation", err.Operation),
				)
			}
		}
	} else {
		logger.Error("Failed to get diagnostics", zap.Error(err))
	}

	// Test error handling
	logger.Info("Testing error handling with invalid operations...")

	// Try to read a non-existent tag
	invalidTag := &protocols.Tag{
		ID:       "invalid_tag",
		Name:     "Invalid Tag",
		Address:  "NonExistentTag",
		DataType: string(protocols.DataTypeInt32),
		Writable: false,
	}

	if _, err := handler.ReadTag(device, invalidTag); err != nil {
		logger.Info("Expected error for invalid tag", zap.Error(err))
	}

	// Try to write to a non-writable tag
	readOnlyTag := &protocols.Tag{
		ID:       "readonly_tag",
		Name:     "Read Only Tag",
		Address:  *tagName,
		DataType: string(protocols.DataTypeInt32),
		Writable: false,
	}

	if err := handler.WriteTag(device, readOnlyTag, int32(123)); err != nil {
		logger.Info("Expected error for read-only tag", zap.Error(err))
	}

	// Get diagnostics again to see error tracking
	if diag, err := handler.GetDiagnostics(device); err == nil {
		logger.Info("Updated Diagnostics After Errors",
			zap.Uint64("error_count", diag.ErrorCount),
			zap.Float64("success_rate", diag.SuccessRate),
		)
	}
}
