package connectors

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"

	"bifrost-gateway/internal/cloud"
)

// InfluxDBConnector implements CloudConnector for InfluxDB time-series database
type InfluxDBConnector struct {
	logger            *zap.Logger
	config            *cloud.ConnectorConfig
	influxConfig      *InfluxDBConfig
	client            InfluxDBClient
	buffer            cloud.Buffer
	resilienceManager *cloud.ResilienceManager
	
	// Connection state
	connected     bool
	connectTime   time.Time
	mutex         sync.RWMutex
	
	// Metrics
	metrics       *cloud.ConnectorMetrics
	health        *cloud.HealthStatus
	lastPingTime  time.Time
}

// InfluxDBConfig holds InfluxDB-specific configuration
type InfluxDBConfig struct {
	URL              string `yaml:"url"`
	Database         string `yaml:"database"`
	Username         string `yaml:"username"`
	Password         string `yaml:"password"`
	Token            string `yaml:"token"`
	Organization     string `yaml:"organization"`
	Bucket           string `yaml:"bucket"`
	Precision        string `yaml:"precision"`
	RetentionPolicy  string `yaml:"retention_policy"`
	Measurement      string `yaml:"measurement"`
	BatchSize        int    `yaml:"batch_size"`
	FlushInterval    time.Duration `yaml:"flush_interval"`
	Version          string `yaml:"version"` // "1.x" or "2.x"
}

// InfluxDBClient interface for testability
type InfluxDBClient interface {
	Connect() error
	Disconnect() error
	IsConnected() bool
	WritePoint(ctx context.Context, point *InfluxPoint) error
	WriteBatch(ctx context.Context, points []*InfluxPoint) error
	Ping(ctx context.Context) error
}

// InfluxPoint represents a data point for InfluxDB
type InfluxPoint struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{}
	Timestamp   time.Time
}

// DefaultInfluxDBConfig returns default InfluxDB configuration
func DefaultInfluxDBConfig() *InfluxDBConfig {
	return &InfluxDBConfig{
		Database:        "bifrost",
		Precision:       "ns",
		RetentionPolicy: "autogen",
		Measurement:     "industrial_data",
		BatchSize:       1000,
		FlushInterval:   10 * time.Second,
		Version:         "1.x",
	}
}

// NewInfluxDBConnector creates a new InfluxDB connector
func NewInfluxDBConnector(logger *zap.Logger, config *cloud.ConnectorConfig) (*InfluxDBConnector, error) {
	// Parse InfluxDB-specific configuration
	influxConfig := DefaultInfluxDBConfig()
	if config.ProviderConfig != nil {
		if err := parseInfluxDBProviderConfig(config.ProviderConfig, influxConfig); err != nil {
			return nil, fmt.Errorf("failed to parse InfluxDB config: %w", err)
		}
	}
	
	// Set URL from main config if not provided
	if influxConfig.URL == "" {
		influxConfig.URL = config.Endpoint
	}
	
	// Create buffer
	bufferConfig := &cloud.BufferConfig{
		MaxSize:        config.BufferSize,
		FlushInterval:  config.FlushInterval,
		PersistentPath: "/tmp/bifrost-influxdb-buffer",
	}
	
	var buffer cloud.Buffer
	var err error
	if config.DiskPersistent {
		buffer, err = cloud.NewDiskBuffer(logger, bufferConfig)
	} else {
		buffer = cloud.NewMemoryBuffer(logger, bufferConfig)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create buffer: %w", err)
	}
	
	// Create resilience manager
	retryConfig := &cloud.RetryConfig{
		MaxRetries:   config.RetryCount,
		InitialDelay: config.RetryDelay,
		MaxDelay:     30 * time.Second,
		Strategy:     cloud.RetryStrategyExponential,
		Jitter:       true,
	}
	
	resilienceManager := cloud.NewResilienceManager(logger, retryConfig, nil)
	
	connector := &InfluxDBConnector{
		logger:            logger,
		config:            config,
		influxConfig:      influxConfig,
		buffer:            buffer,
		resilienceManager: resilienceManager,
		metrics: &cloud.ConnectorMetrics{},
		health: &cloud.HealthStatus{
			IsHealthy: false,
		},
	}
	
	// Create InfluxDB client
	connector.client = NewMockInfluxDBClient()
	
	return connector, nil
}

// Connect establishes connection to InfluxDB
func (c *InfluxDBConnector) Connect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if c.connected {
		return nil
	}
	
	return c.resilienceManager.Execute(ctx, func(ctx context.Context) error {
		c.metrics.ConnectionAttempts++
		
		if err := c.client.Connect(); err != nil {
			c.metrics.DataPointsFailed++
			c.health.ErrorCount++
			c.health.LastError = err.Error()
			return cloud.NewCloudError("influxdb_connect_failed", err.Error(), "influxdb", "connect", true)
		}
		
		c.connected = true
		c.connectTime = time.Now()
		c.health.IsHealthy = true
		c.health.LastCommunication = time.Now()
		
		c.logger.Info("Connected to InfluxDB", 
			zap.String("url", c.influxConfig.URL),
			zap.String("database", c.influxConfig.Database),
			zap.String("version", c.influxConfig.Version))
		
		return nil
	}, "influxdb_connect")
}

// Disconnect closes connection to InfluxDB
func (c *InfluxDBConnector) Disconnect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if !c.connected {
		return nil
	}
	
	if err := c.client.Disconnect(); err != nil {
		c.logger.Warn("Error disconnecting from InfluxDB", zap.Error(err))
	}
	
	c.connected = false
	c.health.IsHealthy = false
	
	c.logger.Info("Disconnected from InfluxDB")
	
	return nil
}

// IsConnected returns connection status
func (c *InfluxDBConnector) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected && c.client.IsConnected()
}

// SendData sends a single data point to InfluxDB
func (c *InfluxDBConnector) SendData(ctx context.Context, data *cloud.CloudData) error {
	return c.resilienceManager.Execute(ctx, func(ctx context.Context) error {
		// Convert CloudData to InfluxPoint
		point := c.convertToInfluxPoint(data)
		
		// Write point to InfluxDB
		if err := c.client.WritePoint(ctx, point); err != nil {
			c.metrics.DataPointsFailed++
			c.health.ErrorCount++
			c.health.LastError = err.Error()
			return cloud.NewCloudError("influxdb_write_failed", err.Error(), "influxdb", "send_data", true)
		}
		
		c.metrics.DataPointsSent++
		c.health.LastCommunication = time.Now()
		
		return nil
	}, "influxdb_send_data")
}

// SendBatch sends multiple data points to InfluxDB
func (c *InfluxDBConnector) SendBatch(ctx context.Context, batch []*cloud.CloudData) error {
	if len(batch) == 0 {
		return nil
	}
	
	return c.resilienceManager.Execute(ctx, func(ctx context.Context) error {
		startTime := time.Now()
		
		// Convert CloudData to InfluxPoints
		points := make([]*InfluxPoint, len(batch))
		for i, data := range batch {
			points[i] = c.convertToInfluxPoint(data)
		}
		
		// Write batch to InfluxDB
		if err := c.client.WriteBatch(ctx, points); err != nil {
			c.metrics.BatchesFailed++
			c.health.ErrorCount++
			c.health.LastError = err.Error()
			return cloud.NewCloudError("influxdb_batch_write_failed", err.Error(), "influxdb", "send_batch", true)
		}
		
		// Update metrics
		c.metrics.BatchesSent++
		c.metrics.DataPointsSent += uint64(len(batch))
		c.health.LastCommunication = time.Now()
		
		duration := time.Since(startTime)
		c.updateAverageResponseTime(duration)
		
		c.logger.Debug("Sent batch to InfluxDB", 
			zap.Int("dataPoints", len(batch)),
			zap.Duration("duration", duration))
		
		return nil
	}, "influxdb_send_batch")
}

// Ping tests connectivity to InfluxDB
func (c *InfluxDBConnector) Ping(ctx context.Context) error {
	if !c.IsConnected() {
		return cloud.NewCloudError("influxdb_not_connected", "Not connected to InfluxDB", "influxdb", "ping", true)
	}
	
	start := time.Now()
	err := c.client.Ping(ctx)
	duration := time.Since(start)
	
	if err != nil {
		c.health.ErrorCount++
		c.health.LastError = err.Error()
		return cloud.NewCloudError("influxdb_ping_failed", err.Error(), "influxdb", "ping", true)
	}
	
	c.health.ResponseTime = duration
	c.health.LastCommunication = time.Now()
	c.lastPingTime = time.Now()
	
	return nil
}

// GetHealth returns current health status
func (c *InfluxDBConnector) GetHealth() *cloud.HealthStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	health := *c.health // Copy
	health.IsHealthy = c.connected && c.client.IsConnected()
	
	if c.connected {
		health.ConnectionUptime = time.Since(c.connectTime)
	}
	
	// Calculate success rate
	totalOperations := c.metrics.DataPointsSent + c.metrics.DataPointsFailed + 
		c.metrics.BatchesSent + c.metrics.BatchesFailed
	if totalOperations > 0 {
		successfulOperations := c.metrics.DataPointsSent + c.metrics.BatchesSent
		health.SuccessRate = float64(successfulOperations) / float64(totalOperations)
	}
	
	return &health
}

// GetMetrics returns current metrics
func (c *InfluxDBConnector) GetMetrics() *cloud.ConnectorMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	metrics := *c.metrics // Copy
	if c.connected {
		metrics.TotalUptime = time.Since(c.connectTime)
	}
	
	return &metrics
}

// GetConfig returns connector configuration
func (c *InfluxDBConnector) GetConfig() *cloud.ConnectorConfig {
	return c.config
}

// ValidateConfig validates the connector configuration
func (c *InfluxDBConnector) ValidateConfig() error {
	if c.influxConfig.URL == "" {
		return fmt.Errorf("InfluxDB URL is required")
	}
	
	// Validate URL format
	if _, err := url.Parse(c.influxConfig.URL); err != nil {
		return fmt.Errorf("invalid InfluxDB URL: %w", err)
	}
	
	// Version-specific validation
	switch c.influxConfig.Version {
	case "1.x":
		if c.influxConfig.Database == "" {
			return fmt.Errorf("database name is required for InfluxDB 1.x")
		}
	case "2.x":
		if c.influxConfig.Bucket == "" {
			return fmt.Errorf("bucket name is required for InfluxDB 2.x")
		}
		if c.influxConfig.Organization == "" {
			return fmt.Errorf("organization is required for InfluxDB 2.x")
		}
		if c.influxConfig.Token == "" {
			return fmt.Errorf("token is required for InfluxDB 2.x")
		}
	default:
		return fmt.Errorf("unsupported InfluxDB version: %s", c.influxConfig.Version)
	}
	
	return nil
}

// convertToInfluxPoint converts CloudData to InfluxPoint
func (c *InfluxDBConnector) convertToInfluxPoint(data *cloud.CloudData) *InfluxPoint {
	// Create tags
	tags := map[string]string{
		"device_id": data.DeviceID,
		"tag_name":  data.TagName,
		"quality":   data.Quality,
	}
	
	// Add metadata as tags if present
	if data.Metadata != nil {
		for key, value := range data.Metadata {
			if strValue, ok := value.(string); ok {
				tags[key] = strValue
			}
		}
	}
	
	// Create fields
	fields := map[string]interface{}{
		"value": data.Value,
	}
	
	// Add numeric metadata as fields
	if data.Metadata != nil {
		for key, value := range data.Metadata {
			switch v := value.(type) {
			case int, int32, int64, float32, float64, bool:
				fields[key] = v
			}
		}
	}
	
	return &InfluxPoint{
		Measurement: c.influxConfig.Measurement,
		Tags:        tags,
		Fields:      fields,
		Timestamp:   data.Timestamp,
	}
}

// updateAverageResponseTime updates the average response time metric
func (c *InfluxDBConnector) updateAverageResponseTime(duration time.Duration) {
	// Simple moving average calculation
	if c.metrics.AverageResponseTime == 0 {
		c.metrics.AverageResponseTime = duration
	} else {
		c.metrics.AverageResponseTime = (c.metrics.AverageResponseTime + duration) / 2
	}
}

// parseInfluxDBProviderConfig parses provider-specific configuration into InfluxDBConfig
func parseInfluxDBProviderConfig(providerConfig map[string]interface{}, influxConfig *InfluxDBConfig) error {
	if url, ok := providerConfig["url"].(string); ok {
		influxConfig.URL = url
	}
	if database, ok := providerConfig["database"].(string); ok {
		influxConfig.Database = database
	}
	if username, ok := providerConfig["username"].(string); ok {
		influxConfig.Username = username
	}
	if password, ok := providerConfig["password"].(string); ok {
		influxConfig.Password = password
	}
	if token, ok := providerConfig["token"].(string); ok {
		influxConfig.Token = token
	}
	if organization, ok := providerConfig["organization"].(string); ok {
		influxConfig.Organization = organization
	}
	if bucket, ok := providerConfig["bucket"].(string); ok {
		influxConfig.Bucket = bucket
	}
	if precision, ok := providerConfig["precision"].(string); ok {
		influxConfig.Precision = precision
	}
	if retentionPolicy, ok := providerConfig["retention_policy"].(string); ok {
		influxConfig.RetentionPolicy = retentionPolicy
	}
	if measurement, ok := providerConfig["measurement"].(string); ok {
		influxConfig.Measurement = measurement
	}
	if batchSize, ok := providerConfig["batch_size"].(float64); ok {
		influxConfig.BatchSize = int(batchSize)
	}
	if flushIntervalStr, ok := providerConfig["flush_interval"].(string); ok {
		if duration, err := time.ParseDuration(flushIntervalStr); err == nil {
			influxConfig.FlushInterval = duration
		}
	}
	if version, ok := providerConfig["version"].(string); ok {
		influxConfig.Version = version
	}
	
	return nil
}

// MockInfluxDBClient provides a mock implementation for testing
type MockInfluxDBClient struct {
	connected bool
	points    []*InfluxPoint
	mutex     sync.RWMutex
}

func NewMockInfluxDBClient() *MockInfluxDBClient {
	return &MockInfluxDBClient{
		points: make([]*InfluxPoint, 0),
	}
}

func (m *MockInfluxDBClient) Connect() error {
	m.connected = true
	return nil
}

func (m *MockInfluxDBClient) Disconnect() error {
	m.connected = false
	return nil
}

func (m *MockInfluxDBClient) IsConnected() bool {
	return m.connected
}

func (m *MockInfluxDBClient) WritePoint(ctx context.Context, point *InfluxPoint) error {
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.points = append(m.points, point)
	return nil
}

func (m *MockInfluxDBClient) WriteBatch(ctx context.Context, points []*InfluxPoint) error {
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.points = append(m.points, points...)
	return nil
}

func (m *MockInfluxDBClient) Ping(ctx context.Context) error {
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	return nil
}

func (m *MockInfluxDBClient) GetPoints() []*InfluxPoint {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	result := make([]*InfluxPoint, len(m.points))
	copy(result, m.points)
	return result
}

func (m *MockInfluxDBClient) ClearPoints() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.points = m.points[:0]
}