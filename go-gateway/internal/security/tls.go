package security

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"
)

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled      bool     `yaml:"enabled"`
	CertFile     string   `yaml:"cert_file"`
	KeyFile      string   `yaml:"key_file"`
	CAFile       string   `yaml:"ca_file"`
	MinVersion   string   `yaml:"min_version"`
	CipherSuites []string `yaml:"cipher_suites"`
}

// CertificateManager handles TLS certificate operations
type CertificateManager struct {
	config TLSConfig
	audit  *AuditLogger
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(config TLSConfig, audit *AuditLogger) *CertificateManager {
	return &CertificateManager{
		config: config,
		audit:  audit,
	}
}

// LoadTLSConfig loads and validates TLS configuration
func (cm *CertificateManager) LoadTLSConfig() (*tls.Config, error) {
	if !cm.config.Enabled {
		return nil, nil
	}

	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(cm.config.CertFile, cm.config.KeyFile)
	if err != nil {
		cm.audit.LogCryptoOperation("load_certificate", false, map[string]interface{}{
			"cert_file": cm.config.CertFile,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   cm.parseTLSVersion(cm.config.MinVersion),
		CipherSuites: cm.parseCipherSuites(cm.config.CipherSuites),
		Rand:         rand.Reader,
	}

	// Load CA certificates if specified
	if cm.config.CAFile != "" {
		caCert, err := ioutil.ReadFile(cm.config.CAFile)
		if err != nil {
			cm.audit.LogCryptoOperation("load_ca_certificate", false, map[string]interface{}{
				"ca_file": cm.config.CAFile,
				"error":   err.Error(),
			})
			return nil, fmt.Errorf("failed to load CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			cm.audit.LogCryptoOperation("parse_ca_certificate", false, map[string]interface{}{
				"ca_file": cm.config.CAFile,
			})
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	cm.audit.LogCryptoOperation("load_tls_config", true, map[string]interface{}{
		"min_version":    cm.config.MinVersion,
		"cipher_suites":  len(cm.config.CipherSuites),
		"client_auth":    tlsConfig.ClientAuth != tls.NoClientCert,
	})

	return tlsConfig, nil
}

// ValidateCertificate validates a certificate against the configured CA
func (cm *CertificateManager) ValidateCertificate(certPEM []byte) error {
	cert, err := x509.ParseCertificate(certPEM)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check expiration
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate not yet valid")
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate expired")
	}

	// Validate against CA if configured
	if cm.config.CAFile != "" {
		caCert, err := ioutil.ReadFile(cm.config.CAFile)
		if err != nil {
			return fmt.Errorf("failed to load CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to parse CA certificate")
		}

		opts := x509.VerifyOptions{
			Roots: caCertPool,
		}

		_, err = cert.Verify(opts)
		if err != nil {
			return fmt.Errorf("certificate validation failed: %w", err)
		}
	}

	cm.audit.LogCryptoOperation("validate_certificate", true, map[string]interface{}{
		"subject":    cert.Subject.String(),
		"issuer":     cert.Issuer.String(),
		"serial":     cert.SerialNumber.String(),
		"expires":    cert.NotAfter,
	})

	return nil
}

// GetCertificateInfo returns information about a certificate file
func (cm *CertificateManager) GetCertificateInfo(certFile string) (map[string]interface{}, error) {
	certPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	block, _ := x509.ParseCertificate(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate")
	}

	info := map[string]interface{}{
		"subject":     block.Subject.String(),
		"issuer":      block.Issuer.String(),
		"serial":      block.SerialNumber.String(),
		"not_before":  block.NotBefore,
		"not_after":   block.NotAfter,
		"is_ca":       block.IsCA,
		"key_usage":   block.KeyUsage,
		"ext_key_usage": block.ExtKeyUsage,
	}

	return info, nil
}

// parseTLSVersion converts string to TLS version constant
func (cm *CertificateManager) parseTLSVersion(version string) uint16 {
	switch version {
	case "TLS1.0":
		return tls.VersionTLS10
	case "TLS1.1":
		return tls.VersionTLS11
	case "TLS1.2":
		return tls.VersionTLS12
	case "TLS1.3":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS13 // Default to TLS 1.3
	}
}

// parseCipherSuites converts cipher suite names to constants
func (cm *CertificateManager) parseCipherSuites(suites []string) []uint16 {
	var cipherSuites []uint16
	
	cipherMap := map[string]uint16{
		"TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,
		// TLS 1.2 cipher suites
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	}

	for _, suite := range suites {
		if id, ok := cipherMap[suite]; ok {
			cipherSuites = append(cipherSuites, id)
		}
	}

	return cipherSuites
}