package messaging

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMQTTTopicBuilder(t *testing.T) {
	config := &MQTTConfig{
		Topics: struct {
			Telemetry   string `yaml:"telemetry"`
			Commands    string `yaml:"commands"`
			Alarms      string `yaml:"alarms"`
			Events      string `yaml:"events"`
			Diagnostics string `yaml:"diagnostics"`
		}{
			Telemetry: "test/telemetry/{site_id}/{device_id}/{tag_name}",
			Commands:  "test/commands/{site_id}/{device_id}/{command_type}",
		},
	}
	
	builder := NewMQTTTopicBuilder(config)
	
	// Test telemetry topic
	topic := builder.TelemetryTopic("site1", "device1", "temperature")
	expected := "test/telemetry/site1/device1/temperature"
	assert.Equal(t, expected, topic)
	
	// Test command topic
	topic = builder.CommandTopic("site1", "device1", "start")
	expected = "test/commands/site1/device1/start"
	assert.Equal(t, expected, topic)
	
	// Test default alarm topic
	topic = builder.AlarmTopic("site1", "device1", "high")
	expected = "bifrost/alarms/site1/device1/high"
	assert.Equal(t, expected, topic)
}

func TestNATSTopicBuilder(t *testing.T) {
	config := &NATSConfig{
		Subjects: struct {
			Telemetry    string `yaml:"telemetry"`
			Commands     string `yaml:"commands"`
			Events       string `yaml:"events"`
			Control      string `yaml:"control"`
			Coordination string `yaml:"coordination"`
		}{
			Telemetry: "test.telemetry.{site_id}.{device_id}.{tag_name}",
			Commands:  "test.commands.{site_id}.{device_id}.{command_type}",
		},
	}
	
	builder := NewNATSTopicBuilder(config)
	
	// Test telemetry subject
	subject := builder.TelemetryTopic("site1", "device1", "temperature")
	expected := "test.telemetry.site1.device1.temperature"
	assert.Equal(t, expected, subject)
	
	// Test command subject
	subject = builder.CommandTopic("site1", "device1", "start")
	expected = "test.commands.site1.device1.start"
	assert.Equal(t, expected, subject)
	
	// Test subject sanitization
	subject = builder.TelemetryTopic("site-1", "device/1", "temp sensor")
	expected = "test.telemetry.site_1.device.1.temp_sensor"
	assert.Equal(t, expected, subject)
}

func TestMQTTConfig(t *testing.T) {
	config := &MQTTConfig{
		Enabled:        true,
		Broker:         "tcp://localhost:1883",
		ClientID:       "test-client",
		QoS:            1,
		KeepAlive:      60 * time.Second,
		ConnectTimeout: 30 * time.Second,
		WriteTimeout:   10 * time.Second,
		AutoReconnect:  true,
		CleanSession:   true,
	}
	
	logger := zap.NewNop()
	
	// Test MQTT messaging creation (will fail to connect but should create instance)
	mqtt, err := NewMQTTMessaging(config, logger)
	assert.NoError(t, err)
	assert.NotNil(t, mqtt)
	assert.False(t, mqtt.IsConnected()) // Not connected yet
}

func TestNATSConfig(t *testing.T) {
	config := &NATSConfig{
		Enabled:        true,
		Servers:        []string{"localhost:4222"},
		ClientID:       "test-client",
		MaxReconnects:  10,
		ReconnectWait:  2 * time.Second,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 10 * time.Second,
		DrainTimeout:   5 * time.Second,
	}
	
	logger := zap.NewNop()
	
	// Test NATS messaging creation
	nats, err := NewNATSMessaging(config, logger)
	assert.NoError(t, err)
	assert.NotNil(t, nats)
	assert.False(t, nats.IsConnected()) // Not connected yet
}

func TestMessagingDisabled(t *testing.T) {
	logger := zap.NewNop()
	
	// Test disabled MQTT
	mqttConfig := &MQTTConfig{Enabled: false}
	_, err := NewMQTTMessaging(mqttConfig, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
	
	// Test disabled NATS
	natsConfig := &NATSConfig{Enabled: false}
	_, err = NewNATSMessaging(natsConfig, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}