package messaging

import (
	"fmt"
	"strings"
)

// NATSTopicBuilder implements the TopicBuilder interface for NATS
type NATSTopicBuilder struct {
	config *NATSConfig
}

// NewNATSTopicBuilder creates a new NATS topic builder
func NewNATSTopicBuilder(config *NATSConfig) *NATSTopicBuilder {
	return &NATSTopicBuilder{
		config: config,
	}
}

// TelemetryTopic builds a telemetry subject (NATS uses subjects, not topics)
func (tb *NATSTopicBuilder) TelemetryTopic(siteID, deviceID, tagName string) string {
	template := tb.config.Subjects.Telemetry
	if template == "" {
		template = "bifrost.telemetry.{site_id}.{device_id}.{tag_name}"
	}
	
	subject := strings.ReplaceAll(template, "{site_id}", siteID)
	subject = strings.ReplaceAll(subject, "{device_id}", deviceID)
	subject = strings.ReplaceAll(subject, "{tag_name}", tagName)
	
	return sanitizeSubject(subject)
}

// CommandTopic builds a command subject
func (tb *NATSTopicBuilder) CommandTopic(siteID, deviceID, commandType string) string {
	template := tb.config.Subjects.Commands
	if template == "" {
		template = "bifrost.commands.{site_id}.{device_id}.{command_type}"
	}
	
	subject := strings.ReplaceAll(template, "{site_id}", siteID)
	subject = strings.ReplaceAll(subject, "{device_id}", deviceID)
	subject = strings.ReplaceAll(subject, "{command_type}", commandType)
	
	return sanitizeSubject(subject)
}

// AlarmTopic builds an alarm subject
func (tb *NATSTopicBuilder) AlarmTopic(siteID, deviceID, alarmLevel string) string {
	template := "bifrost.alarms.{site_id}.{device_id}.{alarm_level}"
	
	subject := strings.ReplaceAll(template, "{site_id}", siteID)
	subject = strings.ReplaceAll(subject, "{device_id}", deviceID)
	subject = strings.ReplaceAll(subject, "{alarm_level}", alarmLevel)
	
	return sanitizeSubject(subject)
}

// EventTopic builds an event subject
func (tb *NATSTopicBuilder) EventTopic(siteID, deviceID, eventType string) string {
	template := tb.config.Subjects.Events
	if template == "" {
		template = "bifrost.events.{site_id}.{device_id}.{event_type}"
	}
	
	subject := strings.ReplaceAll(template, "{site_id}", siteID)
	subject = strings.ReplaceAll(subject, "{device_id}", deviceID)
	subject = strings.ReplaceAll(subject, "{event_type}", eventType)
	
	return sanitizeSubject(subject)
}

// DiagnosticTopic builds a diagnostic subject
func (tb *NATSTopicBuilder) DiagnosticTopic(siteID, deviceID, metricName string) string {
	template := "bifrost.diagnostics.{site_id}.{device_id}.{metric_name}"
	
	subject := strings.ReplaceAll(template, "{site_id}", siteID)
	subject = strings.ReplaceAll(subject, "{device_id}", deviceID)
	subject = strings.ReplaceAll(subject, "{metric_name}", metricName)
	
	return sanitizeSubject(subject)
}

// ControlTopic builds a control subject for real-time control loops
func (tb *NATSTopicBuilder) ControlTopic(siteID, zone, loop string) string {
	template := tb.config.Subjects.Control
	if template == "" {
		template = "bifrost.control.{site_id}.{zone}.{loop}"
	}
	
	subject := strings.ReplaceAll(template, "{site_id}", siteID)
	subject = strings.ReplaceAll(subject, "{zone}", zone)
	subject = strings.ReplaceAll(subject, "{loop}", loop)
	
	return sanitizeSubject(subject)
}

// CoordinationTopic builds a coordination subject for edge computing
func (tb *NATSTopicBuilder) CoordinationTopic(region, gatewayID string) string {
	template := tb.config.Subjects.Coordination
	if template == "" {
		template = "bifrost.coordination.{region}.{gateway_id}"
	}
	
	subject := strings.ReplaceAll(template, "{region}", region)
	subject = strings.ReplaceAll(subject, "{gateway_id}", gatewayID)
	
	return sanitizeSubject(subject)
}

// ValidateSubject validates that a subject follows NATS conventions
func (tb *NATSTopicBuilder) ValidateSubject(subject string) error {
	if subject == "" {
		return fmt.Errorf("subject cannot be empty")
	}
	
	if len(subject) > 255 {
		return fmt.Errorf("subject too long (max 255 characters)")
	}
	
	// Check for invalid characters
	invalidChars := []string{" ", "\t", "\n", "\r"}
	for _, char := range invalidChars {
		if strings.Contains(subject, char) {
			return fmt.Errorf("subject contains invalid character: %q", char)
		}
	}
	
	// Check wildcard usage
	if strings.Contains(subject, "*") && !isValidNATSWildcard(subject, "*") {
		return fmt.Errorf("invalid wildcard usage")
	}
	
	if strings.Contains(subject, ">") && !isValidNATSWildcard(subject, ">") {
		return fmt.Errorf("invalid wildcard usage")
	}
	
	return nil
}

// sanitizeSubject removes invalid characters and converts to valid NATS subject format
func sanitizeSubject(subject string) string {
	// Replace common invalid characters
	subject = strings.ReplaceAll(subject, " ", "_")
	subject = strings.ReplaceAll(subject, "-", "_")
	subject = strings.ReplaceAll(subject, "/", ".")
	
	// Convert to lowercase for consistency
	subject = strings.ToLower(subject)
	
	return subject
}

// isValidNATSWildcard checks if wildcard usage is valid for NATS
func isValidNATSWildcard(subject, wildcard string) bool {
	if wildcard == ">" {
		// Multi-token wildcard must be at the end and preceded by a dot
		return strings.HasSuffix(subject, ">") && (len(subject) == 1 || strings.HasSuffix(subject, ".>"))
	}
	
	if wildcard == "*" {
		// Single-token wildcard must be a complete token between dots
		tokens := strings.Split(subject, ".")
		for _, token := range tokens {
			if token == "*" {
				continue
			}
			if strings.Contains(token, "*") {
				return false
			}
		}
		return true
	}
	
	return false
}