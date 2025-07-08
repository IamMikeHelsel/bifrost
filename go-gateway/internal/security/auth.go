package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthenticationManager handles user and device authentication
type AuthenticationManager struct {
	config AuthConfig
	audit  *AuditLogger
}

type AuthConfig struct {
	Enabled      bool          `yaml:"enabled"`
	Method       string        `yaml:"method"`        // jwt, certificate, basic
	SecretKey    string        `yaml:"secret_key"`
	TokenExpiry  time.Duration `yaml:"token_expiry"`
	RequireHTTPS bool          `yaml:"require_https"`
}

// User represents a user in the system
type User struct {
	ID           string            `json:"id"`
	Username     string            `json:"username"`
	PasswordHash string            `json:"-"` // Never expose password hash
	Roles        []string          `json:"roles"`
	Permissions  []string          `json:"permissions"`
	Enabled      bool              `json:"enabled"`
	CreatedAt    time.Time         `json:"created_at"`
	LastLogin    *time.Time        `json:"last_login,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// DeviceIdentity represents a device identity for authentication
type DeviceIdentity struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"` // plc, sensor, gateway
	Certificate  string            `json:"certificate,omitempty"`
	APIKey       string            `json:"-"` // Never expose API key
	Enabled      bool              `json:"enabled"`
	Capabilities []string          `json:"capabilities"`
	CreatedAt    time.Time         `json:"created_at"`
	LastSeen     *time.Time        `json:"last_seen,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// AuthenticationResult represents the result of an authentication attempt
type AuthenticationResult struct {
	Success     bool              `json:"success"`
	UserID      string            `json:"user_id,omitempty"`
	DeviceID    string            `json:"device_id,omitempty"`
	Roles       []string          `json:"roles,omitempty"`
	Permissions []string          `json:"permissions,omitempty"`
	Token       string            `json:"token,omitempty"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewAuthenticationManager creates a new authentication manager
func NewAuthenticationManager(config AuthConfig, audit *AuditLogger) *AuthenticationManager {
	return &AuthenticationManager{
		config: config,
		audit:  audit,
	}
}

// AuthenticateUser authenticates a user with username and password
func (am *AuthenticationManager) AuthenticateUser(username, password string) (*AuthenticationResult, error) {
	if !am.config.Enabled {
		return &AuthenticationResult{
			Success: true,
			UserID:  "anonymous",
			Roles:   []string{"anonymous"},
		}, nil
	}

	// In a real implementation, this would lookup the user from a database
	// For now, we'll use a simple in-memory check
	user, err := am.lookupUser(username)
	if err != nil {
		am.audit.LogAuthentication(username, "password", false, map[string]interface{}{
			"error": "user_not_found",
		})
		return &AuthenticationResult{
			Success: false,
			Error:   "authentication failed",
		}, nil
	}

	// Verify password
	if !am.verifyPassword(password, user.PasswordHash) {
		am.audit.LogAuthentication(username, "password", false, map[string]interface{}{
			"error": "invalid_password",
		})
		return &AuthenticationResult{
			Success: false,
			Error:   "authentication failed",
		}, nil
	}

	// Check if user is enabled
	if !user.Enabled {
		am.audit.LogAuthentication(username, "password", false, map[string]interface{}{
			"error": "user_disabled",
		})
		return &AuthenticationResult{
			Success: false,
			Error:   "account disabled",
		}, nil
	}

	// Generate token if using JWT
	var token string
	var expiresAt *time.Time
	if am.config.Method == "jwt" {
		// For now, we'll use a simple token approach
		// In production, use a proper JWT library
		tokenStr, err := am.generateToken(user.ID)
		if err != nil {
			return &AuthenticationResult{
				Success: false,
				Error:   "token generation failed",
			}, err
		}
		token = tokenStr
		expiry := time.Now().Add(am.config.TokenExpiry)
		expiresAt = &expiry
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now

	am.audit.LogAuthentication(username, "password", true, map[string]interface{}{
		"user_id": user.ID,
		"roles":   user.Roles,
	})

	return &AuthenticationResult{
		Success:     true,
		UserID:      user.ID,
		Roles:       user.Roles,
		Permissions: user.Permissions,
		Token:       token,
		ExpiresAt:   expiresAt,
	}, nil
}

// AuthenticateDevice authenticates a device using API key or certificate
func (am *AuthenticationManager) AuthenticateDevice(deviceID, apiKey string) (*AuthenticationResult, error) {
	if !am.config.Enabled {
		return &AuthenticationResult{
			Success:  true,
			DeviceID: deviceID,
			Roles:    []string{"device"},
		}, nil
	}

	device, err := am.lookupDevice(deviceID)
	if err != nil {
		am.audit.LogAuthentication(deviceID, "api_key", false, map[string]interface{}{
			"error": "device_not_found",
		})
		return &AuthenticationResult{
			Success: false,
			Error:   "authentication failed",
		}, nil
	}

	// Verify API key
	if !am.verifyAPIKey(apiKey, device.APIKey) {
		am.audit.LogAuthentication(deviceID, "api_key", false, map[string]interface{}{
			"error": "invalid_api_key",
		})
		return &AuthenticationResult{
			Success: false,
			Error:   "authentication failed",
		}, nil
	}

	// Check if device is enabled
	if !device.Enabled {
		am.audit.LogAuthentication(deviceID, "api_key", false, map[string]interface{}{
			"error": "device_disabled",
		})
		return &AuthenticationResult{
			Success: false,
			Error:   "device disabled",
		}, nil
	}

	// Update last seen
	now := time.Now()
	device.LastSeen = &now

	am.audit.LogAuthentication(deviceID, "api_key", true, map[string]interface{}{
		"device_id":    device.ID,
		"device_type":  device.Type,
		"capabilities": device.Capabilities,
	})

	return &AuthenticationResult{
		Success:     true,
		DeviceID:    device.ID,
		Roles:       []string{"device"},
		Permissions: device.Capabilities,
	}, nil
}

// HashPassword creates a bcrypt hash of a password
func (am *AuthenticationManager) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// verifyPassword verifies a password against its hash
func (am *AuthenticationManager) verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateAPIKey generates a secure API key
func (am *AuthenticationManager) GenerateAPIKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	return base64.URLEncoding.EncodeToString(key), nil
}

// verifyAPIKey verifies an API key using constant-time comparison
func (am *AuthenticationManager) verifyAPIKey(provided, stored string) bool {
	return subtle.ConstantTimeCompare([]byte(provided), []byte(stored)) == 1
}

// generateToken generates a simple token (in production, use JWT)
func (am *AuthenticationManager) generateToken(userID string) (string, error) {
	tokenData := make([]byte, 32)
	if _, err := rand.Read(tokenData); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(tokenData), nil
}

// Placeholder functions for user/device lookup
// In production, these would query a database
func (am *AuthenticationManager) lookupUser(username string) (*User, error) {
	// For demonstration, create a default admin user
	if username == "admin" {
		hash, _ := am.HashPassword("admin123")
		return &User{
			ID:           "admin",
			Username:     "admin",
			PasswordHash: hash,
			Roles:        []string{"admin", "operator"},
			Permissions:  []string{"read", "write", "configure"},
			Enabled:      true,
			CreatedAt:    time.Now(),
		}, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (am *AuthenticationManager) lookupDevice(deviceID string) (*DeviceIdentity, error) {
	// For demonstration, accept any device with ID starting with "device-"
	if len(deviceID) > 7 && deviceID[:7] == "device-" {
		// For testing, use a predictable API key
		return &DeviceIdentity{
			ID:           deviceID,
			Name:         fmt.Sprintf("Device %s", deviceID[7:]),
			Type:         "plc",
			APIKey:       "test-api-key", // Predictable for testing
			Enabled:      true,
			Capabilities: []string{"read", "write"},
			CreatedAt:    time.Now(),
		}, nil
	}
	return nil, fmt.Errorf("device not found")
}