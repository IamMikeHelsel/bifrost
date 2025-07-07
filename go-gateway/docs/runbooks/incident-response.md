# Incident Response Runbook

## Overview

This runbook provides procedures for responding to incidents affecting the Bifrost Gateway in production environments. It covers incident classification, response procedures, and post-incident activities.

## Incident Classification

### Severity Levels

#### Severity 1 (Critical)
- Complete gateway outage
- Security breach
- Data loss or corruption
- Manufacturing process stopped due to gateway failure

**Response Time:** Immediate (within 15 minutes)
**Escalation:** Immediate to on-call manager

#### Severity 2 (High)
- Partial gateway outage (>50% capacity loss)
- Performance degradation (>50% slower)
- Multiple device disconnections
- Memory or disk space critical

**Response Time:** Within 1 hour
**Escalation:** Within 30 minutes if not resolved

#### Severity 3 (Medium)
- Single device disconnections
- Minor performance issues
- Non-critical feature failures
- Warning thresholds exceeded

**Response Time:** Within 4 hours
**Escalation:** Within 2 hours if not resolved

#### Severity 4 (Low)
- Documentation issues
- Minor UI problems
- Informational alerts
- Planned maintenance overruns

**Response Time:** Within 24 hours
**Escalation:** During business hours only

## Incident Response Procedures

### Initial Response (First 15 minutes)

#### 1. Incident Identification
```bash
# Check system status
kubectl get pods -n bifrost-system
kubectl get services -n bifrost-system

# Check recent deployments
kubectl rollout history deployment/bifrost-gateway -n bifrost-system

# Quick health check
curl -f http://gateway-url/health || echo "Health check failed"
```

#### 2. Impact Assessment
- Determine scope of outage
- Identify affected systems/users
- Estimate business impact
- Check if incident is ongoing

#### 3. Communication
- Create incident ticket in tracking system
- Notify stakeholders via status page
- Join incident response channel (#incident-response)
- Assign incident commander if Severity 1 or 2

### Detailed Investigation

#### System Health Assessment
```bash
# Check pod status and events
kubectl describe pods -n bifrost-system -l app=bifrost-gateway

# Check recent logs
kubectl logs -n bifrost-system -l app=bifrost-gateway --tail=500

# Check resource usage
kubectl top pods -n bifrost-system
kubectl top nodes

# Check persistent volumes
kubectl get pv
kubectl get pvc -n bifrost-system
```

#### Network Connectivity
```bash
# Test internal connectivity
kubectl exec -n bifrost-system deployment/bifrost-gateway -- curl -f http://redis-service:6379

# Test external connectivity
kubectl exec -n bifrost-system deployment/bifrost-gateway -- nslookup external-service.com

# Check network policies
kubectl get networkpolicies -n bifrost-system
```

#### Database/Redis Health
```bash
# Check Redis connectivity
kubectl exec -n bifrost-system redis-0 -- redis-cli ping

# Check Redis memory usage
kubectl exec -n bifrost-system redis-0 -- redis-cli info memory

# Check Redis configuration
kubectl exec -n bifrost-system redis-0 -- redis-cli config get maxmemory
```

#### Application-Specific Checks
```bash
# Check application metrics
curl http://gateway-url:2112/metrics | grep bifrost_connections_active
curl http://gateway-url:2112/metrics | grep bifrost_errors_total

# Check application configuration
kubectl get configmap -n bifrost-system bifrost-gateway-config -o yaml

# Test API endpoints
curl -v http://gateway-url/api/devices
curl -v http://gateway-url/api/health
```

## Common Incident Scenarios

### Scenario 1: Gateway Pods Crashing

#### Symptoms
- Pods in CrashLoopBackOff state
- High restart count
- Service unavailable

#### Investigation Steps
```bash
# Check pod status
kubectl get pods -n bifrost-system -l app=bifrost-gateway

# Check pod events
kubectl describe pod -n bifrost-system <pod-name>

# Check logs for crash reason
kubectl logs -n bifrost-system <pod-name> --previous

# Check resource limits
kubectl describe deployment -n bifrost-system bifrost-gateway
```

#### Possible Causes and Solutions

**Out of Memory (OOMKilled)**
```bash
# Check memory usage trends in Grafana
# Increase memory limits temporarily
kubectl patch deployment -n bifrost-system bifrost-gateway -p '{"spec":{"template":{"spec":{"containers":[{"name":"bifrost-gateway","resources":{"limits":{"memory":"1Gi"}}}]}}}}'

# Monitor memory usage
kubectl top pods -n bifrost-system -l app=bifrost-gateway
```

**Configuration Error**
```bash
# Validate configuration
kubectl get configmap -n bifrost-system bifrost-gateway-config -o yaml

# Fix configuration and restart
kubectl rollout restart deployment/bifrost-gateway -n bifrost-system
```

**Failed Health Checks**
```bash
# Check health endpoint directly
kubectl exec -n bifrost-system <pod-name> -- curl -f http://localhost:8080/health

# Adjust health check parameters if needed
kubectl patch deployment -n bifrost-system bifrost-gateway --type='merge' -p='{"spec":{"template":{"spec":{"containers":[{"name":"bifrost-gateway","livenessProbe":{"initialDelaySeconds":60}}]}}}}'
```

### Scenario 2: High Error Rate

#### Symptoms
- Error rate >5% in metrics
- Alerts firing for error threshold
- User reports of failed operations

#### Investigation Steps
```bash
# Check error metrics
curl http://gateway-url:2112/metrics | grep bifrost_errors_total

# Check application logs for error patterns
kubectl logs -n bifrost-system -l app=bifrost-gateway | grep -i error | tail -50

# Check upstream dependencies
curl -f http://redis-service:6379 || echo "Redis connection failed"
```

#### Common Error Patterns

**Modbus Connection Errors**
```bash
# Check Modbus device connectivity
kubectl exec -n bifrost-system deployment/bifrost-gateway -- telnet modbus-device-ip 502

# Check firewall rules
# Review network policies
kubectl get networkpolicies -n bifrost-system

# Check device configuration
kubectl logs -n bifrost-system -l app=bifrost-gateway | grep -i modbus
```

**Database Connection Errors**
```bash
# Check Redis connectivity
kubectl exec -n bifrost-system redis-0 -- redis-cli ping

# Check connection pool settings
kubectl get configmap -n bifrost-system bifrost-gateway-config -o jsonpath='{.data.gateway\.yaml}' | grep -A5 redis

# Restart Redis if needed
kubectl rollout restart statefulset/redis -n bifrost-system
```

### Scenario 3: Performance Degradation

#### Symptoms
- Response time >2 seconds
- High CPU/memory usage
- Timeouts in client applications

#### Investigation Steps
```bash
# Check resource usage
kubectl top pods -n bifrost-system
kubectl top nodes

# Check performance metrics
curl http://gateway-url:2112/metrics | grep bifrost_request_duration

# Profile application performance
curl -o cpu.prof http://gateway-url:8080/debug/pprof/profile?seconds=30
curl -o heap.prof http://gateway-url:8080/debug/pprof/heap
```

#### Performance Optimization

**High CPU Usage**
```bash
# Analyze CPU profile
go tool pprof cpu.prof

# Scale horizontally if needed
kubectl scale deployment/bifrost-gateway -n bifrost-system --replicas=6

# Increase CPU limits
kubectl patch deployment -n bifrost-system bifrost-gateway -p '{"spec":{"template":{"spec":{"containers":[{"name":"bifrost-gateway","resources":{"limits":{"cpu":"1000m"}}}]}}}}'
```

**High Memory Usage**
```bash
# Analyze heap profile
go tool pprof heap.prof

# Check for memory leaks in logs
kubectl logs -n bifrost-system -l app=bifrost-gateway | grep -i "memory\|heap\|gc"

# Restart pods to clear memory
kubectl rollout restart deployment/bifrost-gateway -n bifrost-system
```

### Scenario 4: Security Incident

#### Symptoms
- Unauthorized access detected
- Unusual network traffic patterns
- Security alerts from monitoring tools

#### Immediate Actions
```bash
# Isolate affected pods
kubectl label pod <pod-name> -n bifrost-system quarantine=true
kubectl patch networkpolicy -n bifrost-system bifrost-network-policy -p '{"spec":{"podSelector":{"matchLabels":{"quarantine":"true"}},"policyTypes":["Ingress","Egress"],"ingress":[],"egress":[]}}'

# Capture forensic data
kubectl logs -n bifrost-system <pod-name> > incident-logs-$(date +%Y%m%d-%H%M%S).txt
kubectl exec -n bifrost-system <pod-name> -- ps aux > incident-processes-$(date +%Y%m%d-%H%M%S).txt

# Check for indicators of compromise
kubectl exec -n bifrost-system <pod-name> -- find /tmp -type f -mtime -1
kubectl exec -n bifrost-system <pod-name> -- netstat -tuln
```

#### Investigation
- Review authentication logs
- Check for privilege escalation
- Analyze network traffic patterns
- Coordinate with security team

## Recovery Procedures

### Service Recovery
```bash
# Restart deployment
kubectl rollout restart deployment/bifrost-gateway -n bifrost-system

# Wait for deployment to complete
kubectl rollout status deployment/bifrost-gateway -n bifrost-system --timeout=300s

# Verify recovery
curl -f http://gateway-url/health
curl -f http://gateway-url:2112/metrics
```

### Data Recovery
```bash
# Check backup availability
kubectl get cronjob -n bifrost-system bifrost-backup

# Restore from backup if needed
kubectl create job --from=cronjob/bifrost-backup manual-restore-$(date +%Y%m%d-%H%M%S) -n bifrost-system

# Verify data integrity
# Run data validation checks
```

### Rollback Procedures
```bash
# View rollout history
kubectl rollout history deployment/bifrost-gateway -n bifrost-system

# Rollback to previous version
kubectl rollout undo deployment/bifrost-gateway -n bifrost-system

# Verify rollback
kubectl rollout status deployment/bifrost-gateway -n bifrost-system
```

## Communication Templates

### Initial Alert Message
```
🚨 **INCIDENT ALERT** 🚨
Severity: [LEVEL]
Service: Bifrost Gateway
Issue: [Brief description]
Impact: [User/business impact]
ETA: [Estimated resolution time]
Updates: Status page and #incident-response
Incident ID: INC-[NUMBER]
```

### Status Update Template
```
📢 **INCIDENT UPDATE** 📢
Incident ID: INC-[NUMBER]
Status: [Investigating/Identified/Fixing/Resolved]
Update: [What we've found/done]
Next Steps: [What we're doing next]
ETA: [Updated estimate]
```

### Resolution Message
```
✅ **INCIDENT RESOLVED** ✅
Incident ID: INC-[NUMBER]
Duration: [Total time]
Root Cause: [Brief summary]
Resolution: [What fixed it]
Prevention: [Steps taken to prevent recurrence]
Post-mortem: [Link to detailed analysis]
```

## Post-Incident Activities

### Immediate Actions (Within 24 hours)
- [ ] Verify full service restoration
- [ ] Document timeline of events
- [ ] Collect feedback from stakeholders
- [ ] Schedule post-mortem meeting

### Post-Mortem Process
1. **Timeline Creation**: Document exact sequence of events
2. **Root Cause Analysis**: Use 5-whys or fishbone diagram
3. **Action Items**: Identify specific improvements
4. **Follow-up**: Track action item completion

### Learning and Improvement
- Update runbooks based on lessons learned
- Improve monitoring and alerting
- Enhance automation
- Conduct training if needed

## Tools and Resources

### Monitoring Dashboards
- Grafana: https://grafana.company.com/d/bifrost-overview
- Prometheus: https://prometheus.company.com
- Kibana: https://kibana.company.com

### Incident Management
- Incident tracking: https://company.atlassian.net
- Status page: https://status.company.com
- War room: #incident-response

### Documentation
- Architecture diagrams: /docs/architecture/
- Network diagrams: /docs/network/
- Deployment guides: /docs/deployment/

### Emergency Contacts
- On-call engineer: ops-oncall@company.com
- Incident commander: ops-manager@company.com
- Security team: security@company.com
- Executive escalation: cto@company.com

## Checklist for Major Incidents

### During Incident
- [ ] Incident commander assigned
- [ ] War room established
- [ ] Stakeholders notified
- [ ] Status page updated
- [ ] Timeline started
- [ ] Subject matter experts engaged

### Post-Resolution
- [ ] Service fully restored
- [ ] Stakeholders notified of resolution
- [ ] Status page updated
- [ ] Timeline completed
- [ ] Post-mortem scheduled
- [ ] Immediate actions documented

### Follow-up
- [ ] Post-mortem conducted
- [ ] Action items assigned
- [ ] Runbooks updated
- [ ] Preventive measures implemented
- [ ] Team debriefed

Last updated: [Date]
Next review: [Date + 3 months]