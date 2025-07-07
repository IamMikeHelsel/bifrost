# Bifrost (Legacy Python Package)

> **⚠️ DEPRECATED**: This Python package has been superseded by the [Go Gateway](../../go-gateway/) implementation. 
> 
> **Current Bifrost**: High-performance Go gateway with proven 18,879 ops/sec throughput
> 
> **See**: [Go Gateway Documentation](../../go-gateway/README.md) for current installation and usage.

---

*The content below represents the legacy Python implementation, preserved for historical reference.*

🌉 **Bridge your industrial equipment to modern IT infrastructure**

Bifrost makes industrial equipment speak the language of modern software. Built for engineers stuck between the OT and IT worlds.

## Current Installation (Go Gateway)

```bash
# Production deployment - single binary
wget https://github.com/bifrost/gateway/releases/latest/download/bifrost-gateway-linux-amd64
chmod +x bifrost-gateway-linux-amd64
./bifrost-gateway-linux-amd64

# Development environment  
git clone https://github.com/bifrost/bifrost
cd bifrost/go-gateway
make build
```

## Usage

### Connect to Any Device

```python
import bifrost

# Simple connection
async with bifrost.connect("modbus://192.168.1.100") as plc:
    data = await plc.read_tags(["temperature", "pressure"])
    print(data)

# Discovery
devices = await bifrost.discover_devices("192.168.1.0/24")
for device in devices:
    print(f"Found {device.type} at {device.address}")
```

### Beautiful CLI

```bash
# Discover devices
bifrost discover --scan 192.168.1.0/24

# Interactive connection
bifrost connect

# Live monitoring  
bifrost monitor --dashboard
```

### Modern Python Patterns

```python
from bifrost import ModbusConnection, Tag, DataType

# Type-safe tag definitions
tags = [
    Tag("temp_1", address=40001, datatype=DataType.FLOAT32),
    Tag("pressure", address=40003, datatype=DataType.FLOAT32),
    Tag("running", address=1, datatype=DataType.BOOL),
]

# Async context managers
async with ModbusConnection("192.168.1.100") as plc:
    # Bulk operations
    values = await plc.read_tags(tags)
    
    # High-frequency polling
    async for snapshot in plc.poll(tags, interval_ms=100):
        process_data(snapshot)
```

## What's Included

- **Modbus Support**: TCP and RTU with high performance
- **Beautiful CLI**: Rich terminal interface with colors and progress
- **Type Safety**: Full type hints and runtime validation
- **Async First**: Built on asyncio for high concurrency
- **Connection Pooling**: Efficient resource management
- **Smart Imports**: Install only what you need

## Optional Extensions

### OPC UA

```bash
uv add bifrost[opcua]
```

```python
from bifrost import OPCUAClient

async with OPCUAClient("opc.tcp://localhost:4840") as client:
    values = await client.read_values(["ns=2;i=1", "ns=2;i=2"])
```

### Edge Analytics

```bash
uv add bifrost[analytics]
```

```python
from bifrost import Pipeline, AnomalyDetector

pipeline = Pipeline()
    .source(plc_connection)
    .detect_anomalies(AnomalyDetector.isolation_forest())
    .sink(alert_system)
```

### Cloud Integration

```bash
uv add bifrost[cloud]
```

```python
from bifrost import CloudBridge, AWSConnector

bridge = CloudBridge()
    .source(plc_data)
    .destination(AWSConnector("my-iot-topic"))
    .start()
```

## Why Bifrost?

- **🚀 Fast**: Rust-powered performance for critical operations
- **🎯 Focused**: Designed specifically for industrial use cases
- **🔧 Practical**: Solves real problems automation engineers face
- **🌐 Modern**: Uses latest Python features and patterns
- **📦 Modular**: Install only what you need
- **🎨 Beautiful**: CLI that's actually enjoyable to use

## Supported Protocols

| Protocol | Status | Package |
|----------|--------|---------|
| Modbus TCP/RTU | ✅ Included | `bifrost` |
| OPC UA | ✅ Optional | `bifrost[opcua]` |
| Ethernet/IP | 🚧 Coming | `bifrost[protocols]` |
| Siemens S7 | 🚧 Coming | `bifrost[protocols]` |
| DNP3 | 📋 Planned | `bifrost[protocols]` |

## Documentation

- [Getting Started](https://bifrost.readthedocs.io/getting-started/)
- [API Reference](https://bifrost.readthedocs.io/api/)
- [Examples](https://bifrost.readthedocs.io/examples/)
- [CLI Guide](https://bifrost.readthedocs.io/cli/)

## License

MIT License - see LICENSE file for details.
