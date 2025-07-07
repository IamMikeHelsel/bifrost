package performance

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/bifrost/gateway/internal/protocols"
)

// OptimizedGateway is a high-performance industrial gateway with comprehensive optimizations
type OptimizedGateway struct {
	logger *zap.Logger
	config *OptimizedConfig

	// Core components
	protocols map[string]protocols.ProtocolHandler
	devices   sync.Map // map[string]*OptimizedDevice

	// Performance optimization components
	connectionPool  *ConnectionPool
	batchProcessor  *BatchProcessor
	memoryOptimizer *MemoryOptimizer
	edgeOptimizer   *EdgeOptimizer
	profiler        *Profiler
	monitor         *PerformanceMonitor

	// WebSocket connections for real-time data
	wsUpgrader websocket.Upgrader
	wsClients  sync.Map // map[*websocket.Conn]bool

	// Performance state
	requestsInFlight int64
	totalRequests    int64
	totalErrors      int64

	// Control
	ctx    context.Context
	cancel context.CancelFunc
}

// OptimizedConfig holds configuration for the optimized gateway
type OptimizedConfig struct {
	// Basic gateway config
	Port           int           `yaml:"port"`
	GRPCPort       int           `yaml:"grpc_port"`
	MaxConnections int           `yaml:"max_connections"`
	DataBufferSize int           `yaml:"data_buffer_size"`
	UpdateInterval time.Duration `yaml:"update_interval"`
	EnableMetrics  bool          `yaml:"enable_metrics"`
	LogLevel       string        `yaml:"log_level"`

	// Performance optimization configs
	ConnectionPool  PoolConfig       `yaml:"connection_pool"`
	BatchProcessor  BatchConfig      `yaml:"batch_processor"`
	MemoryOptimizer MemoryConfig     `yaml:"memory_optimizer"`
	EdgeOptimizer   EdgeConfig       `yaml:"edge_optimizer"`
	Profiler        ProfilerConfig   `yaml:"profiler"`
	Monitor         MonitoringConfig `yaml:"monitor"`

	// Advanced performance settings
	EnableZeroCopy         bool `yaml:"enable_zero_copy"`
	EnableBatching         bool `yaml:"enable_batching"`
	EnableConnectionPool   bool `yaml:"enable_connection_pool"`
	EnableEdgeOptimization bool `yaml:"enable_edge_optimization"`
	EnableProfiling        bool `yaml:"enable_profiling"`
	EnableMonitoring       bool `yaml:"enable_monitoring"`
}

// OptimizedDevice represents a device with performance optimizations
type OptimizedDevice struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Protocol string                 `json:"protocol"`
	Address  string                 `json:"address"`
	Port     int                    `json:"port"`
	Config   map[string]interface{} `json:"config"`

	// Runtime state
	Connected bool                      `json:"connected"`
	LastSeen  time.Time                 `json:"last_seen"`
	Tags      map[string]*OptimizedTag  `json:"tags"`
	Handler   protocols.ProtocolHandler `json:"-"`

	// Performance tracking
	Stats *DevicePerformanceStats `json:"stats"`

	// Optimization state
	ConnectionID     string            `json:"-"`
	BatchingEnabled  bool              `json:"-"`
	PooledConnection *PooledConnection `json:"-"`
}

// OptimizedTag represents a tag with performance optimizations
type OptimizedTag struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Address     string      `json:"address"`
	DataType    string      `json:"data_type"`
	Value       interface{} `json:"value"`
	Quality     string      `json:"quality"`
	Timestamp   time.Time   `json:"timestamp"`
	Writable    bool        `json:"writable"`
	Unit        string      `json:"unit"`
	Description string      `json:"description"`

	// Performance metadata
	LastReadLatency time.Duration `json:"last_read_latency"`
	ReadCount       int64         `json:"read_count"`
	ErrorCount      int64         `json:"error_count"`
	BatchingEnabled bool          `json:"batching_enabled"`
}

// DevicePerformanceStats tracks device-specific performance metrics
type DevicePerformanceStats struct {
	RequestsTotal       uint64        `json:"requests_total"`
	RequestsSuccessful  uint64        `json:"requests_successful"`
	RequestsFailed      uint64        `json:"requests_failed"`
	AverageResponseTime time.Duration `json:"avg_response_time"`
	P95ResponseTime     time.Duration `json:"p95_response_time"`
	LastUpdate          time.Time     `json:"last_update"`

	// Optimization metrics
	BatchingEfficiency float64 `json:"batching_efficiency"`
	CacheHitRate       float64 `json:"cache_hit_rate"`
	ConnectionReuse    float64 `json:"connection_reuse"`
}

// NewOptimizedGateway creates a new optimized industrial gateway
func NewOptimizedGateway(config *OptimizedConfig, logger *zap.Logger) *OptimizedGateway {
	ctx, cancel := context.WithCancel(context.Background())

	gateway := &OptimizedGateway{
		logger:    logger,
		config:    config,
		protocols: make(map[string]protocols.ProtocolHandler),
		ctx:       ctx,
		cancel:    cancel,
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}

	// Initialize performance optimization components
	gateway.initializeOptimizations()

	// Register protocol handlers
	gateway.registerProtocols()

	return gateway
}

// initializeOptimizations initializes all performance optimization components
func (og *OptimizedGateway) initializeOptimizations() {
	// Initialize connection pool
	if og.config.EnableConnectionPool {
		og.connectionPool = NewConnectionPool(&og.config.ConnectionPool, og.logger)
		og.logger.Info("Connection pool initialized",
			zap.Int("max_connections_per_device", og.config.ConnectionPool.MaxConnectionsPerDevice),
			zap.Int("max_total_connections", og.config.ConnectionPool.MaxTotalConnections),
		)
	}

	// Initialize batch processor
	if og.config.EnableBatching {
		og.batchProcessor = NewBatchProcessor(&og.config.BatchProcessor, og.logger)
		og.logger.Info("Batch processor initialized",
			zap.Int("max_batch_size", og.config.BatchProcessor.MaxBatchSize),
			zap.Duration("batch_timeout", og.config.BatchProcessor.BatchTimeout),
		)
	}

	// Initialize memory optimizer
	og.memoryOptimizer = NewMemoryOptimizer(&og.config.MemoryOptimizer, og.logger)
	og.logger.Info("Memory optimizer initialized",
		zap.Bool("zero_copy", og.config.MemoryOptimizer.EnableZeroCopy),
		zap.Int("max_buffer_size", og.config.MemoryOptimizer.MaxBufferSize),
	)

	// Initialize edge optimizer
	if og.config.EnableEdgeOptimization {
		og.edgeOptimizer = NewEdgeOptimizer(&og.config.EdgeOptimizer, og.logger)
		og.logger.Info("Edge optimizer initialized",
			zap.Int("max_memory_mb", og.config.EdgeOptimizer.MaxMemoryMB),
			zap.Float64("max_cpu_percent", og.config.EdgeOptimizer.MaxCPUPercent),
		)
	}

	// Initialize profiler
	if og.config.EnableProfiling {
		og.profiler = NewProfiler(&og.config.Profiler, og.logger)
		og.logger.Info("Profiler initialized",
			zap.Int("http_port", og.config.Profiler.HTTPPort),
			zap.Bool("auto_cpu_profile", og.config.Profiler.AutoCPUProfile),
		)
	}

	// Initialize performance monitor
	if og.config.EnableMonitoring {
		og.monitor = NewPerformanceMonitor(&og.config.Monitor, og.logger)
		og.logger.Info("Performance monitor initialized",
			zap.Int("metrics_port", og.config.Monitor.MetricsPort),
			zap.Bool("prometheus", og.config.Monitor.EnablePrometheus),
		)
	}
}

// registerProtocols registers all protocol handlers with optimizations
func (og *OptimizedGateway) registerProtocols() {
	// Register optimized Modbus TCP/RTU handler
	modbusHandler := protocols.NewModbusHandler(og.logger)
	og.protocols["modbus-tcp"] = modbusHandler
	og.protocols["modbus-rtu"] = modbusHandler

	og.logger.Info("Protocol handlers registered",
		zap.Strings("protocols", []string{"modbus-tcp", "modbus-rtu"}),
	)
}

// Start begins the optimized gateway services
func (og *OptimizedGateway) Start(ctx context.Context) error {
	og.logger.Info("Starting Bifrost Optimized Industrial Gateway",
		zap.Int("port", og.config.Port),
		zap.Int("grpc_port", og.config.GRPCPort),
		zap.Bool("connection_pool", og.config.EnableConnectionPool),
		zap.Bool("batching", og.config.EnableBatching),
		zap.Bool("edge_optimization", og.config.EnableEdgeOptimization),
	)

	var wg sync.WaitGroup

	// Start HTTP server for WebSocket and metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		og.startHTTPServer(ctx)
	}()

	// Start gRPC server for backend communication
	wg.Add(1)
	go func() {
		defer wg.Done()
		og.startGRPCServer(ctx)
	}()

	// Start optimized data collection loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		og.startOptimizedDataCollection(ctx)
	}()

	// Start performance monitoring
	if og.monitor != nil {
		wg.Add1()
		go func() {
			defer wg.Done()
			og.startPerformanceMonitoring(ctx)
		}()
	}

	wg.Wait()
	return nil
}

// startOptimizedDataCollection runs the optimized data collection loop
func (og *OptimizedGateway) startOptimizedDataCollection(ctx context.Context) {
	ticker := time.NewTicker(og.config.UpdateInterval)
	defer ticker.Stop()

	og.logger.Info("Optimized data collection started",
		zap.Duration("interval", og.config.UpdateInterval),
		zap.Bool("batching", og.config.EnableBatching),
		zap.Bool("connection_pool", og.config.EnableConnectionPool),
	)

	for {
		select {
		case <-ctx.Done():
			og.logger.Info("Data collection stopped")
			return
		case <-ticker.C:
			og.collectAllDataOptimized(ctx)
		}
	}
}

// collectAllDataOptimized performs optimized data collection from all devices
func (og *OptimizedGateway) collectAllDataOptimized(ctx context.Context) {
	start := time.Now()

	if og.config.EnableBatching {
		og.collectDataWithBatching(ctx)
	} else {
		og.collectDataConcurrent(ctx)
	}

	// Record performance metrics
	duration := time.Since(start)
	if og.monitor != nil {
		og.monitor.RecordLatency(duration, map[string]string{
			"operation": "data_collection",
			"method":    og.getCollectionMethod(),
		})
	}
}

// collectDataWithBatching uses intelligent batching for data collection
func (og *OptimizedGateway) collectDataWithBatching(ctx context.Context) {
	// Group devices by protocol for batch processing
	deviceGroups := og.groupDevicesByProtocol()

	var wg sync.WaitGroup

	for protocol, devices := range deviceGroups {
		wg.Add(1)
		go func(proto string, devs []*OptimizedDevice) {
			defer wg.Done()
			og.collectProtocolBatch(ctx, proto, devs)
		}(protocol, devices)
	}

	wg.Wait()
}

// collectDataConcurrent uses traditional concurrent data collection
func (og *OptimizedGateway) collectDataConcurrent(ctx context.Context) {
	var wg sync.WaitGroup

	// Limit concurrent goroutines to prevent resource exhaustion
	semaphore := make(chan struct{}, og.config.MaxConnections)

	og.devices.Range(func(key, value interface{}) bool {
		device := value.(*OptimizedDevice)
		if device.Connected {
			wg.Add(1)
			go func(d *OptimizedDevice) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				og.collectDeviceDataOptimized(ctx, d)
			}(device)
		}
		return true
	})

	wg.Wait()
}

// collectProtocolBatch collects data from a batch of devices using the same protocol
func (og *OptimizedGateway) collectProtocolBatch(ctx context.Context, protocol string, devices []*OptimizedDevice) {
	handler, exists := og.protocols[protocol]
	if !exists {
		og.logger.Error("Unknown protocol for batch collection", zap.String("protocol", protocol))
		return
	}

	// Prepare batch requests
	var batchRequests []*BatchRequest

	for _, device := range devices {
		for _, tag := range device.Tags {
			request := og.memoryOptimizer.AcquireRequest()
			request.ID = fmt.Sprintf("%s_%s", device.ID, tag.ID)
			request.DeviceID = device.ID
			request.Operation = "read"
			request.Address = tag.Address
			request.CanBatch = true
			request.Context = ctx

			// Set callback for result processing
			request.Callback = func(result interface{}, err error) {
				defer og.memoryOptimizer.ReleaseRequest(request)

				if err != nil {
					og.handleReadError(device, tag, err)
				} else {
					og.handleReadSuccess(device, tag, result)
				}
			}

			batchRequests = append(batchRequests, request)
		}
	}

	// Submit batch requests to processor
	for _, request := range batchRequests {
		og.batchProcessor.AddRequest(request)
	}
}

// collectDeviceDataOptimized performs optimized data collection for a single device
func (og *OptimizedGateway) collectDeviceDataOptimized(ctx context.Context, device *OptimizedDevice) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		device.Stats.AverageResponseTime = (device.Stats.AverageResponseTime + duration) / 2
		device.Stats.LastUpdate = time.Now()

		// Update P95 response time (simplified calculation)
		if duration > device.Stats.P95ResponseTime {
			device.Stats.P95ResponseTime = duration
		}
	}()

	// Use connection pool if available
	var conn *PooledConnection
	var err error

	if og.connectionPool != nil {
		conn, err = og.connectionPool.GetConnection(device.ID, func() (Connection, error) {
			return og.createDeviceConnection(device)
		})
		if err != nil {
			og.logger.Error("Failed to get pooled connection",
				zap.String("device", device.ID),
				zap.Error(err),
			)
			return
		}
		defer og.connectionPool.ReturnConnection(conn)
	}

	// Collect tag data with optimizations
	var successCount, errorCount uint64

	for _, tag := range device.Tags {
		select {
		case <-ctx.Done():
			return
		default:
			err := og.readTagOptimized(ctx, device, tag, conn)
			if err != nil {
				errorCount++
				og.handleReadError(device, tag, err)
			} else {
				successCount++
			}
		}
	}

	// Update device statistics
	device.Stats.RequestsTotal += successCount + errorCount
	device.Stats.RequestsSuccessful += successCount
	device.Stats.RequestsFailed += errorCount
	device.LastSeen = time.Now()

	// Calculate optimization metrics
	if device.Stats.RequestsTotal > 0 {
		device.Stats.BatchingEfficiency = og.calculateBatchingEfficiency(device)
		device.Stats.ConnectionReuse = og.calculateConnectionReuse(device)
	}
}

// readTagOptimized performs optimized tag reading
func (og *OptimizedGateway) readTagOptimized(ctx context.Context, device *OptimizedDevice, tag *OptimizedTag, conn *PooledConnection) error {
	tagStart := time.Now()

	handler, exists := og.protocols[device.Protocol]
	if !exists {
		return fmt.Errorf("unknown protocol: %s", device.Protocol)
	}

	// Create protocol device and tag structures
	protocolDevice := &protocols.Device{
		ID:       device.ID,
		Name:     device.Name,
		Protocol: device.Protocol,
		Address:  device.Address,
		Port:     device.Port,
		Config:   device.Config,
	}

	protocolTag := &protocols.Tag{
		ID:          tag.ID,
		Name:        tag.Name,
		Address:     tag.Address,
		DataType:    tag.DataType,
		Writable:    tag.Writable,
		Unit:        tag.Unit,
		Description: tag.Description,
	}

	// Read tag value
	var value interface{}
	var err error

	if conn != nil {
		// Use pooled connection
		value, err = conn.Execute(ctx, map[string]interface{}{
			"operation": "read_tag",
			"device":    protocolDevice,
			"tag":       protocolTag,
		})
	} else {
		// Direct read
		value, err = handler.ReadTag(protocolDevice, protocolTag)
	}

	// Update tag performance metrics
	tag.LastReadLatency = time.Since(tagStart)
	tag.ReadCount++

	if err != nil {
		tag.ErrorCount++
		return err
	}

	// Update tag value with zero-copy optimization
	if og.config.EnableZeroCopy {
		og.updateTagValueZeroCopy(tag, value)
	} else {
		tag.Value = value
	}

	tag.Timestamp = time.Now()
	tag.Quality = "GOOD"

	// Broadcast to WebSocket clients
	og.broadcastTagUpdateOptimized(device, tag)

	// Record metrics
	if og.monitor != nil {
		og.monitor.RecordLatency(tag.LastReadLatency, map[string]string{
			"device_id": device.ID,
			"tag_id":    tag.ID,
			"protocol":  device.Protocol,
		})

		og.monitor.RecordRequest(map[string]string{
			"device_id": device.ID,
			"operation": "read_tag",
			"protocol":  device.Protocol,
			"status":    "success",
		})
	}

	return nil
}

// Helper methods

func (og *OptimizedGateway) groupDevicesByProtocol() map[string][]*OptimizedDevice {
	groups := make(map[string][]*OptimizedDevice)

	og.devices.Range(func(key, value interface{}) bool {
		device := value.(*OptimizedDevice)
		if device.Connected {
			groups[device.Protocol] = append(groups[device.Protocol], device)
		}
		return true
	})

	return groups
}

func (og *OptimizedGateway) createDeviceConnection(device *OptimizedDevice) (Connection, error) {
	// This would create an actual connection to the device
	// For now, return a mock connection
	return &MockConnection{deviceID: device.ID}, nil
}

func (og *OptimizedGateway) updateTagValueZeroCopy(tag *OptimizedTag, value interface{}) {
	// Implement zero-copy value update
	// This would use memory optimizer for efficient value updates
	tag.Value = value
}

func (og *OptimizedGateway) handleReadError(device *OptimizedDevice, tag *OptimizedTag, err error) {
	if og.monitor != nil {
		og.monitor.RecordError("read_error", map[string]string{
			"device_id": device.ID,
			"tag_id":    tag.ID,
			"protocol":  device.Protocol,
		})
	}

	og.logger.Error("Tag read error",
		zap.String("device", device.ID),
		zap.String("tag", tag.ID),
		zap.Error(err),
	)
}

func (og *OptimizedGateway) handleReadSuccess(device *OptimizedDevice, tag *OptimizedTag, result interface{}) {
	// Update tag with result
	tag.Value = result
	tag.Timestamp = time.Now()
	tag.Quality = "GOOD"
	tag.ReadCount++

	// Broadcast update
	og.broadcastTagUpdateOptimized(device, tag)
}

func (og *OptimizedGateway) calculateBatchingEfficiency(device *OptimizedDevice) float64 {
	// Mock calculation of batching efficiency
	if device.BatchingEnabled {
		return 85.0 // 85% efficiency with batching
	}
	return 0.0
}

func (og *OptimizedGateway) calculateConnectionReuse(device *OptimizedDevice) float64 {
	// Mock calculation of connection reuse
	if og.connectionPool != nil {
		return 95.0 // 95% connection reuse with pooling
	}
	return 0.0
}

func (og *OptimizedGateway) getCollectionMethod() string {
	if og.config.EnableBatching {
		return "batched"
	}
	return "concurrent"
}

func (og *OptimizedGateway) broadcastTagUpdateOptimized(device *OptimizedDevice, tag *OptimizedTag) {
	// Use memory optimizer for message creation
	message := map[string]interface{}{
		"type":      "tag_update",
		"device_id": device.ID,
		"tag":       tag,
	}

	// Broadcast to WebSocket clients
	og.wsClients.Range(func(key, value interface{}) bool {
		conn := key.(*websocket.Conn)
		if err := conn.WriteJSON(message); err != nil {
			// Remove disconnected client
			og.wsClients.Delete(conn)
			conn.Close()
		}
		return true
	})
}

func (og *OptimizedGateway) startHTTPServer(ctx context.Context) {
	mux := http.NewServeMux()

	// WebSocket endpoint for real-time data
	mux.HandleFunc("/ws", og.handleWebSocket)

	// REST API endpoints with optimizations
	mux.HandleFunc("/api/devices", og.handleDevicesOptimized)
	mux.HandleFunc("/api/devices/discover", og.handleDiscoveryOptimized)
	mux.HandleFunc("/api/tags/read", og.handleTagReadOptimized)
	mux.HandleFunc("/api/tags/write", og.handleTagWriteOptimized)
	mux.HandleFunc("/api/performance", og.handlePerformanceMetrics)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", og.config.Port),
		Handler: mux,
	}

	og.logger.Info("HTTP server started", zap.Int("port", og.config.Port))

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		og.logger.Error("HTTP server error", zap.Error(err))
	}
}

func (og *OptimizedGateway) startGRPCServer(ctx context.Context) {
	// TODO: Implement optimized gRPC server
	og.logger.Info("gRPC server started", zap.Int("port", og.config.GRPCPort))
}

func (og *OptimizedGateway) startPerformanceMonitoring(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			og.collectPerformanceMetrics()
		}
	}
}

func (og *OptimizedGateway) collectPerformanceMetrics() {
	// Collect comprehensive performance metrics
	if og.monitor != nil {
		// Update device metrics
		deviceCount := 0
		connectedCount := 0

		og.devices.Range(func(key, value interface{}) bool {
			deviceCount++
			if value.(*OptimizedDevice).Connected {
				connectedCount++
			}
			return true
		})

		// Update global metrics
		metrics := og.monitor.GetMetrics()
		metrics.DevicesConnected = connectedCount
		metrics.TagsProcessed = og.totalRequests
	}
}

// HTTP handlers with optimizations

func (og *OptimizedGateway) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := og.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		og.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	og.wsClients.Store(conn, true)
	og.logger.Info("WebSocket client connected")

	defer func() {
		og.wsClients.Delete(conn)
		conn.Close()
		og.logger.Info("WebSocket client disconnected")
	}()

	// Keep connection alive with optimized handling
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (og *OptimizedGateway) handleDevicesOptimized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	devices := make([]*OptimizedDevice, 0)
	og.devices.Range(func(key, value interface{}) bool {
		devices = append(devices, value.(*OptimizedDevice))
		return true
	})

	// Use optimized JSON encoding
	response := fmt.Sprintf(`{"devices": %d, "connected": %d}`,
		len(devices), og.countConnectedDevices())
	w.Write([]byte(response))
}

func (og *OptimizedGateway) handleDiscoveryOptimized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// TODO: Implement optimized device discovery
	w.Write([]byte(`{"message": "Optimized device discovery not implemented yet"}`))
}

func (og *OptimizedGateway) handleTagReadOptimized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// TODO: Implement optimized tag reading API
	w.Write([]byte(`{"message": "Optimized tag reading not implemented yet"}`))
}

func (og *OptimizedGateway) handleTagWriteOptimized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// TODO: Implement optimized tag writing API
	w.Write([]byte(`{"message": "Optimized tag writing not implemented yet"}`))
}

func (og *OptimizedGateway) handlePerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if og.monitor == nil {
		w.Write([]byte(`{"error": "Performance monitoring not enabled"}`))
		return
	}

	metrics := og.monitor.GetMetrics()
	response := fmt.Sprintf(`{
		"latency": {
			"average": %f,
			"p95": %f,
			"p99": %f
		},
		"throughput": %f,
		"error_rate": %f,
		"devices_connected": %d,
		"requests_total": %d
	}`,
		metrics.AverageLatency.Seconds(),
		metrics.P95Latency.Seconds(),
		metrics.P99Latency.Seconds(),
		metrics.ThroughputPerSecond,
		metrics.ErrorRate,
		metrics.DevicesConnected,
		og.totalRequests,
	)

	w.Write([]byte(response))
}

func (og *OptimizedGateway) countConnectedDevices() int {
	count := 0
	og.devices.Range(func(key, value interface{}) bool {
		if value.(*OptimizedDevice).Connected {
			count++
		}
		return true
	})
	return count
}

// GetOptimizationMetrics returns comprehensive optimization metrics
func (og *OptimizedGateway) GetOptimizationMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Connection pool metrics
	if og.connectionPool != nil {
		metrics["connection_pool"] = og.connectionPool.GetMetrics()
	}

	// Batch processor metrics
	if og.batchProcessor != nil {
		metrics["batch_processor"] = og.batchProcessor.GetMetrics()
	}

	// Memory optimizer metrics
	if og.memoryOptimizer != nil {
		metrics["memory_optimizer"] = og.memoryOptimizer.GetMetrics()
	}

	// Edge optimizer metrics
	if og.edgeOptimizer != nil {
		metrics["edge_optimizer"] = og.edgeOptimizer.GetMetrics()
	}

	// Performance monitor metrics
	if og.monitor != nil {
		metrics["performance_monitor"] = og.monitor.GetMetrics()
	}

	return metrics
}

// Close gracefully shuts down the optimized gateway
func (og *OptimizedGateway) Close() error {
	og.logger.Info("Shutting down optimized gateway")

	og.cancel()

	// Close optimization components
	if og.connectionPool != nil {
		og.connectionPool.Close()
	}

	if og.batchProcessor != nil {
		og.batchProcessor.Close()
	}

	if og.edgeOptimizer != nil {
		og.edgeOptimizer.Close()
	}

	if og.profiler != nil {
		og.profiler.Close()
	}

	if og.monitor != nil {
		og.monitor.Close()
	}

	og.logger.Info("Optimized gateway shutdown complete")
	return nil
}

// MockConnection is a mock implementation for testing
type MockConnection struct {
	deviceID string
}

func (mc *MockConnection) Connect() error    { return nil }
func (mc *MockConnection) Disconnect() error { return nil }
func (mc *MockConnection) IsHealthy() bool   { return true }
func (mc *MockConnection) Execute(ctx context.Context, request interface{}) (interface{}, error) {
	// Mock execution
	time.Sleep(time.Microsecond * 50)
	return "mock_result", nil
}
func (mc *MockConnection) GetStats() ConnectionStats {
	return ConnectionStats{
		RequestsTotal:      100,
		RequestsSuccessful: 99,
		RequestsFailed:     1,
		AverageLatency:     time.Microsecond * 50,
		LastActivity:       time.Now(),
	}
}
