# Production Deployment Runbook

## Overview

This runbook provides step-by-step procedures for deploying Bifrost Gateway in production environments, handling both Kubernetes and edge device deployments.

## Prerequisites

- [ ] Kubernetes cluster configured and accessible
- [ ] Docker registry access configured
- [ ] Monitoring infrastructure deployed
- [ ] Backup procedures tested
- [ ] Security scanning completed
- [ ] Performance testing passed

## Pre-Deployment Checklist

### Infrastructure Readiness
- [ ] Kubernetes cluster health verified
- [ ] Resource quotas configured
- [ ] Network policies tested
- [ ] Storage classes available
- [ ] Load balancer configured
- [ ] DNS records created

### Security Verification
- [ ] Container images scanned for vulnerabilities
- [ ] TLS certificates valid and deployed
- [ ] RBAC policies configured
- [ ] Network segmentation verified
- [ ] Secrets management tested

### Monitoring Setup
- [ ] Prometheus targets configured
- [ ] Grafana dashboards imported
- [ ] Alert rules deployed
- [ ] Notification channels tested
- [ ] Log aggregation configured

## Deployment Procedures

### 1. Kubernetes Deployment

#### Step 1: Prepare Environment
```bash
# Set environment variables
export NAMESPACE="bifrost-system"
export IMAGE_TAG="v1.0.0"
export CLUSTER_NAME="production"

# Verify cluster access
kubectl cluster-info
kubectl get nodes
```

#### Step 2: Create Namespace and Resources
```bash
# Apply base configuration
kubectl apply -f k8s/namespace.yaml

# Verify namespace creation
kubectl get namespace $NAMESPACE
```

#### Step 3: Deploy Dependencies
```bash
# Deploy Redis
kubectl apply -f k8s/redis.yaml

# Wait for Redis to be ready
kubectl wait --for=condition=ready pod -l app=redis -n $NAMESPACE --timeout=300s

# Deploy monitoring infrastructure
kubectl apply -f k8s/monitoring.yaml

# Wait for Prometheus to be ready
kubectl wait --for=condition=ready pod -l app=prometheus -n $NAMESPACE --timeout=300s
```

#### Step 4: Deploy Configuration
```bash
# Apply ConfigMaps and Secrets
kubectl apply -f k8s/configmap.yaml

# Verify configuration
kubectl get configmaps -n $NAMESPACE
kubectl get secrets -n $NAMESPACE
```

#### Step 5: Deploy Security Policies
```bash
# Apply security policies
kubectl apply -f k8s/security-policies.yaml

# Verify network policies
kubectl get networkpolicies -n $NAMESPACE
```

#### Step 6: Deploy Application
```bash
# Update image tag in deployment
sed -i "s|image: bifrost/gateway:.*|image: bifrost/gateway:${IMAGE_TAG}|" k8s/deployment.yaml

# Deploy the application
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/hpa.yaml

# Wait for deployment to complete
kubectl rollout status deployment/bifrost-gateway -n $NAMESPACE --timeout=600s
```

#### Step 7: Verify Deployment
```bash
# Check pod status
kubectl get pods -n $NAMESPACE -l app=bifrost-gateway

# Check service endpoints
kubectl get endpoints -n $NAMESPACE

# Test health endpoint
kubectl exec -n $NAMESPACE $(kubectl get pod -n $NAMESPACE -l app=bifrost-gateway -o jsonpath='{.items[0].metadata.name}') -- curl -f http://localhost:8080/health

# Check metrics
kubectl exec -n $NAMESPACE $(kubectl get pod -n $NAMESPACE -l app=bifrost-gateway -o jsonpath='{.items[0].metadata.name}') -- curl -f http://localhost:2112/metrics
```

### 2. Edge Device Deployment

#### Step 1: Prepare Edge Device
```bash
# SSH to edge device
ssh admin@edge-device-ip

# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker (if using container deployment)
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
```

#### Step 2: Deploy Using Install Script
```bash
# Copy installation files to edge device
scp -r deployment/edge/ admin@edge-device-ip:/tmp/

# Run installation script
ssh admin@edge-device-ip
cd /tmp/edge
sudo chmod +x install.sh
sudo ./install.sh install
```

#### Step 3: Verify Edge Deployment
```bash
# Check service status
sudo systemctl status bifrost-gateway

# Test health endpoint
curl -f http://localhost:8080/health

# Check logs
sudo journalctl -u bifrost-gateway -f
```

## Post-Deployment Verification

### Functional Testing
```bash
# Health checks
curl -f http://gateway-url/health
curl -f http://gateway-url/readiness

# API testing
curl -X GET http://gateway-url/api/devices
curl -X POST http://gateway-url/api/devices/discover \
  -H "Content-Type: application/json" \
  -d '{"network_range": "192.168.1.0/24"}'

# Metrics verification
curl http://gateway-url:2112/metrics | grep bifrost_
```

### Performance Testing
```bash
# Basic load test
wrk -t12 -c400 -d30s --latency http://gateway-url/health

# Connection test
# Test with actual Modbus devices or simulators
```

### Security Testing
```bash
# TLS verification
openssl s_client -connect gateway-url:443 -servername gateway-url

# Certificate validation
curl -v https://gateway-url/health

# Network policy testing
# Attempt connections from unauthorized sources
```

## Rollback Procedures

### Kubernetes Rollback
```bash
# View rollout history
kubectl rollout history deployment/bifrost-gateway -n $NAMESPACE

# Rollback to previous version
kubectl rollout undo deployment/bifrost-gateway -n $NAMESPACE

# Rollback to specific revision
kubectl rollout undo deployment/bifrost-gateway -n $NAMESPACE --to-revision=2

# Verify rollback
kubectl rollout status deployment/bifrost-gateway -n $NAMESPACE
```

### Edge Device Rollback
```bash
# Stop current service
sudo systemctl stop bifrost-gateway

# Restore previous binary
sudo cp /opt/bifrost/bin/bifrost-gateway.backup /opt/bifrost/bin/bifrost-gateway

# Restore previous configuration
sudo cp /etc/bifrost/gateway.yaml.backup /etc/bifrost/gateway.yaml

# Start service
sudo systemctl start bifrost-gateway

# Verify rollback
sudo systemctl status bifrost-gateway
curl -f http://localhost:8080/health
```

## Troubleshooting

### Common Issues

#### Pod Not Starting
```bash
# Check pod events
kubectl describe pod -n $NAMESPACE -l app=bifrost-gateway

# Check logs
kubectl logs -n $NAMESPACE -l app=bifrost-gateway --tail=100

# Check resource limits
kubectl top pods -n $NAMESPACE
```

#### Service Not Accessible
```bash
# Check service configuration
kubectl get svc -n $NAMESPACE bifrost-gateway-service

# Check endpoints
kubectl get endpoints -n $NAMESPACE bifrost-gateway-service

# Check ingress (if configured)
kubectl get ingress -n $NAMESPACE
```

#### High Memory Usage
```bash
# Check memory metrics
kubectl top pods -n $NAMESPACE

# Check application metrics
curl http://gateway-url:2112/metrics | grep go_memstats

# Analyze memory profile
curl -o heap.prof http://gateway-url:8080/debug/pprof/heap
go tool pprof heap.prof
```

#### Connection Issues
```bash
# Check network connectivity
kubectl exec -n $NAMESPACE deployment/bifrost-gateway -- netstat -tuln

# Test Modbus connectivity
kubectl exec -n $NAMESPACE deployment/bifrost-gateway -- telnet modbus-device-ip 502

# Check firewall rules
kubectl get networkpolicies -n $NAMESPACE
```

## Monitoring and Alerting

### Key Metrics to Monitor
- Gateway uptime and health status
- Active connections count
- Request rate and response time
- Error rate
- Memory and CPU usage
- Disk space utilization
- Network latency

### Alert Thresholds
- Gateway down: Immediate alert
- High error rate (>5%): Alert within 5 minutes
- High response time (>2s): Alert within 10 minutes
- High memory usage (>80%): Alert within 15 minutes
- Disk space low (<20%): Alert within 30 minutes

### Dashboard URLs
- Grafana: https://grafana.company.com/d/bifrost-overview
- Prometheus: https://prometheus.company.com
- Kibana: https://kibana.company.com

## Maintenance Procedures

### Regular Maintenance Tasks

#### Daily
- [ ] Check system health dashboards
- [ ] Review error logs
- [ ] Verify backup completion
- [ ] Monitor resource usage

#### Weekly
- [ ] Review performance metrics
- [ ] Check security alerts
- [ ] Update threat intelligence
- [ ] Test backup restoration

#### Monthly
- [ ] Security patches review
- [ ] Capacity planning review
- [ ] Update documentation
- [ ] Performance optimization review

### Planned Maintenance Windows
- Preferred window: Sunday 2:00 AM - 6:00 AM UTC
- Notification: 48 hours advance notice
- Rollback plan: Must be ready before maintenance
- Communication: Status page updates every 30 minutes

## Emergency Contacts

### On-Call Rotation
- Primary: ops-primary@company.com
- Secondary: ops-secondary@company.com
- Escalation: ops-manager@company.com

### External Contacts
- Cloud Provider Support: +1-800-XXX-XXXX
- Hardware Vendor: +1-800-YYY-YYYY
- Network Provider: +1-800-ZZZ-ZZZZ

## Documentation Updates

This runbook should be updated:
- After each deployment
- When procedures change
- After incident resolution
- During quarterly reviews

Last updated: [Date]
Next review: [Date + 3 months]