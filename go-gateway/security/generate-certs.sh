#!/bin/bash

# Certificate Generation Script for Bifrost Gateway
# This script generates TLS certificates for secure communication

set -euo pipefail

CERT_DIR="./certificates"
CA_DIR="${CERT_DIR}/ca"
SERVER_DIR="${CERT_DIR}/server"
CLIENT_DIR="${CERT_DIR}/client"

# Configuration
COUNTRY="US"
STATE="California"
CITY="San Francisco"
ORG="Bifrost"
OU="Industrial Gateway"
COMMON_NAME="bifrost-gateway.local"
EMAIL="admin@bifrost.local"

# Create directories
mkdir -p "${CA_DIR}" "${SERVER_DIR}" "${CLIENT_DIR}"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Generate CA private key
generate_ca_key() {
    log "Generating CA private key..."
    openssl genrsa -out "${CA_DIR}/ca-key.pem" 4096
    chmod 400 "${CA_DIR}/ca-key.pem"
}

# Generate CA certificate
generate_ca_cert() {
    log "Generating CA certificate..."
    cat > "${CA_DIR}/ca.conf" <<EOF
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_ca
prompt = no

[req_distinguished_name]
C = ${COUNTRY}
ST = ${STATE}
L = ${CITY}
O = ${ORG}
OU = ${OU}
CN = ${ORG} Root CA
emailAddress = ${EMAIL}

[v3_ca]
basicConstraints = critical,CA:TRUE
keyUsage = critical,digitalSignature,keyCertSign
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always,issuer:always
EOF

    openssl req -new -x509 -days 3650 -key "${CA_DIR}/ca-key.pem" \
        -out "${CA_DIR}/ca-cert.pem" -config "${CA_DIR}/ca.conf"
    
    chmod 444 "${CA_DIR}/ca-cert.pem"
}

# Generate server private key
generate_server_key() {
    log "Generating server private key..."
    openssl genrsa -out "${SERVER_DIR}/server-key.pem" 2048
    chmod 400 "${SERVER_DIR}/server-key.pem"
}

# Generate server certificate
generate_server_cert() {
    log "Generating server certificate..."
    cat > "${SERVER_DIR}/server.conf" <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = ${COUNTRY}
ST = ${STATE}
L = ${CITY}
O = ${ORG}
OU = ${OU}
CN = ${COMMON_NAME}
emailAddress = ${EMAIL}

[v3_req]
basicConstraints = CA:FALSE
keyUsage = keyEncipherment,dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = bifrost-gateway.local
DNS.2 = bifrost-gateway
DNS.3 = bifrost-gateway.bifrost-system.svc.cluster.local
DNS.4 = localhost
IP.1 = 127.0.0.1
IP.2 = 10.0.0.1
EOF

    # Generate server CSR
    openssl req -new -key "${SERVER_DIR}/server-key.pem" \
        -out "${SERVER_DIR}/server.csr" -config "${SERVER_DIR}/server.conf"
    
    # Sign server certificate with CA
    openssl x509 -req -in "${SERVER_DIR}/server.csr" -days 365 \
        -CA "${CA_DIR}/ca-cert.pem" -CAkey "${CA_DIR}/ca-key.pem" \
        -out "${SERVER_DIR}/server-cert.pem" -extensions v3_req \
        -extfile "${SERVER_DIR}/server.conf" -CAcreateserial
    
    chmod 444 "${SERVER_DIR}/server-cert.pem"
    rm "${SERVER_DIR}/server.csr"
}

# Generate client private key
generate_client_key() {
    log "Generating client private key..."
    openssl genrsa -out "${CLIENT_DIR}/client-key.pem" 2048
    chmod 400 "${CLIENT_DIR}/client-key.pem"
}

# Generate client certificate
generate_client_cert() {
    log "Generating client certificate..."
    cat > "${CLIENT_DIR}/client.conf" <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = ${COUNTRY}
ST = ${STATE}
L = ${CITY}
O = ${ORG}
OU = ${OU}
CN = bifrost-client
emailAddress = ${EMAIL}

[v3_req]
basicConstraints = CA:FALSE
keyUsage = keyEncipherment,dataEncipherment,digitalSignature
extendedKeyUsage = clientAuth
EOF

    # Generate client CSR
    openssl req -new -key "${CLIENT_DIR}/client-key.pem" \
        -out "${CLIENT_DIR}/client.csr" -config "${CLIENT_DIR}/client.conf"
    
    # Sign client certificate with CA
    openssl x509 -req -in "${CLIENT_DIR}/client.csr" -days 365 \
        -CA "${CA_DIR}/ca-cert.pem" -CAkey "${CA_DIR}/ca-key.pem" \
        -out "${CLIENT_DIR}/client-cert.pem" -extensions v3_req \
        -extfile "${CLIENT_DIR}/client.conf" -CAcreateserial
    
    chmod 444 "${CLIENT_DIR}/client-cert.pem"
    rm "${CLIENT_DIR}/client.csr"
}

# Generate DH parameters
generate_dh_params() {
    log "Generating DH parameters..."
    openssl dhparam -out "${SERVER_DIR}/dhparam.pem" 2048
    chmod 444 "${SERVER_DIR}/dhparam.pem"
}

# Create Kubernetes secrets
create_k8s_secrets() {
    log "Creating Kubernetes TLS secret..."
    
    if command -v kubectl >/dev/null 2>&1; then
        # Create TLS secret for server
        kubectl create secret tls bifrost-gateway-tls \
            --cert="${SERVER_DIR}/server-cert.pem" \
            --key="${SERVER_DIR}/server-key.pem" \
            --namespace=bifrost-system \
            --dry-run=client -o yaml > "${CERT_DIR}/bifrost-tls-secret.yaml"
        
        # Create CA secret
        kubectl create secret generic bifrost-ca \
            --from-file=ca.crt="${CA_DIR}/ca-cert.pem" \
            --namespace=bifrost-system \
            --dry-run=client -o yaml > "${CERT_DIR}/bifrost-ca-secret.yaml"
        
        # Create client certificate secret
        kubectl create secret tls bifrost-client-tls \
            --cert="${CLIENT_DIR}/client-cert.pem" \
            --key="${CLIENT_DIR}/client-key.pem" \
            --namespace=bifrost-system \
            --dry-run=client -o yaml > "${CERT_DIR}/bifrost-client-secret.yaml"
        
        log "Kubernetes secret manifests created in ${CERT_DIR}"
    else
        log "kubectl not found, skipping Kubernetes secret creation"
    fi
}

# Verify certificates
verify_certs() {
    log "Verifying certificates..."
    
    # Verify CA certificate
    openssl x509 -in "${CA_DIR}/ca-cert.pem" -text -noout | head -20
    
    # Verify server certificate
    openssl verify -CAfile "${CA_DIR}/ca-cert.pem" "${SERVER_DIR}/server-cert.pem"
    
    # Verify client certificate
    openssl verify -CAfile "${CA_DIR}/ca-cert.pem" "${CLIENT_DIR}/client-cert.pem"
    
    # Check certificate details
    log "Server certificate details:"
    openssl x509 -in "${SERVER_DIR}/server-cert.pem" -noout -subject -issuer -dates
    
    log "Client certificate details:"
    openssl x509 -in "${CLIENT_DIR}/client-cert.pem" -noout -subject -issuer -dates
}

# Create certificate bundle
create_bundle() {
    log "Creating certificate bundle..."
    
    # Create full chain certificate
    cat "${SERVER_DIR}/server-cert.pem" "${CA_DIR}/ca-cert.pem" > "${SERVER_DIR}/server-fullchain.pem"
    
    # Create PKCS#12 bundle for client
    openssl pkcs12 -export -out "${CLIENT_DIR}/client.p12" \
        -inkey "${CLIENT_DIR}/client-key.pem" \
        -in "${CLIENT_DIR}/client-cert.pem" \
        -certfile "${CA_DIR}/ca-cert.pem" \
        -passout pass:bifrost
    
    log "Certificate bundle created"
}

# Set proper permissions
set_permissions() {
    log "Setting certificate permissions..."
    
    # Set restrictive permissions on private keys
    chmod 400 "${CA_DIR}/ca-key.pem"
    chmod 400 "${SERVER_DIR}/server-key.pem"
    chmod 400 "${CLIENT_DIR}/client-key.pem"
    
    # Set readable permissions on certificates
    chmod 444 "${CA_DIR}/ca-cert.pem"
    chmod 444 "${SERVER_DIR}/server-cert.pem"
    chmod 444 "${CLIENT_DIR}/client-cert.pem"
    chmod 444 "${SERVER_DIR}/server-fullchain.pem"
    chmod 444 "${SERVER_DIR}/dhparam.pem"
}

# Main function
main() {
    log "Starting certificate generation for Bifrost Gateway..."
    
    # Check if OpenSSL is available
    if ! command -v openssl >/dev/null 2>&1; then
        log "Error: OpenSSL is not installed"
        exit 1
    fi
    
    # Generate certificates
    generate_ca_key
    generate_ca_cert
    generate_server_key
    generate_server_cert
    generate_client_key
    generate_client_cert
    generate_dh_params
    
    # Create additional files
    create_bundle
    create_k8s_secrets
    verify_certs
    set_permissions
    
    log "Certificate generation completed successfully!"
    log "Certificates are available in: ${CERT_DIR}"
    log ""
    log "Files generated:"
    log "  CA Certificate: ${CA_DIR}/ca-cert.pem"
    log "  Server Certificate: ${SERVER_DIR}/server-cert.pem"
    log "  Server Private Key: ${SERVER_DIR}/server-key.pem"
    log "  Server Full Chain: ${SERVER_DIR}/server-fullchain.pem"
    log "  Client Certificate: ${CLIENT_DIR}/client-cert.pem"
    log "  Client Private Key: ${CLIENT_DIR}/client-key.pem"
    log "  Client PKCS#12: ${CLIENT_DIR}/client.p12 (password: bifrost)"
    log "  DH Parameters: ${SERVER_DIR}/dhparam.pem"
    log ""
    log "Kubernetes secrets:"
    log "  Server TLS: ${CERT_DIR}/bifrost-tls-secret.yaml"
    log "  CA Certificate: ${CERT_DIR}/bifrost-ca-secret.yaml"
    log "  Client TLS: ${CERT_DIR}/bifrost-client-secret.yaml"
}

# Run main function
main "$@"