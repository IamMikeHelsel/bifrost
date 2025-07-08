package main

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"bifrost-gateway/internal/messaging"
	"bifrost-gateway/internal/protocols"
	"bifrost-gateway/internal/security"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Example 1: MQTT Messaging
	fmt.Println("=== MQTT Messaging Example ===")
	mqttExample(logger)

	// Example 2: NATS Messaging
	fmt.Println("\n=== NATS Messaging Example ===")
	natsExample(logger)

	// Example 3: Security Layer
	fmt.Println("\n=== Security Layer Example ===")
	securityExample(logger)

	// Example 4: Topic/Subject Building
	fmt.Println("\n=== Topic Building Example ===")
	topicExample()
}

func mqttExample(logger *zap.Logger) {
	// MQTT Configuration
	config := &messaging.MQTTConfig{
		Enabled:        true,
		Broker:         "tcp://localhost:1883", // Would be your MQTT broker
		ClientID:       "bifrost-example",
		QoS:            1,
		KeepAlive:      60 * time.Second,
		ConnectTimeout: 30 * time.Second,
		WriteTimeout:   10 * time.Second,
		AutoReconnect:  true,
		CleanSession:   true,
	}

	// Create MQTT messaging instance
	mqtt, err := messaging.NewMQTTMessaging(config, logger)
	if err != nil {
		fmt.Printf("Failed to create MQTT messaging: %v\n", err)
		return
	}

	// Note: Connection would fail without an actual MQTT broker
	fmt.Println("MQTT messaging instance created successfully")
	fmt.Printf("Connected: %v\n", mqtt.IsConnected())

	// Example of what would happen when connected:
	// ctx := context.Background()
	// if err := mqtt.Connect(ctx); err == nil {
	//     tag := &protocols.Tag{
	//         ID:    "temp_001",
	//         Name:  "temperature",
	//         Value: 25.5,
	//         Quality: "good",
	//     }
	//     mqtt.PublishDeviceData("plc001", tag)
	// }
}

func natsExample(logger *zap.Logger) {
	// NATS Configuration
	config := &messaging.NATSConfig{
		Enabled:        true,
		Servers:        []string{"localhost:4222"}, // Would be your NATS server
		ClientID:       "bifrost-example",
		MaxReconnects:  10,
		ReconnectWait:  2 * time.Second,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 10 * time.Second,
		DrainTimeout:   5 * time.Second,
	}

	// Create NATS messaging instance
	nats, err := messaging.NewNATSMessaging(config, logger)
	if err != nil {
		fmt.Printf("Failed to create NATS messaging: %v\n", err)
		return
	}

	fmt.Println("NATS messaging instance created successfully")
	fmt.Printf("Connected: %v\n", nats.IsConnected())

	// Example of what would happen when connected:
	// ctx := context.Background()
	// if err := nats.Connect(ctx); err == nil {
	//     // Request-reply pattern (unique to NATS)
	//     response, err := nats.RequestReply("control.command", []byte("start"), 5*time.Second)
	//     if err == nil {
	//         fmt.Printf("Command response: %s\n", response)
	//     }
	// }
}

func securityExample(logger *zap.Logger) {
	// Security Configuration
	config := &security.SecurityConfig{
		TLS: struct {
			Enabled      bool     `yaml:"enabled"`
			MinVersion   string   `yaml:"min_version"`
			CipherSuites []string `yaml:"cipher_suites"`
			CertFile     string   `yaml:"cert_file"`
			KeyFile      string   `yaml:"key_file"`
			CAFile       string   `yaml:"ca_file"`
		}{
			Enabled:    true,
			MinVersion: "1.3",
		},
		IndustrialSecurity: struct {
			EncryptProtocols    bool `yaml:"encrypt_protocols"`
			RequireDeviceCerts  bool `yaml:"require_device_certs"`
			EncryptTagData      bool `yaml:"encrypt_tag_data"`
			AuditAllOperations  bool `yaml:"audit_all_operations"`
		}{
			EncryptTagData: true,
		},
		Audit: struct {
			Enabled     bool   `yaml:"enabled"`
			LogLevel    string `yaml:"log_level"`
			LogFile     string `yaml:"log_file"`
			MaxFileSize string `yaml:"max_file_size"`
			MaxBackups  int    `yaml:"max_backups"`
			MaxAge      int    `yaml:"max_age"`
		}{
			Enabled:  false, // Disabled for example
			LogLevel: "info",
		},
	}

	// Create security layer
	securityLayer, err := security.NewSecurityLayer(config)
	if err != nil {
		fmt.Printf("Failed to create security layer: %v\n", err)
		return
	}
	defer securityLayer.Close()

	fmt.Println("Security layer created successfully")

	// Example tag encryption
	originalTag := &protocols.Tag{
		ID:        "sensor_001",
		Name:      "temperature",
		Address:   "40001",
		DataType:  "float32",
		Value:     25.5,
		Quality:   "good",
		Timestamp: time.Now(),
		Writable:  false,
	}

	// Encrypt the tag
	encryptedTag, err := securityLayer.EncryptTag(originalTag)
	if err != nil {
		fmt.Printf("Failed to encrypt tag: %v\n", err)
		return
	}

	fmt.Printf("Original tag value: %v\n", originalTag.Value)
	fmt.Printf("Encrypted tag key ID: %s\n", encryptedTag.KeyID)
	fmt.Printf("Encrypted data length: %d bytes\n", len(encryptedTag.EncryptedValue))

	// Decrypt the tag
	decryptedTag, err := securityLayer.DecryptTag(encryptedTag)
	if err != nil {
		fmt.Printf("Failed to decrypt tag: %v\n", err)
		return
	}

	fmt.Printf("Decrypted tag value: %v\n", decryptedTag.Value)
	fmt.Printf("Encryption/decryption successful: %v\n", originalTag.Value == decryptedTag.Value)

	// Example audit event
	auditEvent := security.NewDataAccessEvent(
		"operator_001",
		"plc_001",
		security.SecurityActions.Read,
		"temperature_sensor",
		security.SecurityResults.Success,
		map[string]interface{}{
			"value":     25.5,
			"encrypted": true,
		},
	)

	if err := securityLayer.AuditEvent(auditEvent); err != nil {
		fmt.Printf("Failed to log audit event: %v\n", err)
	} else {
		fmt.Println("Audit event logged successfully")
	}
}

func topicExample() {
	// MQTT Topic Building
	mqttConfig := &messaging.MQTTConfig{
		Topics: struct {
			Telemetry   string `yaml:"telemetry"`
			Commands    string `yaml:"commands"`
			Alarms      string `yaml:"alarms"`
			Events      string `yaml:"events"`
			Diagnostics string `yaml:"diagnostics"`
		}{
			Telemetry: "factory/{site_id}/line/{device_id}/data/{tag_name}",
			Commands:  "factory/{site_id}/line/{device_id}/cmd/{command_type}",
			Alarms:    "factory/{site_id}/line/{device_id}/alarm/{alarm_level}",
		},
	}

	mqttBuilder := messaging.NewMQTTTopicBuilder(mqttConfig)

	fmt.Println("MQTT Topics:")
	fmt.Printf("  Telemetry: %s\n", mqttBuilder.TelemetryTopic("site1", "plc001", "temperature"))
	fmt.Printf("  Command:   %s\n", mqttBuilder.CommandTopic("site1", "plc001", "start"))
	fmt.Printf("  Alarm:     %s\n", mqttBuilder.AlarmTopic("site1", "plc001", "high"))

	// NATS Subject Building
	natsConfig := &messaging.NATSConfig{
		Subjects: struct {
			Telemetry    string `yaml:"telemetry"`
			Commands     string `yaml:"commands"`
			Events       string `yaml:"events"`
			Control      string `yaml:"control"`
			Coordination string `yaml:"coordination"`
		}{
			Telemetry:    "factory.{site_id}.{device_id}.telemetry.{tag_name}",
			Commands:     "factory.{site_id}.{device_id}.command.{command_type}",
			Control:      "factory.{site_id}.control.{zone}.{loop}",
			Coordination: "edge.{region}.gateway.{gateway_id}",
		},
	}

	natsBuilder := messaging.NewNATSTopicBuilder(natsConfig)

	fmt.Println("\nNATS Subjects:")
	fmt.Printf("  Telemetry:    %s\n", natsBuilder.TelemetryTopic("site1", "plc001", "temperature"))
	fmt.Printf("  Command:      %s\n", natsBuilder.CommandTopic("site1", "plc001", "start"))
	fmt.Printf("  Control:      %s\n", natsBuilder.ControlTopic("site1", "zone1", "loop1"))
	fmt.Printf("  Coordination: %s\n", natsBuilder.CoordinationTopic("us-east", "gateway001"))
}