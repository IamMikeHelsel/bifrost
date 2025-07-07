# Bifrost Industrial IoT - VS Code Extension

Bridge your OT equipment to modern IT infrastructure directly from VS Code.

## Features

### 🔍 Device Discovery
- Automatic discovery of industrial devices on your network
- Support for Modbus TCP/RTU, OPC UA, Ethernet/IP, S7 protocols
- Manual device configuration for specific connections

### 📊 Real-time Monitoring
- Live data visualization with interactive charts
- Customizable update intervals
- Multiple device monitoring in separate panels
- Industrial-themed UI optimized for SCADA/HMI workflows

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
- Python 3.13+ with Bifrost installed
- Network access to industrial devices

## Installation

1. Install from VS Code Marketplace (coming soon)
2. Or install from VSIX:
   ```bash
   code --install-extension bifrost-0.1.0.vsix
   ```

## Getting Started

1. **Open the Bifrost view**: Click the Bifrost icon in the Activity Bar
2. **Discover devices**: Click the search icon or run "Bifrost: Discover Industrial Devices"
3. **Connect to a device**: Click the plug icon next to a discovered device
4. **Monitor data**: Click the graph icon to open real-time monitoring

## Commands

- `Bifrost: Discover Industrial Devices` - Scan network for compatible devices
- `Bifrost: Connect to Device` - Establish connection to selected device
- `Bifrost: Open Real-time Monitor` - Launch monitoring panel
- `Bifrost: Export Data` - Export device data to file

## Settings

- `bifrost.discoveryTimeout`: Device discovery timeout (default: 5000ms)
- `bifrost.defaultProtocol`: Default protocol for connections
- `bifrost.dataUpdateInterval`: How often to refresh data (default: 1000ms)
- `bifrost.maxDataPoints`: Maximum points to display in charts (default: 100)
- `bifrost.theme`: UI theme selection (auto/industrial-dark/industrial-light)

## Views

### Industrial Devices
Shows all discovered and configured devices organized by protocol. Provides quick access to connection management and monitoring.

### Live Data Points
Displays real-time values for all tags from connected devices. Supports inline reading and writing of values.

### Diagnostics
System health overview including:
- Connection status summary
- Performance metrics
- Error logs
- Python/Bifrost environment status

## Keyboard Shortcuts

- `Ctrl+Shift+B` `D`: Discover devices
- `Ctrl+Shift+B` `M`: Open monitor for selected device
- `Ctrl+Shift+B` `E`: Export data

## Troubleshooting

### No devices found
1. Ensure devices are powered on and connected to network
2. Check firewall settings for required ports
3. Verify Python and Bifrost are properly installed

### Connection failures
1. Verify device address and port settings
2. Check protocol-specific requirements
3. Review diagnostics panel for detailed errors

### Performance issues
1. Reduce data update interval in settings
2. Limit number of monitored tags
3. Close unused monitoring panels

## Contributing

See the main [Bifrost repository](https://github.com/yourusername/bifrost) for contribution guidelines.

## License

Same as Bifrost project - see LICENSE file.