# Security Framework Documentation

## Overview

The Bifrost Gateway Security Framework provides comprehensive encryption and security capabilities for industrial IoT environments. It is designed to meet the unique challenges of industrial systems where both security and performance are critical.

## Features

### 🔐 Transport Security
- **TLS 1.3 Support**: Modern, high-performance encryption for all network communications
- **Certificate Management**: Automated certificate validation and management
- **Cipher Suite Control**: Configurable cipher suites with secure defaults
- **Perfect Forward Secrecy**: Protection against future key compromises

### 🛡️ Authentication & Authorization
- **Multi-Factor Authentication**: Support for JWT tokens, API keys, and certificates
- **Device Identity Management**: Secure device enrollment and authentication
- **Role-Based Access Control**: Flexible permission system
- **Session Management**: Secure token handling with configurable expiration

### 🔒 Data Protection
- **AES-256-GCM Encryption**: Industry-standard authenticated encryption
- **Key Derivation**: PBKDF2-based key derivation from passwords
- **Hardware Acceleration**: Leverages AES-NI and ARM crypto extensions when available
- **Selective Encryption**: Encrypt only sensitive data to minimize performance impact

### 📊 Audit & Compliance
- **Comprehensive Logging**: All security events are logged with structured data
- **Tamper-Evident Trails**: Cryptographically signed audit logs
- **IEC 62443 Compliance**: Industrial cybersecurity standard compliance
- **Real-Time Monitoring**: Security event streaming for SIEM integration

## Configuration

### Basic Security Configuration

```yaml
security:
  enabled: true
  tls:
    enabled: true
    cert_file: "./certs/server-cert.pem"
    key_file: "./certs/server-key.pem"
    ca_file: "./certs/ca-cert.pem"
    min_version: "TLS1.3"
    cipher_suites:
      - "TLS_AES_256_GCM_SHA384"
      - "TLS_AES_128_GCM_SHA256"
  authentication:
    enabled: true
    method: "jwt"
    token_expiry: "24h"
    require_https: true
  audit:
    enabled: true
    log_file: "./logs/security-audit.log"
    log_level: "info"
```

### Protocol-Specific Security

```yaml
protocols:
  modbus:
    security:
      enable_tls: true
      require_auth: true
      encrypt_data: true
  opcua:
    security_policy: "Basic256Sha256"
    message_security: "SignAndEncrypt"
    certificate_file: "./certs/opcua-cert.pem"
    private_key_file: "./certs/opcua-key.pem"
```

## Usage Examples

### 1. User Authentication

```bash
# Login with username/password
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'

# Response
{
  "success": true,
  "user_id": "admin",
  "roles": ["admin", "operator"],
  "token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...",
  "expires_at": "2024-01-02T15:04:05Z"
}
```

### 2. Device Authentication

```bash
# Authenticate device with API key
curl -X POST http://localhost:8081/api/auth/device \
  -H "Content-Type: application/json" \
  -d '{"device_id": "device-plc-001", "api_key": "your-api-key"}'
```

### 3. Accessing Protected Endpoints

```bash
# Using JWT token
curl -H "Authorization: Bearer your-jwt-token" \
  http://localhost:8081/api/devices

# Using device API key
curl -H "X-API-Key: your-api-key" \
     -H "X-Device-ID: device-plc-001" \
  http://localhost:8081/api/tags/read
```

### 4. Security Status

```bash
# Check security configuration
curl http://localhost:8081/api/security/status

# Response
{
  "security_enabled": true,
  "tls_enabled": true,
  "authentication_enabled": true,
  "audit_enabled": true,
  "auth_method": "jwt",
  "tls_min_version": "TLS1.3"
}
```

## Security Events

The audit system logs comprehensive security events:

### Event Types
- `authentication`: Login attempts, token validation
- `authorization`: Permission checks, access control
- `data_access`: Reading/writing industrial data
- `configuration`: Security setting changes
- `connection`: Device connections/disconnections
- `cryptography`: Encryption/decryption operations
- `protocol`: Protocol-specific security events

### Sample Audit Log Entry

```json
{
  "timestamp": "2024-01-01T15:04:05Z",
  "event_type": "authentication",
  "severity": "info",
  "source": "gateway",
  "user_id": "admin",
  "action": "login",
  "result": "success",
  "message": "User admin authentication success",
  "details": {
    "ip_address": "192.168.1.100",
    "user_agent": "Industrial-Client/1.0"
  }
}
```

## Performance Considerations

### Encryption Performance
- **AES-256-GCM**: >500 MB/s on modern ARM processors
- **Hardware Acceleration**: Automatically detected and used
- **Selective Encryption**: Only sensitive data encrypted by default
- **Connection Pooling**: TLS connections are reused for efficiency

### Memory Usage
- **Base Security Framework**: ~5MB additional memory
- **Per Connection**: ~50KB additional overhead
- **Certificate Storage**: ~10KB per certificate

### Latency Impact
- **TLS Handshake**: ~20ms additional latency (one-time per connection)
- **Encryption/Decryption**: ~1-5µs per operation
- **Authentication**: ~10ms per token validation

## Certificate Management

### Generate Development Certificates

```bash
# Run the certificate generation script
cd security/
./generate-certs.sh

# This creates:
# - CA certificate and key
# - Server certificate and key
# - Client certificate and key
# - DH parameters for perfect forward secrecy
```

### Certificate Structure

```
certificates/
├── ca/
│   ├── ca-cert.pem      # Root CA certificate
│   └── ca-key.pem       # Root CA private key
├── server/
│   ├── server-cert.pem  # Server certificate
│   ├── server-key.pem   # Server private key
│   └── dhparam.pem      # Diffie-Hellman parameters
└── client/
    ├── client-cert.pem  # Client certificate
    └── client-key.pem   # Client private key
```

## Industrial Security Compliance

### IEC 62443 Security Levels

| Level | Description | Implementation Status |
|-------|-------------|----------------------|
| SL-1  | Protection against casual violation | ✅ Basic authentication |
| SL-2  | Protection against intentional violation | ✅ Enhanced access controls |
| SL-3  | Protection against sophisticated attacks | 🔄 Encryption + monitoring |
| SL-4  | Protection against state-sponsored attacks | ⚠️ Future consideration |

### NIST Cybersecurity Framework

| Function | Category | Implementation |
|----------|----------|----------------|
| **Identify** | Asset Management | Device inventory and classification |
| **Protect** | Access Control | Authentication and authorization |
| **Protect** | Data Security | Encryption and data protection |
| **Detect** | Security Monitoring | Real-time event logging |
| **Respond** | Response Planning | Automated security responses |
| **Recover** | Recovery Planning | Backup and recovery procedures |

## Best Practices

### 1. Production Deployment
- Always enable TLS in production environments
- Use strong, unique passwords and API keys
- Regularly rotate certificates and keys
- Monitor audit logs for suspicious activity
- Keep the gateway software updated

### 2. Network Security
- Deploy gateways in segmented networks
- Use firewalls to restrict access
- Monitor network traffic for anomalies
- Implement intrusion detection systems

### 3. Key Management
- Store private keys securely (HSM when possible)
- Use strong key derivation functions
- Implement automatic key rotation
- Maintain secure key backup procedures

### 4. Compliance
- Regular security assessments
- Penetration testing
- Vulnerability scanning
- Documentation maintenance

## Troubleshooting

### Common Issues

1. **Certificate Errors**
   ```
   Error: failed to load certificate
   Solution: Check file paths and permissions
   ```

2. **Authentication Failures**
   ```
   Error: authentication failed
   Solution: Verify credentials and check audit logs
   ```

3. **TLS Handshake Failures**
   ```
   Error: TLS handshake failed
   Solution: Check cipher suite compatibility
   ```

### Debug Mode

Enable debug logging for security troubleshooting:

```yaml
security:
  audit:
    log_level: "debug"
```

### Monitoring

Monitor these metrics for security health:
- Authentication success/failure rates
- Certificate expiration dates
- Encryption/decryption performance
- Audit log volume and patterns

## API Reference

### Authentication Endpoints

- `POST /api/auth/login` - User authentication
- `POST /api/auth/device` - Device authentication

### Security Management

- `GET /api/security/status` - Security configuration status
- `GET /api/security/certificates` - Certificate information

### Protected Endpoints

All API endpoints under `/api/` (except auth) require authentication when security is enabled.

## Migration Guide

### From Unsecured Gateway

1. **Enable Security Gradually**
   ```yaml
   security:
     enabled: true
     authentication:
       enabled: false  # Start with disabled auth
   ```

2. **Add TLS**
   ```yaml
   security:
     tls:
       enabled: true
   ```

3. **Enable Authentication**
   ```yaml
   security:
     authentication:
       enabled: true
   ```

### Backward Compatibility

The security framework is designed to be backward compatible:
- Existing API endpoints remain functional
- Security can be enabled incrementally
- Default configurations are secure but permissive
- Migration tools are provided for existing deployments

For more information, see the [Implementation Roadmap](../IMPLEMENTATION_ROADMAP.md) and [Security Testing Guide](../SECURITY_TESTING_ISSUE.md).