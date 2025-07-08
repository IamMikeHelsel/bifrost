package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"bifrost-gateway/internal/protocols"
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	TLS struct {
		Enabled      bool     `yaml:"enabled"`
		MinVersion   string   `yaml:"min_version"`   // TLS 1.3
		CipherSuites []string `yaml:"cipher_suites"` // Industrial-grade only
		CertFile     string   `yaml:"cert_file"`
		KeyFile      string   `yaml:"key_file"`
		CAFile       string   `yaml:"ca_file"`
	} `yaml:"tls"`
	
	IndustrialSecurity struct {
		EncryptProtocols    bool `yaml:"encrypt_protocols"`    // Modbus over TLS
		RequireDeviceCerts  bool `yaml:"require_device_certs"` // mTLS for devices
		EncryptTagData      bool `yaml:"encrypt_tag_data"`     // AES-256-GCM
		AuditAllOperations  bool `yaml:"audit_all_operations"`
	} `yaml:"industrial_security"`
	
	KeyManagement struct {
		Provider         string        `yaml:"provider"`          // vault, aws-kms, azure-kv
		VaultAddress     string        `yaml:"vault_address"`
		KeyRotationEnabled bool        `yaml:"key_rotation_enabled"`
		RotationInterval time.Duration `yaml:"rotation_interval"`
		BackupEncryption bool          `yaml:"backup_encryption"`
	} `yaml:"key_management"`
	
	Audit struct {
		Enabled     bool   `yaml:"enabled"`
		LogLevel    string `yaml:"log_level"`
		LogFile     string `yaml:"log_file"`
		MaxFileSize string `yaml:"max_file_size"`
		MaxBackups  int    `yaml:"max_backups"`
		MaxAge      int    `yaml:"max_age"`
	} `yaml:"audit"`
}

// SecurityLayer provides encryption and security services
type SecurityLayer struct {
	config    *SecurityConfig
	tlsConfig *tls.Config
	
	// Encryption keys
	tagEncryptionKey []byte
	keyID           string
	
	// Audit logger
	auditLogger AuditLogger
}

// EncryptedTag represents an encrypted tag with metadata
type EncryptedTag struct {
	*protocols.Tag
	EncryptedValue []byte `json:"encrypted_value"`
	KeyID          string `json:"key_id"`
	Nonce          []byte `json:"nonce"`
	Signature      []byte `json:"signature"` // HMAC for integrity
}

// AuditEvent represents a security audit event
type AuditEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	UserID      string                 `json:"user_id,omitempty"`
	DeviceID    string                 `json:"device_id,omitempty"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource,omitempty"`
	Result      string                 `json:"result"` // success, failure, error
	Details     map[string]interface{} `json:"details,omitempty"`
	RemoteAddr  string                 `json:"remote_addr,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
}

// AuditLogger interface for audit logging implementations
type AuditLogger interface {
	LogEvent(event *AuditEvent) error
	Close() error
}

// NewSecurityLayer creates a new security layer instance
func NewSecurityLayer(config *SecurityConfig) (*SecurityLayer, error) {
	s := &SecurityLayer{
		config: config,
		keyID:  "tag-encryption-v1",
	}
	
	// Initialize TLS configuration
	if config.TLS.Enabled {
		tlsConfig, err := s.buildTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		s.tlsConfig = tlsConfig
	}
	
	// Initialize encryption key (in production, this would come from a key management system)
	if config.IndustrialSecurity.EncryptTagData {
		key := make([]byte, 32) // AES-256
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("failed to generate encryption key: %w", err)
		}
		s.tagEncryptionKey = key
	}
	
	// Initialize audit logger
	if config.Audit.Enabled {
		auditLogger, err := NewFileAuditLogger(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create audit logger: %w", err)
		}
		s.auditLogger = auditLogger
	}
	
	return s, nil
}

// GetTLSConfig returns the TLS configuration
func (s *SecurityLayer) GetTLSConfig() *tls.Config {
	return s.tlsConfig
}

// EncryptTag encrypts a tag value using AES-256-GCM
func (s *SecurityLayer) EncryptTag(tag *protocols.Tag) (*EncryptedTag, error) {
	if !s.config.IndustrialSecurity.EncryptTagData {
		return nil, fmt.Errorf("tag encryption is disabled")
	}
	
	if s.tagEncryptionKey == nil {
		return nil, fmt.Errorf("encryption key not available")
	}
	
	// Create AES cipher
	block, err := aes.NewCipher(s.tagEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// Marshal tag value to JSON
	valueBytes, err := json.Marshal(tag.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tag value: %w", err)
	}
	
	// Encrypt the value
	encrypted := gcm.Seal(nil, nonce, valueBytes, nil)
	
	return &EncryptedTag{
		Tag:            tag,
		EncryptedValue: encrypted,
		KeyID:          s.keyID,
		Nonce:          nonce,
	}, nil
}

// DecryptTag decrypts an encrypted tag
func (s *SecurityLayer) DecryptTag(encryptedTag *EncryptedTag) (*protocols.Tag, error) {
	if !s.config.IndustrialSecurity.EncryptTagData {
		return encryptedTag.Tag, nil
	}
	
	if s.tagEncryptionKey == nil {
		return nil, fmt.Errorf("encryption key not available")
	}
	
	// Create AES cipher
	block, err := aes.NewCipher(s.tagEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	// Decrypt the value
	decryptedBytes, err := gcm.Open(nil, encryptedTag.Nonce, encryptedTag.EncryptedValue, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt tag value: %w", err)
	}
	
	// Unmarshal the decrypted value
	var value interface{}
	if err := json.Unmarshal(decryptedBytes, &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decrypted value: %w", err)
	}
	
	// Create a copy of the tag with decrypted value
	decryptedTag := *encryptedTag.Tag
	decryptedTag.Value = value
	
	return &decryptedTag, nil
}

// AuditEvent logs a security audit event
func (s *SecurityLayer) AuditEvent(event *AuditEvent) error {
	if !s.config.Audit.Enabled || s.auditLogger == nil {
		return nil
	}
	
	event.Timestamp = time.Now()
	return s.auditLogger.LogEvent(event)
}

// buildTLSConfig constructs a TLS configuration based on security settings
func (s *SecurityLayer) buildTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13, // Default to TLS 1.3
	}
	
	// Set minimum TLS version
	switch s.config.TLS.MinVersion {
	case "1.2":
		tlsConfig.MinVersion = tls.VersionTLS12
	case "1.3":
		tlsConfig.MinVersion = tls.VersionTLS13
	default:
		tlsConfig.MinVersion = tls.VersionTLS13
	}
	
	// Set cipher suites for industrial-grade security
	if len(s.config.TLS.CipherSuites) > 0 {
		tlsConfig.CipherSuites = s.mapCipherSuites(s.config.TLS.CipherSuites)
	} else {
		// Default industrial-grade cipher suites
		tlsConfig.CipherSuites = []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
		}
	}
	
	// Load certificates if specified
	if s.config.TLS.CertFile != "" && s.config.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(s.config.TLS.CertFile, s.config.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	
	return tlsConfig, nil
}

// mapCipherSuites maps cipher suite names to their constants
func (s *SecurityLayer) mapCipherSuites(suites []string) []uint16 {
	cipherMap := map[string]uint16{
		"TLS_AES_256_GCM_SHA384":        tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256":  tls.TLS_CHACHA20_POLY1305_SHA256,
		"TLS_AES_128_GCM_SHA256":        tls.TLS_AES_128_GCM_SHA256,
	}
	
	var result []uint16
	for _, suite := range suites {
		if cipher, ok := cipherMap[suite]; ok {
			result = append(result, cipher)
		}
	}
	
	return result
}

// Close cleans up security resources
func (s *SecurityLayer) Close() error {
	if s.auditLogger != nil {
		return s.auditLogger.Close()
	}
	return nil
}