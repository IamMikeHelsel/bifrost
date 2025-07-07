#!/bin/bash

# Bifrost Gateway Restore Script
# This script restores backups of configuration, logs, and data

set -euo pipefail

# Configuration
BACKUP_DIR="/opt/bifrost/backups"
RESTORE_DIR="/opt/bifrost/restore"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check if running in Kubernetes
is_kubernetes() {
    [ -n "${KUBERNETES_SERVICE_HOST:-}" ]
}

# Function to display usage
usage() {
    echo "Usage: $0 <backup-file> [options]"
    echo "Options:"
    echo "  -c, --config-only    Restore configuration only"
    echo "  -d, --data-only      Restore data only"
    echo "  -f, --force          Force restore without confirmation"
    echo "  -h, --help           Show this help message"
    echo ""
    echo "Example:"
    echo "  $0 bifrost-backup-20240101_120000.tar.gz"
    echo "  $0 bifrost-backup-20240101_120000.tar.gz --config-only"
}

# Function to verify backup integrity
verify_backup() {
    local backup_file=$1
    local checksum_file="${backup_file}.sha256"
    
    log "Verifying backup integrity..."
    
    if [ ! -f "${checksum_file}" ]; then
        log "Warning: Checksum file not found, skipping verification"
        return 0
    fi
    
    if sha256sum -c "${checksum_file}"; then
        log "Backup integrity verified"
        return 0
    else
        log "Error: Backup integrity check failed"
        return 1
    fi
}

# Function to extract backup
extract_backup() {
    local backup_file=$1
    local extract_dir=$2
    
    log "Extracting backup..."
    
    mkdir -p "${extract_dir}"
    tar -xzf "${backup_file}" -C "${extract_dir}" --strip-components=1
    
    log "Backup extracted to ${extract_dir}"
}

# Function to restore configuration
restore_config() {
    local restore_dir=$1
    
    log "Restoring configuration..."
    
    if is_kubernetes; then
        # In Kubernetes, restore ConfigMaps and Secrets
        if [ -f "${restore_dir}/configmaps.yaml" ]; then
            kubectl apply -f "${restore_dir}/configmaps.yaml"
        fi
        
        if [ -f "${restore_dir}/secrets.yaml" ]; then
            kubectl apply -f "${restore_dir}/secrets.yaml"
        fi
        
        if [ -f "${restore_dir}/deployments.yaml" ]; then
            kubectl apply -f "${restore_dir}/deployments.yaml"
        fi
        
        if [ -f "${restore_dir}/services.yaml" ]; then
            kubectl apply -f "${restore_dir}/services.yaml"
        fi
        
        if [ -f "${restore_dir}/ingress.yaml" ]; then
            kubectl apply -f "${restore_dir}/ingress.yaml"
        fi
        
        if [ -f "${restore_dir}/hpa.yaml" ]; then
            kubectl apply -f "${restore_dir}/hpa.yaml"
        fi
        
        if [ -f "${restore_dir}/pvc.yaml" ]; then
            kubectl apply -f "${restore_dir}/pvc.yaml"
        fi
    else
        # In standalone mode, restore config files
        if [ -d "${restore_dir}/config" ]; then
            cp -r "${restore_dir}/config/"* /etc/bifrost/
        fi
        
        if [ -f "${restore_dir}/gateway.yaml" ]; then
            cp "${restore_dir}/gateway.yaml" /opt/bifrost/
        fi
    fi
    
    log "Configuration restored"
}

# Function to restore Redis data
restore_redis() {
    local restore_dir=$1
    
    log "Restoring Redis data..."
    
    if [ ! -f "${restore_dir}/redis-dump.rdb" ]; then
        log "Redis dump file not found, skipping Redis restore"
        return 0
    fi
    
    if is_kubernetes; then
        # In Kubernetes, restore Redis data
        kubectl exec -n bifrost-system redis-0 -- redis-cli FLUSHALL
        kubectl cp "${restore_dir}/redis-dump.rdb" bifrost-system/redis-0:/data/dump.rdb
        kubectl exec -n bifrost-system redis-0 -- redis-cli DEBUG RELOAD
    else
        # In standalone mode, restore Redis data
        systemctl stop redis
        cp "${restore_dir}/redis-dump.rdb" /var/lib/redis/dump.rdb
        chown redis:redis /var/lib/redis/dump.rdb
        systemctl start redis
    fi
    
    log "Redis data restored"
}

# Function to restore Prometheus data
restore_prometheus() {
    local restore_dir=$1
    
    log "Restoring Prometheus data..."
    
    if [ ! -d "${restore_dir}/prometheus" ]; then
        log "Prometheus data directory not found, skipping Prometheus restore"
        return 0
    fi
    
    if is_kubernetes; then
        log "Prometheus data restore in Kubernetes requires manual intervention"
        log "Please manually restore the Prometheus data from the snapshot"
    else
        # In standalone mode, restore Prometheus data
        systemctl stop prometheus
        rm -rf /var/lib/prometheus/*
        cp -r "${restore_dir}/prometheus/"* /var/lib/prometheus/
        chown -R prometheus:prometheus /var/lib/prometheus/
        systemctl start prometheus
    fi
    
    log "Prometheus data restore completed"
}

# Function to restore certificates
restore_certificates() {
    local restore_dir=$1
    
    log "Restoring certificates..."
    
    if is_kubernetes; then
        # In Kubernetes, restore TLS secrets
        if [ -f "${restore_dir}/tls-secrets.yaml" ]; then
            kubectl apply -f "${restore_dir}/tls-secrets.yaml"
        fi
    else
        # In standalone mode, restore certificate files
        if [ -d "${restore_dir}/certificates" ]; then
            cp -r "${restore_dir}/certificates/"* /etc/ssl/bifrost/
        fi
    fi
    
    log "Certificates restored"
}

# Function to restart services
restart_services() {
    log "Restarting services..."
    
    if is_kubernetes; then
        # In Kubernetes, restart deployments
        kubectl rollout restart deployment/bifrost-gateway -n bifrost-system
        kubectl rollout restart deployment/prometheus -n bifrost-system
        kubectl rollout restart statefulset/redis -n bifrost-system
        
        # Wait for rollout to complete
        kubectl rollout status deployment/bifrost-gateway -n bifrost-system
        kubectl rollout status deployment/prometheus -n bifrost-system
        kubectl rollout status statefulset/redis -n bifrost-system
    else
        # In standalone mode, restart services
        systemctl restart bifrost-gateway
        systemctl restart prometheus
        systemctl restart redis
        systemctl restart grafana
    fi
    
    log "Services restarted"
}

# Function to validate restore
validate_restore() {
    log "Validating restore..."
    
    if is_kubernetes; then
        # Check if pods are running
        kubectl get pods -n bifrost-system
        
        # Check if services are accessible
        kubectl exec -n bifrost-system $(kubectl get pod -n bifrost-system -l app=bifrost-gateway -o jsonpath='{.items[0].metadata.name}') -- curl -f http://localhost:8080/health
    else
        # Check if services are running
        systemctl status bifrost-gateway
        systemctl status prometheus
        systemctl status redis
        
        # Check if gateway is accessible
        curl -f http://localhost:8080/health
    fi
    
    log "Restore validation completed"
}

# Function to send notification
send_notification() {
    local status=$1
    local message=$2
    
    if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"Bifrost Restore ${status}: ${message}\"}" \
            "${SLACK_WEBHOOK_URL}"
    fi
    
    if [ -n "${EMAIL_RECIPIENT:-}" ]; then
        echo "${message}" | mail -s "Bifrost Restore ${status}" "${EMAIL_RECIPIENT}"
    fi
}

# Main restore function
main() {
    local backup_file=""
    local config_only=false
    local data_only=false
    local force=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -c|--config-only)
                config_only=true
                shift
                ;;
            -d|--data-only)
                data_only=true
                shift
                ;;
            -f|--force)
                force=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                if [ -z "${backup_file}" ]; then
                    backup_file="$1"
                else
                    log "Error: Unknown option $1"
                    usage
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    # Check if backup file is provided
    if [ -z "${backup_file}" ]; then
        log "Error: Backup file not specified"
        usage
        exit 1
    fi
    
    # Check if backup file exists
    if [ ! -f "${backup_file}" ]; then
        log "Error: Backup file not found: ${backup_file}"
        exit 1
    fi
    
    # Get full path to backup file
    backup_file=$(realpath "${backup_file}")
    
    # Confirmation prompt
    if [ "${force}" = false ]; then
        echo "This will restore the Bifrost Gateway from backup: ${backup_file}"
        echo "This operation will overwrite existing configuration and data."
        read -p "Are you sure you want to continue? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log "Restore cancelled"
            exit 0
        fi
    fi
    
    log "Starting Bifrost restore process..."
    
    # Verify backup integrity
    if ! verify_backup "${backup_file}"; then
        log "Error: Backup verification failed"
        exit 1
    fi
    
    # Create restore directory
    restore_dir="${RESTORE_DIR}/$(basename "${backup_file}" .tar.gz)"
    mkdir -p "${restore_dir}"
    
    # Extract backup
    extract_backup "${backup_file}" "${restore_dir}"
    
    # Perform restore based on options
    if [ "${config_only}" = true ]; then
        restore_config "${restore_dir}"
    elif [ "${data_only}" = true ]; then
        restore_redis "${restore_dir}"
        restore_prometheus "${restore_dir}"
    else
        # Full restore
        restore_config "${restore_dir}"
        restore_redis "${restore_dir}"
        restore_prometheus "${restore_dir}"
        restore_certificates "${restore_dir}"
    fi
    
    # Restart services
    restart_services
    
    # Validate restore
    validate_restore
    
    # Cleanup
    rm -rf "${restore_dir}"
    
    log "Restore process completed successfully"
    send_notification "SUCCESS" "Restore from ${backup_file} completed successfully"
}

# Error handling
trap 'log "Restore failed with error"; send_notification "FAILED" "Restore failed with error"; exit 1' ERR

# Run main function
main "$@"