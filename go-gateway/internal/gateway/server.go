package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/bifrost/gateway/internal/protocols"
)

// IndustrialGateway is the main server handling multiple industrial protocols
type IndustrialGateway struct {
	logger    *zap.Logger
	devices   sync.Map // map[string]*Device
	protocols map[string]protocols.ProtocolHandler
	
	// Performance metrics
	metrics struct {
		connectionsTotal    prometheus.Counter
		dataPointsProcessed prometheus.Counter
		errorRate          prometheus.Counter
		responseTime       prometheus.Histogram
	}
	
	// WebSocket connections for real-time data
	wsUpgrader websocket.Upgrader
	wsClients  sync.Map // map[*websocket.Conn]bool
	
	// Configuration
	config *Config
}

type Config struct {
	Port              int           `yaml:"port"`
	GRPCPort         int           `yaml:"grpc_port"`
	MaxConnections   int           `yaml:"max_connections"`
	DataBufferSize   int           `yaml:"data_buffer_size"`
	UpdateInterval   time.Duration `yaml:"update_interval"`
	EnableMetrics    bool          `yaml:"enable_metrics"`
	LogLevel         string        `yaml:"log_level"`
}

type Device struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Protocol string                 `json:"protocol"`
	Address  string                 `json:"address"`
	Port     int                    `json:"port"`
	Config   map[string]interface{} `json:"config"`
	
	// Runtime state
	Connected    bool                      `json:"connected"`
	LastSeen     time.Time                `json:"last_seen"`
	Tags         map[string]*Tag          `json:"tags"`
	Handler      protocols.ProtocolHandler `json:"-"`
	
	// Performance tracking
	Stats struct {
		RequestsTotal      uint64    `json:"requests_total"`
		RequestsSuccessful uint64    `json:"requests_successful"`
		RequestsFailed     uint64    `json:"requests_failed"`
		AverageResponseTime float64  `json:"avg_response_time"`
		LastUpdate         time.Time `json:"last_update"`
	} `json:"stats"`
}

type Tag struct {
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
}

// NewIndustrialGateway creates a new gateway instance
func NewIndustrialGateway(config *Config, logger *zap.Logger) *IndustrialGateway {
	gateway := &IndustrialGateway{
		logger:    logger,
		protocols: make(map[string]protocols.ProtocolHandler),
		config:    config,
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}
	
	// Initialize metrics
	gateway.initMetrics()
	
	// Register protocol handlers
	gateway.registerProtocols()
	
	return gateway
}

func (g *IndustrialGateway) initMetrics() {
	g.metrics.connectionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bifrost_connections_total",
		Help: "Total number of device connections",
	})
	
	g.metrics.dataPointsProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bifrost_data_points_processed_total",
		Help: "Total number of data points processed",
	})
	
	g.metrics.errorRate = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bifrost_errors_total",
		Help: "Total number of errors",
	})
	
	g.metrics.responseTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "bifrost_response_time_seconds",
		Help:    "Response time for device operations",
		Buckets: prometheus.DefBuckets,
	})
	
	// Register metrics
	prometheus.MustRegister(
		g.metrics.connectionsTotal,
		g.metrics.dataPointsProcessed,
		g.metrics.errorRate,
		g.metrics.responseTime,
	)
}

func (g *IndustrialGateway) registerProtocols() {
	// Register Modbus TCP/RTU handler
	modbusHandler := protocols.NewModbusHandler(g.logger)
	g.protocols["modbus-tcp"] = modbusHandler
	g.protocols["modbus-rtu"] = modbusHandler
	
	// TODO: Register OPC UA handler
	// opcuaHandler := protocols.NewOPCUAHandler(g.logger)
	// g.protocols["opcua"] = opcuaHandler
	
	// TODO: Add Ethernet/IP, S7, etc.
}

// Start begins the gateway services
func (g *IndustrialGateway) Start(ctx context.Context) error {
	g.logger.Info("Starting Bifrost Industrial Gateway",
		zap.Int("port", g.config.Port),
		zap.Int("grpc_port", g.config.GRPCPort),
	)
	
	var wg sync.WaitGroup
	
	// Start HTTP server for WebSocket and metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		g.startHTTPServer(ctx)
	}()
	
	// Start gRPC server for backend communication
	wg.Add(1)
	go func() {
		defer wg.Done()
		g.startGRPCServer(ctx)
	}()
	
	// Start data collection loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		g.startDataCollection(ctx)
	}()
	
	wg.Wait()
	return nil
}

func (g *IndustrialGateway) startHTTPServer(ctx context.Context) {
	mux := http.NewServeMux()
	
	// WebSocket endpoint for real-time data
	mux.HandleFunc("/ws", g.handleWebSocket)
	
	// REST API endpoints
	mux.HandleFunc("/api/devices", g.handleDevices)
	mux.HandleFunc("/api/devices/discover", g.handleDiscovery)
	mux.HandleFunc("/api/tags/read", g.handleTagRead)
	mux.HandleFunc("/api/tags/write", g.handleTagWrite)
	
	// Metrics endpoint
	if g.config.EnableMetrics {
		mux.Handle("/metrics", promhttp.Handler())
	}
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", g.config.Port),
		Handler: mux,
	}
	
	g.logger.Info("HTTP server started", zap.Int("port", g.config.Port))
	
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()
	
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		g.logger.Error("HTTP server error", zap.Error(err))
	}
}

func (g *IndustrialGateway) startGRPCServer(ctx context.Context) {
	// TODO: Implement gRPC server for backend API
	g.logger.Info("gRPC server started", zap.Int("port", g.config.GRPCPort))
}

// startDataCollection runs the main data collection loop
func (g *IndustrialGateway) startDataCollection(ctx context.Context) {
	ticker := time.NewTicker(g.config.UpdateInterval)
	defer ticker.Stop()
	
	g.logger.Info("Data collection started", zap.Duration("interval", g.config.UpdateInterval))
	
	for {
		select {
		case <-ctx.Done():
			g.logger.Info("Data collection stopped")
			return
		case <-ticker.C:
			g.collectAllData(ctx)
		}
	}
}

func (g *IndustrialGateway) collectAllData(ctx context.Context) {
	var wg sync.WaitGroup
	
	// Collect data from all connected devices concurrently
	g.devices.Range(func(key, value interface{}) bool {
		device := value.(*Device)
		if device.Connected {
			wg.Add(1)
			go func(d *Device) {
				defer wg.Done()
				g.collectDeviceData(ctx, d)
			}(device)
		}
		return true
	})
	
	wg.Wait()
}

func (g *IndustrialGateway) collectDeviceData(ctx context.Context, device *Device) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		g.metrics.responseTime.Observe(duration.Seconds())
	}()
	
	handler, exists := g.protocols[device.Protocol]
	if !exists {
		g.logger.Error("Unknown protocol", zap.String("protocol", device.Protocol))
		return
	}
	
	// Create protocols.Device from gateway.Device
	protocolDevice := &protocols.Device{
		ID:       device.ID,
		Name:     device.Name,
		Protocol: device.Protocol,
		Address:  device.Address,
		Port:     device.Port,
		Config:   device.Config,
	}
	
	// Read all tags for this device
	for _, tag := range device.Tags {
		select {
		case <-ctx.Done():
			return
		default:
			// Create protocols.Tag from gateway.Tag
			protocolTag := &protocols.Tag{
				ID:       tag.ID,
				Name:     tag.Name,
				Address:  tag.Address,
				DataType: tag.DataType,
				Writable: tag.Writable,
				Unit:     tag.Unit,
				Description: tag.Description,
			}
			
			value, err := handler.ReadTag(protocolDevice, protocolTag)
			if err != nil {
				g.metrics.errorRate.Inc()
				device.Stats.RequestsFailed++
				g.logger.Error("Failed to read tag",
					zap.String("device", device.ID),
					zap.String("tag", tag.ID),
					zap.Error(err),
				)
				continue
			}
			
			// Update tag value
			tag.Value = value
			tag.Timestamp = time.Now()
			tag.Quality = "GOOD"
			
			device.Stats.RequestsSuccessful++
			g.metrics.dataPointsProcessed.Inc()
			
			// Broadcast to WebSocket clients
			g.broadcastTagUpdate(device, tag)
		}
	}
	
	device.LastSeen = time.Now()
	device.Stats.LastUpdate = time.Now()
}

// ConnectDevice establishes connection to an industrial device
func (g *IndustrialGateway) ConnectDevice(ctx context.Context, device *Device) error {
	handler, exists := g.protocols[device.Protocol]
	if !exists {
		return fmt.Errorf("unsupported protocol: %s", device.Protocol)
	}
	
	// Create protocols.Device from gateway.Device
	protocolDevice := &protocols.Device{
		ID:       device.ID,
		Name:     device.Name,
		Protocol: device.Protocol,
		Address:  device.Address,
		Port:     device.Port,
		Config:   device.Config,
	}
	
	// Attempt connection
	if err := handler.Connect(protocolDevice); err != nil {
		g.metrics.errorRate.Inc()
		return fmt.Errorf("failed to connect to device %s: %w", device.ID, err)
	}
	
	device.Connected = true
	device.LastSeen = time.Now()
	g.metrics.connectionsTotal.Inc()
	
	// Store device
	g.devices.Store(device.ID, device)
	
	g.logger.Info("Device connected",
		zap.String("id", device.ID),
		zap.String("protocol", device.Protocol),
		zap.String("address", device.Address),
	)
	
	return nil
}

// DisconnectDevice closes connection to a device
func (g *IndustrialGateway) DisconnectDevice(deviceID string) error {
	deviceInterface, exists := g.devices.Load(deviceID)
	if !exists {
		return fmt.Errorf("device not found: %s", deviceID)
	}
	
	device := deviceInterface.(*Device)
	
	if device.Handler != nil {
		// Create protocols.Device from gateway.Device
		protocolDevice := &protocols.Device{
			ID:       device.ID,
			Name:     device.Name,
			Protocol: device.Protocol,
			Address:  device.Address,
			Port:     device.Port,
			Config:   device.Config,
		}
		
		if err := device.Handler.Disconnect(protocolDevice); err != nil {
			g.logger.Error("Error disconnecting device", zap.Error(err))
		}
	}
	
	device.Connected = false
	g.devices.Store(deviceID, device)
	
	g.logger.Info("Device disconnected", zap.String("id", deviceID))
	return nil
}

func (g *IndustrialGateway) broadcastTagUpdate(device *Device, tag *Tag) {
	message := map[string]interface{}{
		"type":      "tag_update",
		"device_id": device.ID,
		"tag":       tag,
	}
	
	// Broadcast to all WebSocket clients
	g.wsClients.Range(func(key, value interface{}) bool {
		conn := key.(*websocket.Conn)
		if err := conn.WriteJSON(message); err != nil {
			// Remove disconnected client
			g.wsClients.Delete(conn)
			conn.Close()
		}
		return true
	})
}

// GetStats returns gateway performance statistics
func (g *IndustrialGateway) GetStats() map[string]interface{} {
	deviceCount := 0
	connectedCount := 0
	
	g.devices.Range(func(key, value interface{}) bool {
		deviceCount++
		if value.(*Device).Connected {
			connectedCount++
		}
		return true
	})
	
	return map[string]interface{}{
		"devices_total":     deviceCount,
		"devices_connected": connectedCount,
		"uptime":           time.Since(time.Now()), // TODO: Track actual uptime
	}
}

// HTTP handlers

func (g *IndustrialGateway) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := g.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		g.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}
	
	// Register client
	g.wsClients.Store(conn, true)
	
	g.logger.Info("WebSocket client connected")
	
	// Handle client disconnect
	defer func() {
		g.wsClients.Delete(conn)
		conn.Close()
		g.logger.Info("WebSocket client disconnected")
	}()
	
	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (g *IndustrialGateway) handleDevices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	devices := make([]*Device, 0)
	g.devices.Range(func(key, value interface{}) bool {
		devices = append(devices, value.(*Device))
		return true
	})
	
	// Simple JSON response (in production, use proper JSON library)
	w.Write([]byte(fmt.Sprintf(`{"devices": %v}`, devices)))
}

func (g *IndustrialGateway) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// TODO: Implement device discovery
	w.Write([]byte(`{"message": "Device discovery not implemented yet"}`))
}

func (g *IndustrialGateway) handleTagRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// TODO: Implement tag reading
	w.Write([]byte(`{"message": "Tag reading not implemented yet"}`))
}

func (g *IndustrialGateway) handleTagWrite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// TODO: Implement tag writing
	w.Write([]byte(`{"message": "Tag writing not implemented yet"}`))
}