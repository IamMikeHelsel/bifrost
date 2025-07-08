package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"

	"bifrost-gateway/internal/cloud"
	"bifrost-gateway/internal/cloud/connectors"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create cloud manager configuration
	config := &cloud.ManagerConfig{
		DefaultConnector:    "mock-mqtt",
		BatchSize:          50,
		BatchTimeout:       3 * time.Second,
		HealthCheckInterval: 10 * time.Second,
		BufferConfig: &cloud.BufferConfig{
			MaxSize:        1000,
			FlushInterval:  5 * time.Second,
			PersistentPath: "/tmp/bifrost-example-buffer",
		},
		Connectors:   make(map[string]*cloud.ConnectorConfig),
		RoutingRules: make([]*cloud.RoutingRule, 0),
	}

	// Create cloud manager
	manager, err := cloud.NewManager(logger, config)
	if err != nil {
		log.Fatalf("Failed to create cloud manager: %v", err)
	}

	// Create and register a mock MQTT connector for demonstration
	mqttConfig := &cloud.ConnectorConfig{
		Name:           "Mock MQTT",
		Type:           "mqtt",
		Enabled:        true,
		Endpoint:       "tcp://localhost:1883",
		Timeout:        30 * time.Second,
		RetryCount:     3,
		RetryDelay:     2 * time.Second,
		BufferSize:     500,
		FlushInterval:  5 * time.Second,
		DiskPersistent: true,
		ProviderConfig: map[string]interface{}{
			"broker":       "tcp://localhost:1883",
			"client_id":    "bifrost-example",
			"qos":          1,
			"topic_prefix": "bifrost/example",
		},
	}

	mqttConnector, err := connectors.NewMQTTConnector(logger, mqttConfig)
	if err != nil {
		log.Fatalf("Failed to create MQTT connector: %v", err)
	}

	err = manager.RegisterConnector("mock-mqtt", mqttConnector)
	if err != nil {
		log.Fatalf("Failed to register MQTT connector: %v", err)
	}

	// Connect all connectors
	ctx := context.Background()
	err = manager.ConnectAll(ctx)
	if err != nil {
		logger.Error("Failed to connect some connectors", zap.Error(err))
		// Continue with example even if connection fails
	}

	// Simulate sending industrial data
	logger.Info("Starting cloud connector example...")

	// Send individual data points
	for i := 0; i < 10; i++ {
		data := &cloud.CloudData{
			ID:        fmt.Sprintf("example-%d", i),
			DeviceID:  "PLC001",
			TagName:   "temperature",
			Value:     20.0 + float64(i)*0.5,
			Quality:   "GOOD",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"unit":        "°C",
				"sensor_type": "RTD",
				"location":    "Tank A",
			},
		}

		err = manager.SendData(ctx, data)
		if err != nil {
			logger.Error("Failed to send data", zap.Error(err))
		} else {
			logger.Info("Sent data point", 
				zap.String("id", data.ID),
				zap.Float64("value", data.Value.(float64)))
		}

		time.Sleep(1 * time.Second)
	}

	// Send a batch of data
	batch := make([]*cloud.CloudData, 5)
	for i := 0; i < 5; i++ {
		batch[i] = &cloud.CloudData{
			ID:        fmt.Sprintf("batch-%d", i),
			DeviceID:  "PLC002",
			TagName:   "pressure",
			Value:     100.0 + float64(i)*10.0,
			Quality:   "GOOD",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"unit":        "PSI",
				"sensor_type": "pressure_transmitter",
				"location":    "Pipeline B",
			},
		}
	}

	err = manager.SendBatch(ctx, batch)
	if err != nil {
		logger.Error("Failed to send batch", zap.Error(err))
	} else {
		logger.Info("Sent batch data", zap.Int("count", len(batch)))
	}

	// Display health and metrics
	time.Sleep(2 * time.Second)

	health := manager.GetHealth()
	for name, status := range health {
		logger.Info("Connector health",
			zap.String("connector", name),
			zap.Bool("healthy", status.IsHealthy),
			zap.Float64("success_rate", status.SuccessRate),
			zap.Duration("uptime", status.ConnectionUptime))
	}

	metrics := manager.GetMetrics()
	for name, metric := range metrics {
		logger.Info("Connector metrics",
			zap.String("connector", name),
			zap.Uint64("data_points_sent", metric.DataPointsSent),
			zap.Uint64("batches_sent", metric.BatchesSent),
			zap.Duration("avg_response_time", metric.AverageResponseTime))
	}

	// Graceful shutdown
	logger.Info("Shutting down cloud manager...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = manager.Shutdown(shutdownCtx)
	if err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
	}

	logger.Info("Cloud connector example completed")
}