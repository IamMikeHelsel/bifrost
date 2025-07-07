# Performance Monitoring and SLA Tracking Runbook

## Overview

This runbook provides procedures for monitoring the performance of Bifrost Gateway and tracking Service Level Agreements (SLAs) in production environments.

## Service Level Objectives (SLOs)

### Availability SLOs
- **Gateway Uptime**: 99.9% (43 minutes downtime per month)
- **API Availability**: 99.95% (21.6 minutes downtime per month)
- **Device Connectivity**: 99.5% (3.6 hours downtime per month)

### Performance SLOs
- **Response Time**: 95th percentile < 500ms
- **API Latency**: 95th percentile < 200ms
- **Throughput**: >10,000 requests per second peak capacity
- **Connection Establishment**: <2 seconds for Modbus devices

### Reliability SLOs
- **Error Rate**: <0.1% of all requests
- **Data Accuracy**: 99.99% (1 error per 10,000 data points)
- **Recovery Time**: <5 minutes for automatic failover

## Key Performance Indicators (KPIs)

### System Metrics
```promql
# Gateway uptime
up{job="bifrost-gateway"}

# Request rate
rate(bifrost_requests_total[5m])

# Error rate
rate(bifrost_errors_total[5m]) / rate(bifrost_requests_total[5m])

# Response time percentiles
histogram_quantile(0.95, rate(bifrost_request_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(bifrost_request_duration_seconds_bucket[5m]))
```

### Application Metrics
```promql
# Active connections
bifrost_connections_active

# Connection success rate
rate(bifrost_connections_established_total[5m]) / rate(bifrost_connections_attempted_total[5m])

# Data throughput
rate(bifrost_data_points_processed_total[5m])

# Memory usage
go_memstats_heap_inuse_bytes / go_memstats_heap_sys_bytes
```

### Business Metrics
```promql
# Device uptime
bifrost_device_connected

# Protocol-specific metrics
rate(bifrost_modbus_reads_total[5m])
rate(bifrost_modbus_writes_total[5m])

# Data quality metrics
rate(bifrost_data_validation_failures_total[5m])
```

## Monitoring Dashboards

### Real-time Operations Dashboard

#### Gateway Health Panel
```json
{
  "title": "Gateway Health Status",
  "type": "stat",
  "targets": [
    {
      "expr": "up{job=\"bifrost-gateway\"}",
      "legendFormat": "Gateway Status"
    }
  ],
  "fieldConfig": {
    "defaults": {
      "mappings": [
        {"options": {"0": {"text": "DOWN", "color": "red"}}, "type": "value"},
        {"options": {"1": {"text": "UP", "color": "green"}}, "type": "value"}
      ]
    }
  }
}
```

#### Performance Metrics Panel
```json
{
  "title": "Response Time Percentiles",
  "type": "graph",
  "targets": [
    {
      "expr": "histogram_quantile(0.50, rate(bifrost_request_duration_seconds_bucket[5m]))",
      "legendFormat": "50th percentile"
    },
    {
      "expr": "histogram_quantile(0.95, rate(bifrost_request_duration_seconds_bucket[5m]))",
      "legendFormat": "95th percentile"
    },
    {
      "expr": "histogram_quantile(0.99, rate(bifrost_request_duration_seconds_bucket[5m]))",
      "legendFormat": "99th percentile"
    }
  ]
}
```

### Capacity Planning Dashboard

#### Resource Utilization
```promql
# CPU utilization
rate(process_cpu_seconds_total[5m]) * 100

# Memory utilization
(go_memstats_heap_inuse_bytes / go_memstats_heap_sys_bytes) * 100

# Network I/O
rate(process_network_receive_bytes_total[5m])
rate(process_network_transmit_bytes_total[5m])

# Disk I/O
rate(process_disk_read_bytes_total[5m])
rate(process_disk_write_bytes_total[5m])
```

#### Connection Pooling
```promql
# Connection pool usage
bifrost_connection_pool_active / bifrost_connection_pool_max * 100

# Connection wait time
histogram_quantile(0.95, rate(bifrost_connection_wait_duration_seconds_bucket[5m]))

# Connection timeouts
rate(bifrost_connection_timeouts_total[5m])
```

### SLA Tracking Dashboard

#### Monthly SLA Tracking
```promql
# Monthly uptime calculation
(
  sum(up{job="bifrost-gateway"} * 60) / 
  (30 * 24 * 60)
) * 100

# Monthly error budget remaining
(
  1 - (
    sum(rate(bifrost_errors_total[30d])) / 
    sum(rate(bifrost_requests_total[30d]))
  )
) * 100
```

## Performance Monitoring Procedures

### Daily Performance Review

#### Morning Health Check (9:00 AM)
```bash
# Check overnight performance
curl -s "http://prometheus:9090/api/v1/query?query=up{job=\"bifrost-gateway\"}" | jq '.data.result[0].value[1]'

# Review error rates from last 24 hours
curl -s "http://prometheus:9090/api/v1/query?query=rate(bifrost_errors_total[24h])" | jq '.data.result[0].value[1]'

# Check response time trends
curl -s "http://prometheus:9090/api/v1/query?query=histogram_quantile(0.95,rate(bifrost_request_duration_seconds_bucket[24h]))" | jq '.data.result[0].value[1]'
```

#### Key Metrics Review
1. **Availability**: Check uptime percentage
2. **Performance**: Review response time trends
3. **Errors**: Investigate any error spikes
4. **Capacity**: Monitor resource utilization trends

#### Action Items
- Document any performance issues
- Create tickets for investigation
- Update capacity planning if needed
- Communicate findings to team

### Weekly Performance Analysis

#### Trend Analysis
```bash
# Generate weekly performance report
cat > weekly_performance_report.sh << 'EOF'
#!/bin/bash

WEEK_START=$(date -d "7 days ago" +%Y-%m-%dT%H:%M:%SZ)
WEEK_END=$(date +%Y-%m-%dT%H:%M:%SZ)

echo "=== Weekly Performance Report ==="
echo "Period: $WEEK_START to $WEEK_END"
echo

# Availability
UPTIME=$(curl -s "http://prometheus:9090/api/v1/query_range?query=up{job=\"bifrost-gateway\"}&start=$WEEK_START&end=$WEEK_END&step=3600" | jq -r '.data.result[0].values | map(.[1] | tonumber) | add / length * 100')
echo "Average Uptime: $UPTIME%"

# Response Time
RESPONSE_TIME=$(curl -s "http://prometheus:9090/api/v1/query_range?query=histogram_quantile(0.95,rate(bifrost_request_duration_seconds_bucket[1h]))&start=$WEEK_START&end=$WEEK_END&step=3600" | jq -r '.data.result[0].values | map(.[1] | tonumber) | add / length')
echo "Average 95th Percentile Response Time: ${RESPONSE_TIME}s"

# Error Rate
ERROR_RATE=$(curl -s "http://prometheus:9090/api/v1/query_range?query=rate(bifrost_errors_total[1h])/rate(bifrost_requests_total[1h])&start=$WEEK_START&end=$WEEK_END&step=3600" | jq -r '.data.result[0].values | map(.[1] | tonumber) | add / length * 100')
echo "Average Error Rate: $ERROR_RATE%"

echo
echo "=== SLA Status ==="
if (( $(echo "$UPTIME >= 99.9" | bc -l) )); then
    echo "✅ Availability SLA: MEETING ($UPTIME% >= 99.9%)"
else
    echo "❌ Availability SLA: FAILING ($UPTIME% < 99.9%)"
fi

if (( $(echo "$RESPONSE_TIME <= 0.5" | bc -l) )); then
    echo "✅ Response Time SLA: MEETING (${RESPONSE_TIME}s <= 0.5s)"
else
    echo "❌ Response Time SLA: FAILING (${RESPONSE_TIME}s > 0.5s)"
fi

if (( $(echo "$ERROR_RATE <= 0.1" | bc -l) )); then
    echo "✅ Error Rate SLA: MEETING ($ERROR_RATE% <= 0.1%)"
else
    echo "❌ Error Rate SLA: FAILING ($ERROR_RATE% > 0.1%)"
fi
EOF

chmod +x weekly_performance_report.sh
./weekly_performance_report.sh
```

#### Capacity Planning Review
```bash
# Check resource trends
echo "=== Resource Utilization Trends ==="

# CPU trends
CPU_TREND=$(curl -s "http://prometheus:9090/api/v1/query?query=rate(process_cpu_seconds_total[7d])*100" | jq -r '.data.result[0].value[1]')
echo "Average CPU Usage (7 days): $CPU_TREND%"

# Memory trends
MEM_TREND=$(curl -s "http://prometheus:9090/api/v1/query?query=go_memstats_heap_inuse_bytes/go_memstats_heap_sys_bytes*100" | jq -r '.data.result[0].value[1]')
echo "Current Memory Usage: $MEM_TREND%"

# Connection trends
CONN_TREND=$(curl -s "http://prometheus:9090/api/v1/query?query=bifrost_connections_active" | jq -r '.data.result[0].value[1]')
echo "Current Active Connections: $CONN_TREND"

# Predict capacity needs
if (( $(echo "$CPU_TREND > 70" | bc -l) )); then
    echo "⚠️  CPU usage trending high - consider scaling"
fi

if (( $(echo "$MEM_TREND > 80" | bc -l) )); then
    echo "⚠️  Memory usage high - monitor for memory leaks"
fi

if (( $(echo "$CONN_TREND > 800" | bc -l) )); then
    echo "⚠️  Connection count approaching limits - plan for scaling"
fi
```

### Monthly SLA Review

#### SLA Compliance Report
```bash
# Generate monthly SLA report
cat > monthly_sla_report.sh << 'EOF'
#!/bin/bash

MONTH_START=$(date -d "1 month ago" +%Y-%m-01T00:00:00Z)
MONTH_END=$(date +%Y-%m-01T00:00:00Z)
MONTH_NAME=$(date -d "1 month ago" +%B\ %Y)

echo "=== Monthly SLA Report for $MONTH_NAME ==="
echo

# Calculate total minutes in month
TOTAL_MINUTES=$(( $(date -d "$MONTH_END" +%s) - $(date -d "$MONTH_START" +%s) )) / 60

# Availability calculation
DOWNTIME_MINUTES=$(curl -s "http://prometheus:9090/api/v1/query_range?query=1-up{job=\"bifrost-gateway\"}&start=$MONTH_START&end=$MONTH_END&step=60" | jq -r '.data.result[0].values | map(.[1] | tonumber) | add')
UPTIME_PERCENT=$(echo "scale=4; (1 - $DOWNTIME_MINUTES / $TOTAL_MINUTES) * 100" | bc)

echo "Total Minutes in Month: $TOTAL_MINUTES"
echo "Downtime Minutes: $DOWNTIME_MINUTES"
echo "Uptime Percentage: $UPTIME_PERCENT%"
echo "SLA Target: 99.9% (Max 43.8 minutes downtime)"

if (( $(echo "$DOWNTIME_MINUTES <= 43.8" | bc -l) )); then
    echo "✅ Monthly Availability SLA: ACHIEVED"
    ERROR_BUDGET_REMAINING=$(echo "scale=2; (43.8 - $DOWNTIME_MINUTES) / 43.8 * 100" | bc)
    echo "Error Budget Remaining: $ERROR_BUDGET_REMAINING%"
else
    echo "❌ Monthly Availability SLA: MISSED"
    ERROR_BUDGET_OVERAGE=$(echo "scale=2; ($DOWNTIME_MINUTES - 43.8) / 43.8 * 100" | bc)
    echo "Error Budget Overage: $ERROR_BUDGET_OVERAGE%"
fi

echo
echo "=== Performance Metrics ==="

# Average response time
AVG_RESPONSE=$(curl -s "http://prometheus:9090/api/v1/query_range?query=histogram_quantile(0.95,rate(bifrost_request_duration_seconds_bucket[1h]))&start=$MONTH_START&end=$MONTH_END&step=3600" | jq -r '.data.result[0].values | map(.[1] | tonumber) | add / length')
echo "Average 95th Percentile Response Time: ${AVG_RESPONSE}s (Target: <0.5s)"

# Average error rate
AVG_ERROR_RATE=$(curl -s "http://prometheus:9090/api/v1/query_range?query=rate(bifrost_errors_total[1h])/rate(bifrost_requests_total[1h])&start=$MONTH_START&end=$MONTH_END&step=3600" | jq -r '.data.result[0].values | map(.[1] | tonumber) | add / length * 100')
echo "Average Error Rate: $AVG_ERROR_RATE% (Target: <0.1%)"

# Total requests processed
TOTAL_REQUESTS=$(curl -s "http://prometheus:9090/api/v1/query?query=increase(bifrost_requests_total[1M])" | jq -r '.data.result[0].value[1]')
echo "Total Requests Processed: $TOTAL_REQUESTS"

echo
echo "=== Action Items ==="
if (( $(echo "$DOWNTIME_MINUTES > 43.8" | bc -l) )); then
    echo "- Investigate causes of SLA miss"
    echo "- Implement additional redundancy"
    echo "- Review incident response procedures"
fi

if (( $(echo "$AVG_RESPONSE > 0.5" | bc -l) )); then
    echo "- Optimize application performance"
    echo "- Review resource allocation"
    echo "- Consider caching improvements"
fi

if (( $(echo "$AVG_ERROR_RATE > 0.1" | bc -l) )); then
    echo "- Investigate error patterns"
    echo "- Improve error handling"
    echo "- Enhance monitoring and alerting"
fi
EOF

chmod +x monthly_sla_report.sh
./monthly_sla_report.sh
```

## Performance Optimization

### Response Time Optimization

#### Database Query Optimization
```bash
# Monitor Redis response times
redis-cli --latency-history -i 1

# Check slow queries
redis-cli SLOWLOG GET 10

# Optimize Redis configuration
redis-cli CONFIG SET maxmemory-policy allkeys-lru
redis-cli CONFIG SET maxmemory 512mb
```

#### Connection Pool Tuning
```yaml
# Optimize connection pool settings
redis:
  pool_size: 20              # Increase pool size
  max_idle_timeout: 300s     # Adjust timeout
  dial_timeout: 5s           # Connection timeout
  read_timeout: 3s           # Read timeout
  write_timeout: 3s          # Write timeout
```

#### Caching Strategy
```go
// Implement caching for frequently accessed data
type CacheConfig struct {
    TTL           time.Duration
    MaxSize       int
    EvictionPolicy string
}

// Cache device metadata
func CacheDeviceMetadata(deviceID string, metadata *DeviceMetadata) {
    cache.Set(fmt.Sprintf("device:%s:metadata", deviceID), metadata, 5*time.Minute)
}
```

### Throughput Optimization

#### Horizontal Scaling
```bash
# Scale deployment based on load
kubectl scale deployment/bifrost-gateway -n bifrost-system --replicas=10

# Configure HPA for automatic scaling
kubectl apply -f k8s/hpa.yaml

# Monitor scaling events
kubectl get hpa -n bifrost-system -w
```

#### Load Balancing
```yaml
# Configure session affinity for stateful connections
apiVersion: v1
kind: Service
metadata:
  name: bifrost-gateway-service
spec:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 300
```

### Memory Optimization

#### Garbage Collection Tuning
```bash
# Set GC target percentage
export GOGC=50

# Monitor GC performance
curl http://localhost:2112/metrics | grep go_gc_duration_seconds
```

#### Memory Profiling
```bash
# Capture heap profile
curl -o heap.prof http://localhost:8080/debug/pprof/heap

# Analyze memory usage
go tool pprof heap.prof
(pprof) top10
(pprof) list main.main
(pprof) web
```

## Alerting and Escalation

### Performance Alert Rules

#### Response Time Alerts
```yaml
- alert: HighResponseTime
  expr: histogram_quantile(0.95, rate(bifrost_request_duration_seconds_bucket[5m])) > 0.5
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "High response time detected"
    description: "95th percentile response time is {{ $value }}s"
```

#### Throughput Alerts
```yaml
- alert: LowThroughput
  expr: rate(bifrost_requests_total[5m]) < 100
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Low request throughput"
    description: "Request rate is only {{ $value }} req/s"
```

#### Error Rate Alerts
```yaml
- alert: HighErrorRate
  expr: rate(bifrost_errors_total[5m]) / rate(bifrost_requests_total[5m]) > 0.01
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "High error rate detected"
    description: "Error rate is {{ $value | humanizePercentage }}"
```

### Escalation Procedures

#### Level 1: Automated Response
- Auto-scaling triggered
- Circuit breakers activated
- Load balancer adjustments

#### Level 2: Engineering Team
- Performance team notified
- Investigation begins
- Immediate optimizations applied

#### Level 3: Management Escalation
- SLA breach confirmed
- Business impact assessment
- Executive notification

## Performance Testing

### Load Testing Schedule
- **Daily**: Smoke tests during off-peak hours
- **Weekly**: Sustained load tests
- **Monthly**: Peak capacity tests
- **Quarterly**: Stress tests and chaos engineering

### Performance Test Scenarios

#### Sustained Load Test
```bash
# Run 4-hour sustained load test
wrk -t12 -c400 -d4h --latency http://gateway-url/api/devices
```

#### Peak Load Test
```bash
# Test peak capacity
wrk -t20 -c1000 -d10m --latency http://gateway-url/health
```

#### Connection Burst Test
```bash
# Test connection handling
for i in {1..1000}; do
  curl -s http://gateway-url/api/devices &
done
wait
```

## Documentation and Reporting

### Weekly Performance Reports
- Automated generation every Monday
- Distribution to engineering team
- Trend analysis and recommendations

### Monthly SLA Reports
- Business-focused metrics
- Executive summary
- Action plan for improvements

### Quarterly Performance Reviews
- Comprehensive analysis
- Capacity planning updates
- Performance roadmap adjustments

Last updated: [Date]
Next review: [Date + 3 months]