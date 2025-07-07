#!/bin/bash

# Bifrost Gateway Edge Installation Script
# This script installs and configures Bifrost Gateway on edge devices

set -euo pipefail

# Configuration
INSTALL_DIR="/opt/bifrost"
CONFIG_DIR="/etc/bifrost"
LOG_DIR="/var/log/bifrost"
DATA_DIR="/var/lib/bifrost"
USER="bifrost"
GROUP="bifrost"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

log_info() {
    log "${GREEN}INFO${NC}: $1"
}

log_warn() {
    log "${YELLOW}WARN${NC}: $1"
}

log_error() {
    log "${RED}ERROR${NC}: $1"
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

# Detect system architecture and OS
detect_system() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="armv7"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    log_info "Detected system: $OS/$ARCH"
}

# Check system requirements
check_requirements() {
    log_info "Checking system requirements..."
    
    # Check minimum memory (512MB)
    MEMORY_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    MEMORY_MB=$((MEMORY_KB / 1024))
    
    if [ $MEMORY_MB -lt 512 ]; then
        log_warn "Low memory detected: ${MEMORY_MB}MB (recommended: 1GB+)"
    else
        log_info "Memory: ${MEMORY_MB}MB"
    fi
    
    # Check disk space (1GB free)
    DISK_FREE=$(df / | awk 'NR==2 {print $4}')
    DISK_FREE_MB=$((DISK_FREE / 1024))
    
    if [ $DISK_FREE_MB -lt 1024 ]; then
        log_error "Insufficient disk space: ${DISK_FREE_MB}MB (required: 1GB+)"
        exit 1
    else
        log_info "Disk space: ${DISK_FREE_MB}MB available"
    fi
    
    # Check for systemd
    if ! command -v systemctl >/dev/null 2>&1; then
        log_error "systemd is required but not found"
        exit 1
    fi
    
    log_info "System requirements check passed"
}

# Create user and group
create_user() {
    log_info "Creating user and group..."
    
    if ! getent group $GROUP >/dev/null 2>&1; then
        groupadd --system $GROUP
        log_info "Created group: $GROUP"
    fi
    
    if ! getent passwd $USER >/dev/null 2>&1; then
        useradd --system --gid $GROUP --home-dir $DATA_DIR \
                --shell /sbin/nologin --comment "Bifrost Gateway" $USER
        log_info "Created user: $USER"
    fi
}

# Create directories
create_directories() {
    log_info "Creating directories..."
    
    mkdir -p $INSTALL_DIR/bin
    mkdir -p $CONFIG_DIR
    mkdir -p $LOG_DIR
    mkdir -p $DATA_DIR
    
    # Set ownership and permissions
    chown root:root $INSTALL_DIR
    chown root:root $CONFIG_DIR
    chown $USER:$GROUP $LOG_DIR
    chown $USER:$GROUP $DATA_DIR
    
    chmod 755 $INSTALL_DIR
    chmod 755 $CONFIG_DIR
    chmod 755 $LOG_DIR
    chmod 755 $DATA_DIR
    
    log_info "Directories created and configured"
}

# Download and install binary
install_binary() {
    log_info "Installing Bifrost Gateway binary..."
    
    # In production, this would download from a release URL
    # For now, copy from current directory
    if [ -f "./bin/bifrost-gateway" ]; then
        cp "./bin/bifrost-gateway" "$INSTALL_DIR/bin/"
    elif [ -f "./bifrost-gateway" ]; then
        cp "./bifrost-gateway" "$INSTALL_DIR/bin/"
    else
        log_error "Binary not found. Please ensure bifrost-gateway binary is available."
        exit 1
    fi
    
    # Set permissions
    chown root:root "$INSTALL_DIR/bin/bifrost-gateway"
    chmod 755 "$INSTALL_DIR/bin/bifrost-gateway"
    
    # Verify binary
    if "$INSTALL_DIR/bin/bifrost-gateway" -help >/dev/null 2>&1; then
        log_info "Binary installation verified"
    else
        log_error "Binary verification failed"
        exit 1
    fi
}

# Install configuration
install_config() {
    log_info "Installing configuration..."
    
    cat > "$CONFIG_DIR/gateway.yaml" <<EOF
gateway:
  port: 8080
  grpc_port: 9090
  metrics_port: 2112
  max_connections: 100
  data_buffer_size: 1000
  update_interval: 1s
  enable_metrics: true
  log_level: info
  health_check_interval: 30s

protocols:
  modbus:
    default_timeout: 5s
    default_unit_id: 1
    max_connections: 50
    connection_timeout: 10s
    read_timeout: 5s
    write_timeout: 5s
    enable_keep_alive: true
    connection_pool_size: 10

logging:
  level: info
  format: json
  file: $LOG_DIR/gateway.log
  max_size: 100MB
  max_age: 30
  max_backups: 10

performance:
  max_goroutines: 200
  read_buffer_size: 4096
  write_buffer_size: 4096
  tcp_keep_alive: true
  tcp_no_delay: true
EOF
    
    chown root:root "$CONFIG_DIR/gateway.yaml"
    chmod 644 "$CONFIG_DIR/gateway.yaml"
    
    log_info "Configuration installed"
}

# Install systemd service
install_systemd_service() {
    log_info "Installing systemd service..."
    
    # Copy service file (assumes it's in the same directory)
    if [ -f "./systemd/bifrost-gateway.service" ]; then
        cp "./systemd/bifrost-gateway.service" "/etc/systemd/system/"
    else
        log_error "systemd service file not found"
        exit 1
    fi
    
    # Reload systemd and enable service
    systemctl daemon-reload
    systemctl enable bifrost-gateway.service
    
    log_info "systemd service installed and enabled"
}

# Configure firewall
configure_firewall() {
    log_info "Configuring firewall..."
    
    if command -v ufw >/dev/null 2>&1; then
        # Ubuntu/Debian UFW
        ufw allow 8080/tcp comment "Bifrost Gateway HTTP"
        ufw allow 9090/tcp comment "Bifrost Gateway gRPC"
        ufw allow 2112/tcp comment "Bifrost Gateway Metrics"
        log_info "UFW firewall rules added"
    elif command -v firewall-cmd >/dev/null 2>&1; then
        # RHEL/CentOS firewalld
        firewall-cmd --permanent --add-port=8080/tcp
        firewall-cmd --permanent --add-port=9090/tcp
        firewall-cmd --permanent --add-port=2112/tcp
        firewall-cmd --reload
        log_info "firewalld rules added"
    else
        log_warn "No supported firewall found. Please configure manually:"
        log_warn "  - Allow TCP port 8080 (HTTP API)"
        log_warn "  - Allow TCP port 9090 (gRPC)"
        log_warn "  - Allow TCP port 2112 (Metrics)"
    fi
}

# Install monitoring agent (optional)
install_monitoring() {
    log_info "Setting up monitoring..."
    
    # Create monitoring configuration
    cat > "$CONFIG_DIR/node-exporter.yaml" <<EOF
# Node Exporter configuration for Bifrost Gateway edge device
# This file can be used with Prometheus Node Exporter
collectors:
  - cpu
  - meminfo
  - diskstats
  - filesystem
  - netdev
  - loadavg
  - time
EOF
    
    # Install lightweight monitoring script
    cat > "$INSTALL_DIR/bin/monitor.sh" <<'EOF'
#!/bin/bash
# Simple monitoring script for edge devices

LOG_FILE="/var/log/bifrost/monitor.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

# Check gateway health
if curl -s -f http://localhost:8080/health > /dev/null; then
    log "Gateway health check: OK"
else
    log "Gateway health check: FAILED"
    # Optionally restart service
    systemctl restart bifrost-gateway.service
fi

# Check disk space
DISK_USAGE=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
if [ "$DISK_USAGE" -gt 90 ]; then
    log "WARNING: Disk usage high: $DISK_USAGE%"
fi

# Check memory usage
MEMORY_USAGE=$(free | awk 'NR==2{printf "%.2f", $3*100/$2}')
if (( $(echo "$MEMORY_USAGE > 90" | bc -l) )); then
    log "WARNING: Memory usage high: $MEMORY_USAGE%"
fi
EOF
    
    chmod +x "$INSTALL_DIR/bin/monitor.sh"
    
    # Add monitoring cron job
    echo "*/5 * * * * $USER $INSTALL_DIR/bin/monitor.sh" > /etc/cron.d/bifrost-monitor
    
    log_info "Monitoring setup completed"
}

# Start service
start_service() {
    log_info "Starting Bifrost Gateway service..."
    
    systemctl start bifrost-gateway.service
    
    # Wait a moment and check status
    sleep 3
    
    if systemctl is-active --quiet bifrost-gateway.service; then
        log_info "Service started successfully"
        
        # Verify API is responding
        sleep 2
        if curl -s -f http://localhost:8080/health > /dev/null; then
            log_info "Gateway API is responding"
        else
            log_warn "Gateway API not responding yet (may take a moment)"
        fi
    else
        log_error "Service failed to start"
        systemctl status bifrost-gateway.service
        exit 1
    fi
}

# Show status
show_status() {
    echo
    log_info "Installation completed successfully!"
    echo
    echo "Service Status:"
    systemctl status bifrost-gateway.service --no-pager
    echo
    echo "Gateway Information:"
    echo "  HTTP API: http://localhost:8080"
    echo "  gRPC API: localhost:9090"
    echo "  Metrics:  http://localhost:2112/metrics"
    echo "  Config:   $CONFIG_DIR/gateway.yaml"
    echo "  Logs:     $LOG_DIR/gateway.log"
    echo
    echo "Useful commands:"
    echo "  sudo systemctl status bifrost-gateway"
    echo "  sudo systemctl restart bifrost-gateway"
    echo "  sudo journalctl -u bifrost-gateway -f"
    echo "  curl http://localhost:8080/health"
    echo
}

# Uninstall function
uninstall() {
    log_info "Uninstalling Bifrost Gateway..."
    
    # Stop and disable service
    systemctl stop bifrost-gateway.service || true
    systemctl disable bifrost-gateway.service || true
    
    # Remove service file
    rm -f /etc/systemd/system/bifrost-gateway.service
    systemctl daemon-reload
    
    # Remove installation directories
    rm -rf $INSTALL_DIR
    rm -rf $CONFIG_DIR
    
    # Remove user and group
    userdel $USER 2>/dev/null || true
    groupdel $GROUP 2>/dev/null || true
    
    # Remove cron job
    rm -f /etc/cron.d/bifrost-monitor
    
    log_info "Uninstallation completed"
}

# Main installation function
main() {
    case "${1:-install}" in
        install)
            check_root
            detect_system
            check_requirements
            create_user
            create_directories
            install_binary
            install_config
            install_systemd_service
            configure_firewall
            install_monitoring
            start_service
            show_status
            ;;
        uninstall)
            check_root
            uninstall
            ;;
        *)
            echo "Usage: $0 [install|uninstall]"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"