package messaging

import (
	"fmt"
	"strings"
)

// MQTTTopicBuilder implements the TopicBuilder interface for MQTT
type MQTTTopicBuilder struct {
	config *MQTTConfig
}

// NewMQTTTopicBuilder creates a new MQTT topic builder
func NewMQTTTopicBuilder(config *MQTTConfig) *MQTTTopicBuilder {
	return &MQTTTopicBuilder{
		config: config,
	}
}

// TelemetryTopic builds a telemetry topic
func (tb *MQTTTopicBuilder) TelemetryTopic(siteID, deviceID, tagName string) string {
	template := tb.config.Topics.Telemetry
	if template == "" {
		template = "bifrost/telemetry/{site_id}/{device_id}/{tag_name}"
	}
	
	topic := strings.ReplaceAll(template, "{site_id}", siteID)
	topic = strings.ReplaceAll(topic, "{device_id}", deviceID)
	topic = strings.ReplaceAll(topic, "{tag_name}", tagName)
	
	return topic
}

// CommandTopic builds a command topic
func (tb *MQTTTopicBuilder) CommandTopic(siteID, deviceID, commandType string) string {
	template := tb.config.Topics.Commands
	if template == "" {
		template = "bifrost/commands/{site_id}/{device_id}/{command_type}"
	}
	
	topic := strings.ReplaceAll(template, "{site_id}", siteID)
	topic = strings.ReplaceAll(topic, "{device_id}", deviceID)
	topic = strings.ReplaceAll(topic, "{command_type}", commandType)
	
	return topic
}

// AlarmTopic builds an alarm topic
func (tb *MQTTTopicBuilder) AlarmTopic(siteID, deviceID, alarmLevel string) string {
	template := tb.config.Topics.Alarms
	if template == "" {
		template = "bifrost/alarms/{site_id}/{device_id}/{alarm_level}"
	}
	
	topic := strings.ReplaceAll(template, "{site_id}", siteID)
	topic = strings.ReplaceAll(topic, "{device_id}", deviceID)
	topic = strings.ReplaceAll(topic, "{alarm_level}", alarmLevel)
	
	return topic
}

// EventTopic builds an event topic
func (tb *MQTTTopicBuilder) EventTopic(siteID, deviceID, eventType string) string {
	template := tb.config.Topics.Events
	if template == "" {
		template = "bifrost/events/{site_id}/{device_id}/{event_type}"
	}
	
	topic := strings.ReplaceAll(template, "{site_id}", siteID)
	topic = strings.ReplaceAll(topic, "{device_id}", deviceID)
	topic = strings.ReplaceAll(topic, "{event_type}", eventType)
	
	return topic
}

// DiagnosticTopic builds a diagnostic topic
func (tb *MQTTTopicBuilder) DiagnosticTopic(siteID, deviceID, metricName string) string {
	template := tb.config.Topics.Diagnostics
	if template == "" {
		template = "bifrost/diagnostics/{site_id}/{device_id}/{metric_name}"
	}
	
	topic := strings.ReplaceAll(template, "{site_id}", siteID)
	topic = strings.ReplaceAll(topic, "{device_id}", deviceID)
	topic = strings.ReplaceAll(topic, "{metric_name}", metricName)
	
	return topic
}

// ValidateTopic validates that a topic follows MQTT conventions
func (tb *MQTTTopicBuilder) ValidateTopic(topic string) error {
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	
	if len(topic) > 65535 {
		return fmt.Errorf("topic too long (max 65535 characters)")
	}
	
	// Check for invalid characters
	if strings.Contains(topic, "+") && !isValidWildcard(topic, "+") {
		return fmt.Errorf("invalid single-level wildcard usage")
	}
	
	if strings.Contains(topic, "#") && !isValidWildcard(topic, "#") {
		return fmt.Errorf("invalid multi-level wildcard usage")
	}
	
	return nil
}

// isValidWildcard checks if wildcard usage is valid
func isValidWildcard(topic, wildcard string) bool {
	// This is a simplified validation - full MQTT spec validation would be more complex
	if wildcard == "#" {
		// Multi-level wildcard must be at the end
		return strings.HasSuffix(topic, "#") && (len(topic) == 1 || strings.HasSuffix(topic, "/#"))
	}
	
	if wildcard == "+" {
		// Single-level wildcard must be between slashes or at boundaries
		parts := strings.Split(topic, "/")
		for _, part := range parts {
			if part == "+" {
				continue
			}
			if strings.Contains(part, "+") {
				return false
			}
		}
		return true
	}
	
	return false
}