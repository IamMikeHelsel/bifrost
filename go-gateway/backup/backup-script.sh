#!/bin/bash

# Bifrost Gateway Backup Script
# This script creates backups of configuration, logs, and data

set -euo pipefail

# Configuration
BACKUP_DIR="/opt/bifrost/backups"
RETENTION_DAYS=30
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="bifrost-backup-${TIMESTAMP}"
BACKUP_PATH="${BACKUP_DIR}/${BACKUP_NAME}"

# Create backup directory
mkdir -p "${BACKUP_PATH}"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "${BACKUP_PATH}/backup.log"
}

# Function to check if running in Kubernetes
is_kubernetes() {
    [ -n "${KUBERNETES_SERVICE_HOST:-}" ]
}

# Function to backup configuration
backup_config() {
    log "Backing up configuration files..."
    
    if is_kubernetes; then
        # In Kubernetes, backup ConfigMaps and Secrets
        kubectl get configmap -n bifrost-system -o yaml > "${BACKUP_PATH}/configmaps.yaml"
        kubectl get secret -n bifrost-system -o yaml > "${BACKUP_PATH}/secrets.yaml"
        kubectl get deployment -n bifrost-system -o yaml > "${BACKUP_PATH}/deployments.yaml"
        kubectl get service -n bifrost-system -o yaml > "${BACKUP_PATH}/services.yaml"
        kubectl get ingress -n bifrost-system -o yaml > "${BACKUP_PATH}/ingress.yaml"
        kubectl get hpa -n bifrost-system -o yaml > "${BACKUP_PATH}/hpa.yaml"
        kubectl get pvc -n bifrost-system -o yaml > "${BACKUP_PATH}/pvc.yaml"
    else
        # In standalone mode, backup config files
        cp -r /etc/bifrost/ "${BACKUP_PATH}/config/" 2>/dev/null || true
        cp /opt/bifrost/gateway.yaml "${BACKUP_PATH}/" 2>/dev/null || true
    fi
    
    log "Configuration backup completed"
}

# Function to backup logs
backup_logs() {
    log "Backing up log files..."
    
    if is_kubernetes; then
        # In Kubernetes, get logs from pods
        kubectl logs -n bifrost-system -l app=bifrost-gateway --tail=10000 > "${BACKUP_PATH}/gateway-logs.txt"
        kubectl logs -n bifrost-system -l app=prometheus --tail=10000 > "${BACKUP_PATH}/prometheus-logs.txt"
        kubectl logs -n bifrost-system -l app=redis --tail=10000 > "${BACKUP_PATH}/redis-logs.txt"
    else
        # In standalone mode, backup log files
        cp -r /var/log/bifrost/ "${BACKUP_PATH}/logs/" 2>/dev/null || true
    fi
    
    log "Log backup completed"
}

# Function to backup Redis data
backup_redis() {
    log "Backing up Redis data..."
    
    if is_kubernetes; then
        # Create Redis backup using kubectl exec
        kubectl exec -n bifrost-system redis-0 -- redis-cli BGSAVE
        sleep 10  # Wait for background save to complete
        kubectl cp bifrost-system/redis-0:/data/dump.rdb "${BACKUP_PATH}/redis-dump.rdb"
    else
        # In standalone mode, backup Redis data
        redis-cli BGSAVE
        sleep 10
        cp /var/lib/redis/dump.rdb "${BACKUP_PATH}/redis-dump.rdb" 2>/dev/null || true
    fi
    
    log "Redis backup completed"
}

# Function to backup Prometheus data
backup_prometheus() {
    log "Backing up Prometheus data..."
    
    if is_kubernetes; then
        # Create Prometheus snapshot
        kubectl exec -n bifrost-system prometheus-0 -- curl -X POST http://localhost:9090/api/v1/admin/tsdb/snapshot
        # Note: In production, you'd want to retrieve the snapshot
        log "Prometheus snapshot created (manual retrieval required)"
    else
        # In standalone mode, backup Prometheus data directory
        cp -r /var/lib/prometheus/ "${BACKUP_PATH}/prometheus/" 2>/dev/null || true
    fi
    
    log "Prometheus backup completed"
}

# Function to backup certificates
backup_certificates() {
    log "Backing up certificates..."
    
    if is_kubernetes; then
        # Backup TLS secrets
        kubectl get secret -n bifrost-system -l type=kubernetes.io/tls -o yaml > "${BACKUP_PATH}/tls-secrets.yaml"
    else
        # In standalone mode, backup certificate files
        cp -r /etc/ssl/bifrost/ "${BACKUP_PATH}/certificates/" 2>/dev/null || true
    fi
    
    log "Certificate backup completed"
}

# Function to create backup archive
create_archive() {
    log "Creating backup archive..."
    
    cd "${BACKUP_DIR}"
    tar -czf "${BACKUP_NAME}.tar.gz" "${BACKUP_NAME}"
    
    # Calculate checksum
    sha256sum "${BACKUP_NAME}.tar.gz" > "${BACKUP_NAME}.tar.gz.sha256"
    
    # Remove uncompressed backup
    rm -rf "${BACKUP_NAME}"
    
    log "Backup archive created: ${BACKUP_NAME}.tar.gz"
}

# Function to clean old backups
cleanup_old_backups() {
    log "Cleaning up old backups..."
    
    find "${BACKUP_DIR}" -name "bifrost-backup-*.tar.gz" -mtime +${RETENTION_DAYS} -delete
    find "${BACKUP_DIR}" -name "bifrost-backup-*.tar.gz.sha256" -mtime +${RETENTION_DAYS} -delete
    
    log "Old backups cleaned up"
}

# Function to upload backup to cloud storage (optional)
upload_to_cloud() {
    if [ -n "${BACKUP_S3_BUCKET:-}" ]; then
        log "Uploading backup to S3..."
        aws s3 cp "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" "s3://${BACKUP_S3_BUCKET}/bifrost-backups/"
        aws s3 cp "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz.sha256" "s3://${BACKUP_S3_BUCKET}/bifrost-backups/"
        log "Backup uploaded to S3"
    fi
    
    if [ -n "${BACKUP_GCS_BUCKET:-}" ]; then
        log "Uploading backup to GCS..."
        gsutil cp "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" "gs://${BACKUP_GCS_BUCKET}/bifrost-backups/"
        gsutil cp "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz.sha256" "gs://${BACKUP_GCS_BUCKET}/bifrost-backups/"
        log "Backup uploaded to GCS"
    fi
}

# Function to send notification
send_notification() {
    local status=$1
    local message=$2
    
    if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"Bifrost Backup ${status}: ${message}\"}" \
            "${SLACK_WEBHOOK_URL}"
    fi
    
    if [ -n "${EMAIL_RECIPIENT:-}" ]; then
        echo "${message}" | mail -s "Bifrost Backup ${status}" "${EMAIL_RECIPIENT}"
    fi
}

# Main backup function
main() {
    log "Starting Bifrost backup process..."
    
    # Check if backup directory exists
    if [ ! -d "${BACKUP_DIR}" ]; then
        log "Creating backup directory: ${BACKUP_DIR}"
        mkdir -p "${BACKUP_DIR}"
    fi
    
    # Perform backup tasks
    backup_config
    backup_logs
    backup_redis
    backup_prometheus
    backup_certificates
    create_archive
    cleanup_old_backups
    upload_to_cloud
    
    log "Backup process completed successfully"
    send_notification "SUCCESS" "Backup ${BACKUP_NAME} completed successfully"
}

# Error handling
trap 'log "Backup failed with error"; send_notification "FAILED" "Backup failed with error"; exit 1' ERR

# Run main function
main "$@"