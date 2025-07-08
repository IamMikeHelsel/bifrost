package security

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticationManager(t *testing.T) {
	// Create test audit logger
	auditConfig := AuditConfig{
		Enabled:  false, // Disable for testing
		LogLevel: "info",
	}
	audit, err := NewAuditLogger(auditConfig)
	require.NoError(t, err)

	// Create authentication manager
	authConfig := AuthConfig{
		Enabled:      true,
		Method:       "jwt",
		TokenExpiry:  24 * time.Hour,
		RequireHTTPS: false,
	}
	authManager := NewAuthenticationManager(authConfig, audit)

	// Test user authentication (using built-in admin user)
	result, err := authManager.AuthenticateUser("admin", "admin123")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "admin", result.UserID)
	assert.Contains(t, result.Roles, "admin")
	assert.NotEmpty(t, result.Token)

	// Test invalid user authentication
	result, err = authManager.AuthenticateUser("invalid", "password")
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Empty(t, result.UserID)

	// Test device authentication
	result, err = authManager.AuthenticateDevice("device-test", "test-api-key")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "device-test", result.DeviceID)
	assert.Contains(t, result.Roles, "device")

	// Test API key generation
	apiKey, err := authManager.GenerateAPIKey()
	require.NoError(t, err)
	assert.NotEmpty(t, apiKey)
	assert.True(t, len(apiKey) > 20) // Should be a reasonable length

	// Test password hashing
	password := "test-password"
	hash, err := authManager.HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestAuditLogger(t *testing.T) {
	// Create temporary log file
	tmpFile, err := ioutil.TempFile("", "audit-test-*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create audit logger
	config := AuditConfig{
		Enabled:  true,
		LogFile:  tmpFile.Name(),
		LogLevel: "info",
	}

	audit, err := NewAuditLogger(config)
	require.NoError(t, err)
	defer audit.Close()

	// Test logging an event
	event := SecurityEvent{
		EventType: EventTypeAuthentication,
		Severity:  SeverityInfo,
		Source:    "test",
		Action:    "login",
		Result:    ResultSuccess,
		Message:   "Test authentication",
		Details: map[string]interface{}{
			"user_id": "test-user",
		},
	}

	audit.LogEvent(event)

	// Test convenience methods
	audit.LogAuthentication("test-user", "api", true, map[string]interface{}{
		"ip": "127.0.0.1",
	})

	audit.LogDataAccess("test-user", "device-1", "read", true, map[string]interface{}{
		"tags": []string{"tag1", "tag2"},
	})

	audit.LogCryptoOperation("encrypt", true, map[string]interface{}{
		"algorithm": "AES-256-GCM",
	})

	// Sync and verify log file was written
	err = audit.Close()
	require.NoError(t, err)

	logContent, err := ioutil.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, string(logContent), "Test authentication")
	assert.Contains(t, string(logContent), "test-user")
}

func TestTLSConfig(t *testing.T) {
	// Create mock audit logger
	auditConfig := AuditConfig{
		Enabled: false,
	}
	audit, err := NewAuditLogger(auditConfig)
	require.NoError(t, err)

	// Test TLS config with disabled TLS
	tlsConfig := TLSConfig{
		Enabled: false,
	}

	certManager := NewCertificateManager(tlsConfig, audit)
	config, err := certManager.LoadTLSConfig()
	require.NoError(t, err)
	assert.Nil(t, config) // Should be nil when disabled

	// Test TLS version parsing
	certManagerEnabled := NewCertificateManager(TLSConfig{
		Enabled:    false, // We don't have certificates for testing
		MinVersion: "TLS1.3",
		CipherSuites: []string{
			"TLS_AES_256_GCM_SHA384",
			"TLS_AES_128_GCM_SHA256",
		},
	}, audit)

	// Test private methods through reflection or by testing the whole flow
	// For now, just verify the manager was created
	assert.NotNil(t, certManagerEnabled)
}