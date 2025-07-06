# Certificate Fixtures

## Purpose
Test certificates and security credentials for OPC UA and secure protocol testing.

## Planned Contents
- **OPC UA Certificates**: Client/server certificates for all security policies
- **CA Certificates**: Certificate authority chains for validation
- **Expired Certificates**: Invalid certificate testing scenarios
- **Self-signed Certificates**: Basic security testing

## Certificate Types
- RSA-2048 and RSA-4096 key pairs
- SHA-1 and SHA-256 signatures
- Client authentication certificates
- Server certificates with proper extensions
- Certificate revocation lists (CRL)

## Security Policies
- None (no security)
- Basic128Rsa15
- Basic256
- Basic256Sha256
- Aes128_Sha256_RsaOaep
- Aes256_Sha256_RsaPss

## Organization
- Separate folders by key size and algorithm
- Valid and invalid certificate sets
- Certificate chain hierarchies
- Revoked certificate scenarios

## Usage
- OPC UA security testing
- Certificate validation testing
- Authentication failure scenarios
- Security policy compliance