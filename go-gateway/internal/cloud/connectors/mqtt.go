package connectors

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"go.uber.org/zap"

	"bifrost-gateway/internal/cloud"
)

// MQTTConnector implements CloudConnector for generic MQTT brokers
type MQTTConnector struct {
	logger          *zap.Logger
	config          *cloud.ConnectorConfig
	mqttConfig      *MQTTConfig
	client          MQTTClient
	buffer          cloud.Buffer
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

// MQTTConfig holds MQTT-specific configuration
type MQTTConfig struct {
	Broker          string `yaml:"broker"`
	ClientID        string `yaml:"client_id"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	QoS             byte   `yaml:"qos"`
	Retain          bool   `yaml:"retain"`
	CleanSession    bool   `yaml:"clean_session"`
	KeepAlive       int    `yaml:"keep_alive"`
	TopicPrefix     string `yaml:"topic_prefix"`
	DataTopic       string `yaml:"data_topic"`
	EventTopic      string `yaml:"event_topic"`
	AlarmTopic      string `yaml:"alarm_topic"`
	CommandTopic    string `yaml:"command_topic"`
}

// MQTTClient interface for testability
type MQTTClient interface {
	Connect() error
	Disconnect(quiesce uint)
	IsConnected() bool
	Publish(topic string, qos byte, retained bool, payload interface{}) error
	Subscribe(topic string, qos byte, callback func(topic string, payload []byte)) error
	Unsubscribe(topics ...string) error
}

// DefaultMQTTConfig returns default MQTT configuration
func DefaultMQTTConfig() *MQTTConfig {
	return &MQTTConfig{
		ClientID:     fmt.Sprintf("bifrost-gateway-%d", time.Now().Unix()),
		QoS:          1,
		Retain:       false,
		CleanSession: true,
		KeepAlive:    60,
		TopicPrefix:  "bifrost",
		DataTopic:    "data",
		EventTopic:   "events",
		AlarmTopic:   "alarms",
		CommandTopic: "commands",
	}
}

// NewMQTTConnector creates a new MQTT connector
func NewMQTTConnector(logger *zap.Logger, config *cloud.ConnectorConfig) (*MQTTConnector, error) {
	// Parse MQTT-specific configuration
	mqttConfig := DefaultMQTTConfig()
	if config.ProviderConfig != nil {
		if err := parseProviderConfig(config.ProviderConfig, mqttConfig); err != nil {
			return nil, fmt.Errorf("failed to parse MQTT config: %w", err)
		}
	}
	
	// Create buffer
	bufferConfig := &cloud.BufferConfig{
		MaxSize:        config.BufferSize,
		FlushInterval:  config.FlushInterval,
		PersistentPath: "/tmp/bifrost-mqtt-buffer",
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
	
	connector := &MQTTConnector{
		logger:            logger,
		config:            config,
		mqttConfig:        mqttConfig,
		buffer:            buffer,
		resilienceManager: resilienceManager,
		metrics: &cloud.ConnectorMetrics{},
		health: &cloud.HealthStatus{
			IsHealthy: false,
		},
	}
	
	// Create MQTT client
	connector.client = NewMockMQTTClient() // Using mock for now
	
	return connector, nil
}

// Connect establishes connection to MQTT broker
func (c *MQTTConnector) Connect(ctx context.Context) error {
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
			return cloud.NewCloudError("mqtt_connect_failed", err.Error(), "mqtt", "connect", true)
		}
		
		c.connected = true
		c.connectTime = time.Now()
		c.health.IsHealthy = true
		c.health.LastCommunication = time.Now()
		
		c.logger.Info("Connected to MQTT broker", 
			zap.String("broker", c.mqttConfig.Broker),
			zap.String("clientID", c.mqttConfig.ClientID))
		
		return nil
	}, "mqtt_connect")
}

// Disconnect closes connection to MQTT broker
func (c *MQTTConnector) Disconnect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if !c.connected {
		return nil
	}
	
	c.client.Disconnect(250) // 250ms quiesce time
	c.connected = false
	c.health.IsHealthy = false
	
	c.logger.Info("Disconnected from MQTT broker")
	
	return nil
}

// IsConnected returns connection status
func (c *MQTTConnector) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected && c.client.IsConnected()
}

// SendData sends a single data point to MQTT
func (c *MQTTConnector) SendData(ctx context.Context, data *cloud.CloudData) error {
	return c.resilienceManager.Execute(ctx, func(ctx context.Context) error {
		// Determine topic based on data type
		topic := c.buildTopic(c.mqttConfig.DataTopic, data.DeviceID)
		
		// Convert data to JSON
		payload, err := json.Marshal(data)
		if err != nil {
			return cloud.NewCloudError("mqtt_marshal_failed", err.Error(), "mqtt", "send_data", false)
		}
		
		// Publish to MQTT
		if err := c.client.Publish(topic, c.mqttConfig.QoS, c.mqttConfig.Retain, payload); err != nil {
			c.metrics.DataPointsFailed++
			c.health.ErrorCount++
			c.health.LastError = err.Error()
			return cloud.NewCloudError("mqtt_publish_failed", err.Error(), "mqtt", "send_data", true)
		}
		
		c.metrics.DataPointsSent++
		c.health.LastCommunication = time.Now()
		
		return nil
	}, "mqtt_send_data")
}

// SendBatch sends multiple data points to MQTT
func (c *MQTTConnector) SendBatch(ctx context.Context, batch []*cloud.CloudData) error {
	if len(batch) == 0 {
		return nil
	}
	
	return c.resilienceManager.Execute(ctx, func(ctx context.Context) error {
		startTime := time.Now()
		
		// Group by device for more efficient publishing
		deviceGroups := make(map[string][]*cloud.CloudData)
		for _, data := range batch {
			deviceGroups[data.DeviceID] = append(deviceGroups[data.DeviceID], data)
		}
		
		// Publish each device group
		for deviceID, deviceData := range deviceGroups {
			topic := c.buildTopic(c.mqttConfig.DataTopic, deviceID)
			
			// Create batch payload
			batchPayload := map[string]interface{}{
				"device_id":  deviceID,
				"timestamp":  time.Now(),
				"data_count": len(deviceData),
				"data":       deviceData,
			}
			
			payload, err := json.Marshal(batchPayload)
			if err != nil {
				return cloud.NewCloudError("mqtt_marshal_batch_failed", err.Error(), "mqtt", "send_batch", false)
			}
			
			if err := c.client.Publish(topic, c.mqttConfig.QoS, c.mqttConfig.Retain, payload); err != nil {
				c.metrics.BatchesFailed++
				c.health.ErrorCount++
				c.health.LastError = err.Error()
				return cloud.NewCloudError("mqtt_publish_batch_failed", err.Error(), "mqtt", "send_batch", true)
			}
		}
		
		// Update metrics
		c.metrics.BatchesSent++
		c.metrics.DataPointsSent += uint64(len(batch))
		c.health.LastCommunication = time.Now()
		
		duration := time.Since(startTime)
		c.updateAverageResponseTime(duration)
		
		c.logger.Debug("Sent batch to MQTT", 
			zap.Int("dataPoints", len(batch)),
			zap.Int("deviceGroups", len(deviceGroups)),
			zap.Duration("duration", duration))
		
		return nil
	}, "mqtt_send_batch")
}

// Ping tests connectivity to MQTT broker
func (c *MQTTConnector) Ping(ctx context.Context) error {
	if !c.IsConnected() {
		return cloud.NewCloudError("mqtt_not_connected", "Not connected to MQTT broker", "mqtt", "ping", true)
	}
	
	// MQTT doesn't have a built-in ping, so we publish to a test topic
	testTopic := c.buildTopic("ping", "test")
	testPayload := map[string]interface{}{
		"type":      "ping",
		"timestamp": time.Now(),
		"client_id": c.mqttConfig.ClientID,
	}
	
	payload, _ := json.Marshal(testPayload)
	
	start := time.Now()
	err := c.client.Publish(testTopic, 0, false, payload) // QoS 0 for ping
	duration := time.Since(start)
	
	if err != nil {
		c.health.ErrorCount++
		c.health.LastError = err.Error()
		return cloud.NewCloudError("mqtt_ping_failed", err.Error(), "mqtt", "ping", true)
	}
	
	c.health.ResponseTime = duration
	c.health.LastCommunication = time.Now()
	c.lastPingTime = time.Now()
	
	return nil
}

// GetHealth returns current health status
func (c *MQTTConnector) GetHealth() *cloud.HealthStatus {
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
func (c *MQTTConnector) GetMetrics() *cloud.ConnectorMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	metrics := *c.metrics // Copy
	if c.connected {
		metrics.TotalUptime = time.Since(c.connectTime)
	}
	
	return &metrics
}

// GetConfig returns connector configuration
func (c *MQTTConnector) GetConfig() *cloud.ConnectorConfig {
	return c.config
}

// ValidateConfig validates the connector configuration
func (c *MQTTConnector) ValidateConfig() error {
	if c.config.Endpoint == "" && c.mqttConfig.Broker == "" {
		return fmt.Errorf("MQTT broker endpoint is required")
	}
	
	if c.mqttConfig.ClientID == "" {
		return fmt.Errorf("MQTT client ID is required")
	}
	
	if c.mqttConfig.QoS > 2 {
		return fmt.Errorf("MQTT QoS must be 0, 1, or 2")
	}
	
	return nil
}

// buildTopic constructs MQTT topic with prefix and hierarchy
func (c *MQTTConnector) buildTopic(baseTopic, deviceID string) string {
	if c.mqttConfig.TopicPrefix == "" {
		return fmt.Sprintf("%s/%s", baseTopic, deviceID)
	}
	return fmt.Sprintf("%s/%s/%s", c.mqttConfig.TopicPrefix, baseTopic, deviceID)
}

// updateAverageResponseTime updates the average response time metric
func (c *MQTTConnector) updateAverageResponseTime(duration time.Duration) {
	// Simple moving average calculation
	if c.metrics.AverageResponseTime == 0 {
		c.metrics.AverageResponseTime = duration
	} else {
		c.metrics.AverageResponseTime = (c.metrics.AverageResponseTime + duration) / 2
	}
}

// createTLSConfig creates TLS configuration for secure MQTT connection
func (c *MQTTConnector) createTLSConfig() (*tls.Config, error) {
	if !c.config.TLSEnabled {
		return nil, nil
	}
	
	tlsConfig := &tls.Config{}
	
	// Load client certificate if provided
	if c.config.CertFile != "" && c.config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.config.CertFile, c.config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	
	// Load CA certificate if provided
	if c.config.CAFile != "" {
		caCert, err := ioutil.ReadFile(c.config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}
	
	return tlsConfig, nil
}

// parseProviderConfig parses provider-specific configuration into MQTTConfig
func parseProviderConfig(providerConfig map[string]interface{}, mqttConfig *MQTTConfig) error {
	// This is a simplified parser - in a real implementation you might use mapstructure
	if broker, ok := providerConfig["broker"].(string); ok {
		mqttConfig.Broker = broker
	}
	if clientID, ok := providerConfig["client_id"].(string); ok {
		mqttConfig.ClientID = clientID
	}
	if username, ok := providerConfig["username"].(string); ok {
		mqttConfig.Username = username
	}
	if password, ok := providerConfig["password"].(string); ok {
		mqttConfig.Password = password
	}
	if qos, ok := providerConfig["qos"].(float64); ok {
		mqttConfig.QoS = byte(qos)
	}
	if retain, ok := providerConfig["retain"].(bool); ok {
		mqttConfig.Retain = retain
	}
	if topicPrefix, ok := providerConfig["topic_prefix"].(string); ok {
		mqttConfig.TopicPrefix = topicPrefix
	}
	
	return nil
}

// MockMQTTClient provides a mock implementation for testing
type MockMQTTClient struct {
	connected bool
	publishedMessages []MockMessage
}

type MockMessage struct {
	Topic    string
	QoS      byte
	Retained bool
	Payload  interface{}
}

func NewMockMQTTClient() *MockMQTTClient {
	return &MockMQTTClient{
		publishedMessages: make([]MockMessage, 0),
	}
}

func (m *MockMQTTClient) Connect() error {
	m.connected = true
	return nil
}

func (m *MockMQTTClient) Disconnect(quiesce uint) {
	m.connected = false
}

func (m *MockMQTTClient) IsConnected() bool {
	return m.connected
}

func (m *MockMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	m.publishedMessages = append(m.publishedMessages, MockMessage{
		Topic:    topic,
		QoS:      qos,
		Retained: retained,
		Payload:  payload,
	})
	
	return nil
}

func (m *MockMQTTClient) Subscribe(topic string, qos byte, callback func(topic string, payload []byte)) error {
	// Mock implementation - no actual subscription
	return nil
}

func (m *MockMQTTClient) Unsubscribe(topics ...string) error {
	// Mock implementation - no actual unsubscription
	return nil
}