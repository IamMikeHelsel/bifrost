package security

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bifrost-gateway/internal/protocols"
)

func TestSecurityLayer(t *testing.T) {
	config := &SecurityConfig{
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
			Enabled:  false, // Disable for testing
			LogLevel: "info",
		},
	}
	
	security, err := NewSecurityLayer(config)
	require.NoError(t, err)
	require.NotNil(t, security)
	
	// Test tag encryption/decryption
	originalTag := &protocols.Tag{
		ID:        "test-tag-1",
		Name:      "temperature",
		Address:   "40001",
		DataType:  "float32",
		Value:     25.5,
		Quality:   "good",
		Timestamp: time.Now(),
		Writable:  false,
	}
	
	// Encrypt the tag
	encryptedTag, err := security.EncryptTag(originalTag)
	require.NoError(t, err)
	require.NotNil(t, encryptedTag)
	assert.Equal(t, originalTag.ID, encryptedTag.ID)
	assert.Equal(t, originalTag.Name, encryptedTag.Name)
	assert.NotEmpty(t, encryptedTag.EncryptedValue)
	assert.NotEmpty(t, encryptedTag.Nonce)
	assert.Equal(t, "tag-encryption-v1", encryptedTag.KeyID)
	
	// Decrypt the tag
	decryptedTag, err := security.DecryptTag(encryptedTag)
	require.NoError(t, err)
	require.NotNil(t, decryptedTag)
	assert.Equal(t, originalTag.ID, decryptedTag.ID)
	assert.Equal(t, originalTag.Name, decryptedTag.Name)
	assert.Equal(t, originalTag.Value, decryptedTag.Value)
	assert.Equal(t, originalTag.Quality, decryptedTag.Quality)
}

func TestSecurityLayerDisabled(t *testing.T) {
	config := &SecurityConfig{
		IndustrialSecurity: struct {
			EncryptProtocols    bool `yaml:"encrypt_protocols"`
			RequireDeviceCerts  bool `yaml:"require_device_certs"`
			EncryptTagData      bool `yaml:"encrypt_tag_data"`
			AuditAllOperations  bool `yaml:"audit_all_operations"`
		}{
			EncryptTagData: false, // Disabled
		},
		Audit: struct {
			Enabled     bool   `yaml:"enabled"`
			LogLevel    string `yaml:"log_level"`
			LogFile     string `yaml:"log_file"`
			MaxFileSize string `yaml:"max_file_size"`
			MaxBackups  int    `yaml:"max_backups"`
			MaxAge      int    `yaml:"max_age"`
		}{
			Enabled: false,
		},
	}
	
	security, err := NewSecurityLayer(config)
	require.NoError(t, err)
	require.NotNil(t, security)
	
	originalTag := &protocols.Tag{
		ID:       "test-tag-1",
		Name:     "temperature",
		Value:    25.5,
		Quality:  "good",
		Writable: false,
	}
	
	// Try to encrypt when disabled - should fail
	_, err = security.EncryptTag(originalTag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestTLSConfiguration(t *testing.T) {
	config := &SecurityConfig{
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
			CipherSuites: []string{
				"TLS_AES_256_GCM_SHA384",
				"TLS_CHACHA20_POLY1305_SHA256",
			},
		},
		Audit: struct {
			Enabled     bool   `yaml:"enabled"`
			LogLevel    string `yaml:"log_level"`
			LogFile     string `yaml:"log_file"`
			MaxFileSize string `yaml:"max_file_size"`
			MaxBackups  int    `yaml:"max_backups"`
			MaxAge      int    `yaml:"max_age"`
		}{
			Enabled: false,
		},
	}
	
	security, err := NewSecurityLayer(config)
	require.NoError(t, err)
	require.NotNil(t, security)
	
	tlsConfig := security.GetTLSConfig()
	require.NotNil(t, tlsConfig)
	
	// Check TLS version
	assert.Equal(t, uint16(0x0304), tlsConfig.MinVersion) // TLS 1.3
	
	// Check cipher suites
	assert.NotEmpty(t, tlsConfig.CipherSuites)
}

func TestAuditEvents(t *testing.T) {
	// Test audit event creation helpers
	authEvent := NewAuthenticationEvent(
		"user123",
		SecurityActions.Login,
		SecurityResults.Success,
		"192.168.1.100",
		map[string]interface{}{"method": "certificate"},
	)
	
	assert.Equal(t, SecurityEventTypes.Authentication, authEvent.EventType)
	assert.Equal(t, "user123", authEvent.UserID)
	assert.Equal(t, SecurityActions.Login, authEvent.Action)
	assert.Equal(t, SecurityResults.Success, authEvent.Result)
	assert.Equal(t, "192.168.1.100", authEvent.RemoteAddr)
	assert.NotNil(t, authEvent.Details)
	
	dataEvent := NewDataAccessEvent(
		"user123",
		"device001",
		SecurityActions.Read,
		"temperature_sensor",
		SecurityResults.Success,
		map[string]interface{}{"value": 25.5},
	)
	
	assert.Equal(t, SecurityEventTypes.DataAccess, dataEvent.EventType)
	assert.Equal(t, "user123", dataEvent.UserID)
	assert.Equal(t, "device001", dataEvent.DeviceID)
	assert.Equal(t, SecurityActions.Read, dataEvent.Action)
	assert.Equal(t, "temperature_sensor", dataEvent.Resource)
	
	protocolEvent := NewProtocolSecurityEvent(
		"device001",
		SecurityActions.Connect,
		SecurityResults.Success,
		map[string]interface{}{"protocol": "modbus", "encrypted": true},
	)
	
	assert.Equal(t, SecurityEventTypes.ProtocolSecurity, protocolEvent.EventType)
	assert.Equal(t, "device001", protocolEvent.DeviceID)
	assert.Equal(t, SecurityActions.Connect, protocolEvent.Action)
	assert.Equal(t, SecurityResults.Success, protocolEvent.Result)
}