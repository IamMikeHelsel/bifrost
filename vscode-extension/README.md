# Bifrost Industrial Gateway - VS Code Extension

TypeScript-Go powered development environment for industrial automation. Connects to the high-performance Go gateway for real-time device monitoring and control.

## Features

### 🚀 TypeScript-Go Integration
- **10x faster compilation** with Microsoft's experimental TypeScript-Go compiler
- Sub-second build times for rapid development iteration
- Enhanced IntelliSense for industrial protocols

### 🔍 Device Discovery & Management
- Connect to Bifrost Go Gateway (REST API + WebSocket)
- Automatic discovery of industrial devices via gateway
- Support for Modbus TCP/RTU with OPC UA and Ethernet/IP coming soon
- Real-time connection status and diagnostics

### 📊 Real-time Monitoring
- Live data visualization powered by WebSocket streaming
- Sub-second updates for critical industrial processes
- Industrial-themed UI optimized for control room environments
- Performance metrics showing actual 18,879 ops/sec throughput

### 🔧 Device Management
- Tree view of all discovered and configured devices
- Connection status indicators
- Performance statistics and diagnostics
- Error tracking and troubleshooting

### 📝 Data Operations
- Read tag values with real-time updates
- Write values to writable tags
- Export data to CSV or JSON formats
- Batch operations for multiple tags

### 🎨 Professional Industrial UI
- Dark and light themes optimized for industrial use
- Color-coded status indicators:
  - 🟢 Green: Connected and healthy
  - 🟡 Yellow: Warnings or degraded performance
  - 🔴 Red: Errors or disconnected
  - 🔵 Blue: Information and navigation

## Requirements

- VS Code 1.85.0 or higher
- **Bifrost Go Gateway** running locally or remotely
- TypeScript-Go compiler (included in extension)
- Network access to industrial devices

## Installation

### Option 1: VS Code Marketplace
```bash
code --install-extension bifrost.industrial-gateway
```

### Option 2: Development Installation
```bash
# Clone repository
git clone https://github.com/bifrost/bifrost
cd bifrost/vscode-extension

# Install with TypeScript-Go
npm install
npm run compile  # 10x faster than standard TypeScript

# Install in VS Code
code --install-extension .
```

## Quick Start

### 1. Start Bifrost Go Gateway
```bash
# Download and run gateway
wget https://github.com/bifrost/gateway/releases/latest/download/bifrost-gateway-linux-amd64
chmod +x bifrost-gateway-linux-amd64
./bifrost-gateway-linux-amd64
# Gateway runs on http://localhost:8080
```

### 2. Configure Extension
1. **Open the Bifrost view**: Click the Bifrost icon in the Activity Bar
2. **Set gateway URL**: Default is `http://localhost:8080`
3. **Test connection**: Extension will show gateway status

### 3. Discover and Monitor Devices
1. **Discover devices**: Use gateway's device discovery API
2. **Connect to devices**: Real-time WebSocket streaming
3. **Monitor data**: Live tag values with sub-second updates

## Commands

- `Bifrost: Connect to Gateway` - Connect to Bifrost Go Gateway
- `Bifrost: Discover Industrial Devices` - Trigger device discovery via gateway
- `Bifrost: Open Real-time Monitor` - Launch WebSocket monitoring panel
- `Bifrost: Export Data` - Export device data to CSV/JSON
- `Bifrost: View Performance Metrics` - Show gateway performance stats

## Settings

- `bifrost.gatewayUrl`: Gateway URL (default: `http://localhost:8080`)
- `bifrost.webSocketUrl`: WebSocket URL (default: `ws://localhost:8080/ws`)
- `bifrost.dataUpdateInterval`: WebSocket update frequency (default: 1000ms)
- `bifrost.maxDataPoints`: Maximum chart data points (default: 1000)
- `bifrost.theme`: UI theme (auto/industrial-dark/industrial-light)
- `bifrost.performanceMetrics`: Show gateway performance overlay (default: true)

## Views

### Gateway Status
Shows connection status to Bifrost Go Gateway with performance metrics:
- Throughput: Real-time operations per second
- Latency: Average response time (targeting <100µs)
- Connected devices and active tags

### Industrial Devices  
Tree view of devices discovered through gateway, organized by protocol. Real-time status indicators show connection health.

### Live Data Streaming
WebSocket-powered real-time tag value display with:
- Sub-second update rates
- Historical trend lines
- Alarm status indicators

### Performance Dashboard
Gateway performance monitoring:
- 18,879 ops/sec throughput display
- 53µs average latency metrics
- Concurrent connection count
- Memory usage and resource monitoring

## Keyboard Shortcuts

- `Ctrl+Shift+B` `G`: Connect to gateway
- `Ctrl+Shift+B` `D`: Discover devices via gateway  
- `Ctrl+Shift+B` `M`: Open real-time monitor
- `Ctrl+Shift+B` `P`: View performance metrics

## Troubleshooting

### Gateway connection failed
1. Ensure Bifrost Go Gateway is running on configured URL
2. Check if port 8080 is accessible
3. Verify gateway health at `http://localhost:8080/health`

### No devices found
1. Confirm gateway can reach industrial network
2. Check gateway logs for device discovery errors  
3. Verify device network configuration

### Performance issues
1. Check gateway performance metrics in extension
2. Verify WebSocket connection stability
3. Reduce update frequency if needed
4. Monitor gateway resource usage

## Contributing

See the main [Bifrost repository](https://github.com/yourusername/bifrost) for contribution guidelines.

## License

Same as Bifrost project - see LICENSE file.