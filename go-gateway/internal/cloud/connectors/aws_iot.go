package connectors

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"bifrost-gateway/internal/cloud"
)

// AWSIoTConnector implements CloudConnector for AWS IoT Core
type AWSIoTConnector struct {
	logger            *zap.Logger
	config            *cloud.ConnectorConfig
	awsConfig         *AWSIoTConfig
	mqttClient        MQTTClient
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

// AWSIoTConfig holds AWS IoT Core specific configuration
type AWSIoTConfig struct {
	Endpoint         string `yaml:"endpoint"`
	Region           string `yaml:"region"`
	ClientID         string `yaml:"client_id"`
	CertFile         string `yaml:"cert_file"`
	KeyFile          string `yaml:"key_file"`
	CAFile           string `yaml:"ca_file"`
	ThingName        string `yaml:"thing_name"`
	TopicPrefix      string `yaml:"topic_prefix"`
	DataTopic        string `yaml:"data_topic"`
	ShadowUpdate     bool   `yaml:"shadow_update"`
	QoS              byte   `yaml:"qos"`
	KeepAlive        int    `yaml:"keep_alive"`
}

// DefaultAWSIoTConfig returns default AWS IoT configuration
func DefaultAWSIoTConfig() *AWSIoTConfig {
	return &AWSIoTConfig{
		Region:       "us-east-1",
		ClientID:     fmt.Sprintf("bifrost-gateway-%d", time.Now().Unix()),
		TopicPrefix:  "bifrost",
		DataTopic:    "telemetry",
		ShadowUpdate: false,
		QoS:          1,
		KeepAlive:    60,
	}
}

// NewAWSIoTConnector creates a new AWS IoT Core connector
func NewAWSIoTConnector(logger *zap.Logger, config *cloud.ConnectorConfig) (*AWSIoTConnector, error) {
	// Parse AWS IoT specific configuration
	awsConfig := DefaultAWSIoTConfig()
	if config.ProviderConfig != nil {
		if err := parseAWSIoTProviderConfig(config.ProviderConfig, awsConfig); err != nil {
			return nil, fmt.Errorf("failed to parse AWS IoT config: %w", err)
		}
	}
	
	// Use endpoint from main config if not specified
	if awsConfig.Endpoint == "" {
		awsConfig.Endpoint = config.Endpoint
	}
	
	// Override certificate files from main config if specified
	if config.CertFile != "" {
		awsConfig.CertFile = config.CertFile
	}
	if config.KeyFile != "" {
		awsConfig.KeyFile = config.KeyFile
	}
	if config.CAFile != "" {
		awsConfig.CAFile = config.CAFile
	}
	
	// Create buffer
	bufferConfig := &cloud.BufferConfig{
		MaxSize:        config.BufferSize,
		FlushInterval:  config.FlushInterval,
		PersistentPath: "/tmp/bifrost-aws-iot-buffer",
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
	
	// Create resilience manager with AWS IoT specific settings
	retryConfig := &cloud.RetryConfig{
		MaxRetries:   config.RetryCount,
		InitialDelay: config.RetryDelay,
		MaxDelay:     60 * time.Second, // AWS IoT can have longer delays
		Strategy:     cloud.RetryStrategyExponential,
		Jitter:       true,
	}
	
	resilienceManager := cloud.NewResilienceManager(logger, retryConfig, nil)
	
	connector := &AWSIoTConnector{
		logger:            logger,
		config:            config,
		awsConfig:         awsConfig,
		buffer:            buffer,
		resilienceManager: resilienceManager,
		metrics: &cloud.ConnectorMetrics{},
		health: &cloud.HealthStatus{
			IsHealthy: false,
		},
	}
	
	// Create MQTT client configured for AWS IoT
	connector.mqttClient = NewMockMQTTClient() // Using mock for now
	
	return connector, nil
}

// Connect establishes connection to AWS IoT Core
func (c *AWSIoTConnector) Connect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if c.connected {
		return nil
	}
	
	return c.resilienceManager.Execute(ctx, func(ctx context.Context) error {
		c.metrics.ConnectionAttempts++
		
		// AWS IoT Core requires TLS with client certificates
		if err := c.validateCertificates(); err != nil {
			return cloud.NewCloudError("aws_iot_cert_validation_failed", err.Error(), "aws-iot", "connect", false)
		}
		
		if err := c.mqttClient.Connect(); err != nil {
			c.metrics.DataPointsFailed++
			c.health.ErrorCount++
			c.health.LastError = err.Error()
			return cloud.NewCloudError("aws_iot_connect_failed", err.Error(), "aws-iot", "connect", true)
		}
		
		c.connected = true
		c.connectTime = time.Now()
		c.health.IsHealthy = true
		c.health.LastCommunication = time.Now()
		
		c.logger.Info("Connected to AWS IoT Core", 
			zap.String("endpoint", c.awsConfig.Endpoint),
			zap.String("clientID", c.awsConfig.ClientID),
			zap.String("thingName", c.awsConfig.ThingName))
		
		return nil
	}, "aws_iot_connect")
}

// Disconnect closes connection to AWS IoT Core
func (c *AWSIoTConnector) Disconnect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if !c.connected {
		return nil
	}
	
	c.mqttClient.Disconnect(250)
	c.connected = false
	c.health.IsHealthy = false
	
	c.logger.Info("Disconnected from AWS IoT Core")
	
	return nil
}

// IsConnected returns connection status
func (c *AWSIoTConnector) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected && c.mqttClient.IsConnected()
}

// SendData sends a single data point to AWS IoT Core
func (c *AWSIoTConnector) SendData(ctx context.Context, data *cloud.CloudData) error {
	return c.resilienceManager.Execute(ctx, func(ctx context.Context) error {
		// Create AWS IoT message format
		awsMessage := c.convertToAWSIoTMessage(data)
		
		// Determine topic
		topic := c.buildTopic(data.DeviceID, "data")
		
		// Convert to JSON
		payload, err := json.Marshal(awsMessage)
		if err != nil {
			return cloud.NewCloudError("aws_iot_marshal_failed", err.Error(), "aws-iot", "send_data", false)
		}
		
		// Publish to AWS IoT
		if err := c.mqttClient.Publish(topic, c.awsConfig.QoS, false, payload); err != nil {
			c.metrics.DataPointsFailed++
			c.health.ErrorCount++
			c.health.LastError = err.Error()
			return cloud.NewCloudError("aws_iot_publish_failed", err.Error(), "aws-iot", "send_data", true)
		}
		
		c.metrics.DataPointsSent++
		c.health.LastCommunication = time.Now()
		
		// Update device shadow if enabled
		if c.awsConfig.ShadowUpdate {
			c.updateDeviceShadow(data)
		}
		
		return nil
	}, "aws_iot_send_data")
}

// SendBatch sends multiple data points to AWS IoT Core
func (c *AWSIoTConnector) SendBatch(ctx context.Context, batch []*cloud.CloudData) error {
	if len(batch) == 0 {
		return nil
	}
	
	return c.resilienceManager.Execute(ctx, func(ctx context.Context) error {
		startTime := time.Now()
		
		// Group by device for AWS IoT topics
		deviceGroups := make(map[string][]*cloud.CloudData)
		for _, data := range batch {
			deviceGroups[data.DeviceID] = append(deviceGroups[data.DeviceID], data)
		}
		
		// Send each device group
		for deviceID, deviceData := range deviceGroups {
			topic := c.buildTopic(deviceID, "batch")
			
			// Create AWS IoT batch message
			batchMessage := map[string]interface{}{
				"deviceId":   deviceID,
				"timestamp":  time.Now().Unix(),
				"dataCount":  len(deviceData),
				"telemetry":  c.convertBatchToAWSIoTMessages(deviceData),
			}
			
			payload, err := json.Marshal(batchMessage)
			if err != nil {
				return cloud.NewCloudError("aws_iot_marshal_batch_failed", err.Error(), "aws-iot", "send_batch", false)
			}
			
			if err := c.mqttClient.Publish(topic, c.awsConfig.QoS, false, payload); err != nil {
				c.metrics.BatchesFailed++
				c.health.ErrorCount++
				c.health.LastError = err.Error()
				return cloud.NewCloudError("aws_iot_publish_batch_failed", err.Error(), "aws-iot", "send_batch", true)
			}
		}
		
		// Update metrics
		c.metrics.BatchesSent++
		c.metrics.DataPointsSent += uint64(len(batch))
		c.health.LastCommunication = time.Now()
		
		duration := time.Since(startTime)
		c.updateAverageResponseTime(duration)
		
		c.logger.Debug("Sent batch to AWS IoT", 
			zap.Int("dataPoints", len(batch)),
			zap.Int("deviceGroups", len(deviceGroups)),
			zap.Duration("duration", duration))
		
		return nil
	}, "aws_iot_send_batch")
}

// Ping tests connectivity to AWS IoT Core
func (c *AWSIoTConnector) Ping(ctx context.Context) error {
	if !c.IsConnected() {
		return cloud.NewCloudError("aws_iot_not_connected", "Not connected to AWS IoT Core", "aws-iot", "ping", true)
	}
	
	// Publish to a health check topic
	healthTopic := c.buildTopic(c.awsConfig.ThingName, "health")
	healthMessage := map[string]interface{}{
		"type":      "ping",
		"timestamp": time.Now().Unix(),
		"clientId":  c.awsConfig.ClientID,
		"thingName": c.awsConfig.ThingName,
	}
	
	payload, _ := json.Marshal(healthMessage)
	
	start := time.Now()
	err := c.mqttClient.Publish(healthTopic, 0, false, payload) // QoS 0 for ping
	duration := time.Since(start)
	
	if err != nil {
		c.health.ErrorCount++
		c.health.LastError = err.Error()
		return cloud.NewCloudError("aws_iot_ping_failed", err.Error(), "aws-iot", "ping", true)
	}
	
	c.health.ResponseTime = duration
	c.health.LastCommunication = time.Now()
	c.lastPingTime = time.Now()
	
	return nil
}

// GetHealth returns current health status
func (c *AWSIoTConnector) GetHealth() *cloud.HealthStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	health := *c.health // Copy
	health.IsHealthy = c.connected && c.mqttClient.IsConnected()
	
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
func (c *AWSIoTConnector) GetMetrics() *cloud.ConnectorMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	metrics := *c.metrics // Copy
	if c.connected {
		metrics.TotalUptime = time.Since(c.connectTime)
	}
	
	return &metrics
}

// GetConfig returns connector configuration
func (c *AWSIoTConnector) GetConfig() *cloud.ConnectorConfig {
	return c.config
}

// ValidateConfig validates the connector configuration
func (c *AWSIoTConnector) ValidateConfig() error {
	if c.awsConfig.Endpoint == "" {
		return fmt.Errorf("AWS IoT endpoint is required")
	}
	
	if c.awsConfig.ClientID == "" {
		return fmt.Errorf("AWS IoT client ID is required")
	}
	
	if c.awsConfig.CertFile == "" {
		return fmt.Errorf("AWS IoT client certificate file is required")
	}
	
	if c.awsConfig.KeyFile == "" {
		return fmt.Errorf("AWS IoT private key file is required")
	}
	
	if c.awsConfig.CAFile == "" {
		return fmt.Errorf("AWS IoT CA certificate file is required")
	}
	
	return nil
}

// validateCertificates validates that required certificates exist and are valid
func (c *AWSIoTConnector) validateCertificates() error {
	// Load client certificate
	_, err := tls.LoadX509KeyPair(c.awsConfig.CertFile, c.awsConfig.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}
	
	// TODO: Add CA certificate validation
	
	return nil
}

// buildTopic constructs AWS IoT topic with proper hierarchy
func (c *AWSIoTConnector) buildTopic(deviceID, messageType string) string {
	if c.awsConfig.TopicPrefix == "" {
		return fmt.Sprintf("device/%s/%s", deviceID, messageType)
	}
	return fmt.Sprintf("%s/device/%s/%s", c.awsConfig.TopicPrefix, deviceID, messageType)
}

// convertToAWSIoTMessage converts CloudData to AWS IoT message format
func (c *AWSIoTConnector) convertToAWSIoTMessage(data *cloud.CloudData) map[string]interface{} {
	message := map[string]interface{}{
		"deviceId":  data.DeviceID,
		"tagName":   data.TagName,
		"value":     data.Value,
		"quality":   data.Quality,
		"timestamp": data.Timestamp.Unix(),
		"messageId": data.ID,
	}
	
	// Add metadata if present
	if data.Metadata != nil {
		message["metadata"] = data.Metadata
	}
	
	return message
}

// convertBatchToAWSIoTMessages converts batch to AWS IoT message format
func (c *AWSIoTConnector) convertBatchToAWSIoTMessages(batch []*cloud.CloudData) []map[string]interface{} {
	messages := make([]map[string]interface{}, len(batch))
	for i, data := range batch {
		messages[i] = c.convertToAWSIoTMessage(data)
	}
	return messages
}

// updateDeviceShadow updates the AWS IoT device shadow (simplified)
func (c *AWSIoTConnector) updateDeviceShadow(data *cloud.CloudData) error {
	shadowTopic := fmt.Sprintf("$aws/thing/%s/shadow/update", data.DeviceID)
	
	shadowUpdate := map[string]interface{}{
		"state": map[string]interface{}{
			"reported": map[string]interface{}{
				data.TagName: data.Value,
				"timestamp": data.Timestamp.Unix(),
				"quality":   data.Quality,
			},
		},
	}
	
	payload, err := json.Marshal(shadowUpdate)
	if err != nil {
		return err
	}
	
	return c.mqttClient.Publish(shadowTopic, 1, false, payload)
}

// updateAverageResponseTime updates the average response time metric
func (c *AWSIoTConnector) updateAverageResponseTime(duration time.Duration) {
	if c.metrics.AverageResponseTime == 0 {
		c.metrics.AverageResponseTime = duration
	} else {
		c.metrics.AverageResponseTime = (c.metrics.AverageResponseTime + duration) / 2
	}
}

// parseAWSIoTProviderConfig parses provider-specific configuration into AWSIoTConfig
func parseAWSIoTProviderConfig(providerConfig map[string]interface{}, awsConfig *AWSIoTConfig) error {
	if endpoint, ok := providerConfig["endpoint"].(string); ok {
		awsConfig.Endpoint = endpoint
	}
	if region, ok := providerConfig["region"].(string); ok {
		awsConfig.Region = region
	}
	if clientID, ok := providerConfig["client_id"].(string); ok {
		awsConfig.ClientID = clientID
	}
	if certFile, ok := providerConfig["cert_file"].(string); ok {
		awsConfig.CertFile = certFile
	}
	if keyFile, ok := providerConfig["key_file"].(string); ok {
		awsConfig.KeyFile = keyFile
	}
	if caFile, ok := providerConfig["ca_file"].(string); ok {
		awsConfig.CAFile = caFile
	}
	if thingName, ok := providerConfig["thing_name"].(string); ok {
		awsConfig.ThingName = thingName
	}
	if topicPrefix, ok := providerConfig["topic_prefix"].(string); ok {
		awsConfig.TopicPrefix = topicPrefix
	}
	if dataTopic, ok := providerConfig["data_topic"].(string); ok {
		awsConfig.DataTopic = dataTopic
	}
	if shadowUpdate, ok := providerConfig["shadow_update"].(bool); ok {
		awsConfig.ShadowUpdate = shadowUpdate
	}
	if qos, ok := providerConfig["qos"].(float64); ok {
		awsConfig.QoS = byte(qos)
	}
	if keepAlive, ok := providerConfig["keep_alive"].(float64); ok {
		awsConfig.KeepAlive = int(keepAlive)
	}
	
	return nil
}