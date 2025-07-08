# Cloud Connector Implementation Guide

This document describes the cloud connector framework implemented in the Bifrost Industrial Gateway.

## Overview

The cloud connector framework provides reliable, scalable connectivity to various cloud platforms and time-series databases. It implements the requirements from Issue #14:

- ✅ **Smart buffering with disk persistence**
- ✅ **Automatic retry with exponential backoff**
- ✅ **Connection pooling and health monitoring**
- ✅ **End-to-end encryption support**
- ✅ **Certificate management**

## Architecture

### Core Components

1. **CloudConnector Interface** - Unified interface for all cloud platform integrations
2. **Manager** - Coordinates multiple connectors and handles data routing
3. **Buffer** - Smart buffering with disk persistence for reliability
4. **Retry Manager** - Exponential backoff and circuit breaker patterns
5. **Health Checker** - Monitors connector health and handles reconnection

### Supported Connectors

| Connector | Status | Description |
|-----------|--------|-------------|
| Generic MQTT | ✅ Implemented | Generic MQTT broker connectivity |
| AWS IoT Core | ✅ Implemented | AWS IoT Core with device shadows |
| InfluxDB | ✅ Implemented | Time-series database (v1.x and v2.x) |
| Azure IoT Hub | 🔄 Framework Ready | Framework in place for implementation |
| Google Cloud IoT | 🔄 Framework Ready | Framework in place for implementation |
| AMQP | 🔄 Framework Ready | Framework in place for implementation |
| TimescaleDB | 🔄 Framework Ready | Framework in place for implementation |
| Kafka | 🔄 Framework Ready | Framework in place for implementation |

## Configuration

### Gateway Configuration

```yaml
gateway:
  port: 8080
  
  cloud:
    default_connector: "mqtt-local"
    batch_size: 100
    batch_timeout: 5s
    health_check_interval: 30s
    
    buffer:
      max_size: 10000
      flush_interval: 10s
      persistent_path: "/tmp/bifrost-cloud-buffer"
    
    connectors:
      mqtt-local:
        name: "Local MQTT"
        type: "mqtt"
        enabled: true
        endpoint: "tcp://localhost:1883"
        timeout: 30s
        retry_count: 5
        retry_delay: 2s
        tls_enabled: false
        buffer_size: 1000
        flush_interval: 5s
        disk_persistent: true
        provider_config:
          broker: "tcp://localhost:1883"
          client_id: "bifrost-gateway"
          qos: 1
          topic_prefix: "bifrost/industrial"
```

### MQTT Connector Configuration

```yaml
mqtt-connector:
  name: "Production MQTT"
  type: "mqtt"
  enabled: true
  endpoint: "tls://mqtt.example.com:8883"
  tls_enabled: true
  cert_file: "/etc/ssl/certs/client.pem"
  key_file: "/etc/ssl/private/client.key"
  ca_file: "/etc/ssl/certs/ca.pem"
  provider_config:
    broker: "tls://mqtt.example.com:8883"
    client_id: "bifrost-gateway-001"
    username: "username"
    password: "password"
    qos: 1
    retain: false
    topic_prefix: "industrial"
    data_topic: "telemetry"
    event_topic: "events"
    alarm_topic: "alarms"
```

### AWS IoT Core Configuration

```yaml
aws-iot:
  name: "AWS IoT Core"
  type: "aws-iot"
  enabled: true
  endpoint: "your-endpoint.iot.us-east-1.amazonaws.com"
  tls_enabled: true
  cert_file: "/etc/ssl/certs/device-cert.pem"
  key_file: "/etc/ssl/private/device-key.pem"
  ca_file: "/etc/ssl/certs/amazon-root-ca-1.pem"
  provider_config:
    region: "us-east-1"
    client_id: "bifrost-gateway"
    thing_name: "bifrost-gateway-001"
    topic_prefix: "bifrost"
    shadow_update: true
    qos: 1
```

### InfluxDB Configuration

```yaml
influxdb:
  name: "InfluxDB Time Series"
  type: "influxdb"
  enabled: true
  endpoint: "http://localhost:8086"
  provider_config:
    url: "http://localhost:8086"
    database: "bifrost"
    username: "admin"
    password: "password"
    precision: "ns"
    retention_policy: "autogen"
    measurement: "industrial_data"
    batch_size: 1000
    version: "1.x"  # or "2.x"
    # For InfluxDB 2.x:
    # token: "your-token"
    # organization: "your-org"
    # bucket: "your-bucket"
```

## Data Routing

### Routing Rules

Configure automatic data routing based on conditions:

```yaml
routing_rules:
  - name: "temperature_data"
    condition: "tag_name=temperature"
    connectors: ["mqtt-local", "influxdb"]
    priority: 1
    
  - name: "alarm_data"
    condition: "quality=ALARM"
    connectors: ["mqtt-local", "aws-iot"]
    priority: 2
    
  - name: "critical_devices"
    condition: "device_id=PLC001"
    connectors: ["aws-iot", "influxdb"]
    priority: 3
```

## Features

### Smart Buffering

- **Disk Persistence**: Data is preserved across gateway restarts
- **Priority Queuing**: Critical data is processed first
- **Automatic Compression**: Reduces storage requirements
- **Overflow Handling**: Graceful degradation when buffer is full
- **Data Expiration**: Automatic cleanup of old data

### Retry and Resilience

- **Exponential Backoff**: Intelligent retry delays with jitter
- **Circuit Breaker**: Prevents cascade failures
- **Connection Pooling**: Efficient resource utilization
- **Health Monitoring**: Automatic reconnection on failure
- **Dead Letter Queue**: Handles permanently failed messages

### Security

- **TLS/SSL Support**: End-to-end encryption
- **Certificate Management**: X.509 certificate support
- **Mutual Authentication**: Client certificate verification
- **Secrets Management**: Secure credential handling

## API Usage

### Programmatic Usage

```go
// Create cloud manager
manager, err := cloud.NewManager(logger, config)
if err != nil {
    return err
}

// Register connectors
mqttConnector, _ := connectors.NewMQTTConnector(logger, mqttConfig)
manager.RegisterConnector("mqtt", mqttConnector)

// Send data
data := &cloud.CloudData{
    ID:        "sensor-001",
    DeviceID:  "PLC001",
    TagName:   "temperature",
    Value:     25.5,
    Quality:   "GOOD",
    Timestamp: time.Now(),
}

err = manager.SendData(ctx, data)
```

### REST API Endpoints

The gateway automatically exposes cloud connector status:

```bash
# Get connector health
GET /api/cloud/health

# Get connector metrics
GET /api/cloud/metrics

# Send data to specific connector
POST /api/cloud/connectors/{name}/data

# Get connector configuration
GET /api/cloud/connectors/{name}/config
```

## Monitoring and Metrics

### Prometheus Metrics

The cloud connectors expose comprehensive metrics:

- `bifrost_cloud_data_points_sent_total`
- `bifrost_cloud_data_points_failed_total`
- `bifrost_cloud_batch_operations_total`
- `bifrost_cloud_connection_attempts_total`
- `bifrost_cloud_buffer_size`
- `bifrost_cloud_response_time_seconds`

### Health Checks

Automatic health monitoring includes:

- Connection status
- Response time tracking
- Success rate calculation
- Error count monitoring
- Last communication timestamp

## Error Handling

### Error Types

- **Transient Errors**: Network timeouts, temporary unavailability
- **Authentication Errors**: Invalid certificates, expired tokens
- **Configuration Errors**: Invalid endpoints, malformed data
- **Capacity Errors**: Rate limiting, quota exceeded

### Error Recovery

- Automatic retry with exponential backoff
- Circuit breaker to prevent cascade failures
- Fallback to buffering for offline scenarios
- Health check triggered reconnection

## Testing

### Unit Tests

```bash
# Run cloud connector tests
go test ./internal/cloud/... -v

# Run with coverage
go test ./internal/cloud/... -cover
```

### Integration Testing

```bash
# Start test environment
docker-compose up -d mosquitto influxdb

# Run integration tests
go test ./internal/cloud/... -tags=integration
```

## Production Deployment

### Resource Requirements

- **Memory**: 50-100MB base + 1MB per 1000 buffered messages
- **CPU**: Low usage, spikes during batch processing
- **Disk**: Depends on buffer configuration (default ~100MB)
- **Network**: Minimal bandwidth, depends on data volume

### Performance Tuning

- Adjust batch sizes based on data volume
- Configure buffer sizes for your memory constraints
- Tune retry settings for your network conditions
- Set appropriate flush intervals for latency requirements

### Security Considerations

- Use TLS for all production deployments
- Rotate certificates regularly
- Monitor for authentication failures
- Implement proper firewall rules

## Troubleshooting

### Common Issues

1. **Connection Failures**
   - Check network connectivity
   - Verify certificates and credentials
   - Review firewall settings

2. **High Memory Usage**
   - Reduce buffer sizes
   - Check for network issues causing buffering
   - Monitor data volume and batch settings

3. **Data Loss**
   - Enable disk persistence
   - Check disk space availability
   - Review retry configurations

### Debug Logging

Enable debug logging for detailed troubleshooting:

```yaml
gateway:
  log_level: debug
```

### Health Endpoints

Monitor connector health:

```bash
# Check overall gateway health
curl http://localhost:8080/health

# Check cloud connector status
curl http://localhost:8080/api/cloud/health
```

## Future Enhancements

- Azure IoT Hub connector implementation
- Google Cloud IoT connector implementation
- Apache Kafka connector implementation
- TimescaleDB connector implementation
- AMQP connector implementation
- Advanced data transformation capabilities
- Machine learning-based anomaly detection
- Edge computing integration