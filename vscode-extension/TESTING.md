# Testing Bifrost VS Code Extension

## Prerequisites

1. VS Code installed on your machine
2. Node.js 18+ installed
3. Go gateway running (optional, for full functionality)

## Building the Extension

```bash
# Navigate to extension directory
cd vscode-extension

# Install dependencies
npm install

# Compile TypeScript
npm run compile

# Create .vsix package
npx vsce package
```

This creates `bifrost-0.1.0.vsix` in the current directory.

## Installing for Local Testing

### Method 1: Install from VSIX (macOS & Windows)

1. Open VS Code
2. Open Command Palette (`Cmd+Shift+P` on macOS, `Ctrl+Shift+P` on Windows)
3. Type "Extensions: Install from VSIX..."
4. Navigate to `vscode-extension/bifrost-0.1.0.vsix`
5. Click "Install"
6. Reload VS Code when prompted

### Method 2: Command Line Installation

```bash
# macOS/Linux
code --install-extension bifrost-0.1.0.vsix

# Windows
code.exe --install-extension bifrost-0.1.0.vsix
```

### Method 3: Development Mode (F5 Debugging)

1. Open the `vscode-extension` folder in VS Code
2. Press `F5` to launch a new VS Code window with the extension loaded
3. This allows live debugging and hot reload during development

## Testing the Extension

### Basic Functionality Test

1. Open Command Palette (`Cmd+Shift+P` / `Ctrl+Shift+P`)
2. Look for Bifrost commands:
   - "Bifrost: Discover Industrial Devices"
   - "Bifrost: Connect to Device"
   - "Bifrost: Open Monitor"
   - "Bifrost: Export Data"

3. Check the Activity Bar for Bifrost views:
   - Industrial Devices
   - Data Points
   - Diagnostics

### Testing with Go Gateway

1. Start the Go gateway:
   ```bash
   cd go-gateway
   just run
   ```

2. The gateway runs on `http://localhost:8080`

3. Use extension commands to:
   - Discover devices on the network
   - Connect to discovered devices
   - Monitor real-time data
   - Read/write tag values

### Testing Without Gateway

The extension will show mock data when the gateway is unavailable, allowing UI testing.

## Platform-Specific Notes

### macOS

- Extension settings stored in: `~/Library/Application Support/Code/User/settings.json`
- Extension installed in: `~/.vscode/extensions/`
- Logs available in: Help > Toggle Developer Tools

### Windows

- Extension settings stored in: `%APPDATA%\Code\User\settings.json`
- Extension installed in: `%USERPROFILE%\.vscode\extensions\`
- Logs available in: Help > Toggle Developer Tools

## Uninstalling

### From VS Code

1. Open Extensions view (`Cmd+Shift+X` / `Ctrl+Shift+X`)
2. Find "Bifrost Industrial IoT"
3. Click "Uninstall"

### From Command Line

```bash
# List installed extensions
code --list-extensions

# Uninstall
code --uninstall-extension bifrost-team.bifrost
```

## Troubleshooting

### Extension Not Loading

1. Check VS Code version compatibility (requires 1.85.0+)
2. Verify Node.js version (requires 18+)
3. Check Developer Tools console for errors

### Connection Issues

1. Ensure Go gateway is running on port 8080
2. Check firewall settings
3. Verify gateway URL in extension settings

### Performance Issues

- The extension uses TypeScript-Go for faster compilation
- If experiencing slowness, check CPU/memory usage
- Consider disabling real-time monitoring for large datasets

## Development Tips

1. Use `npm run watch` for automatic recompilation
2. Press `Ctrl+R` / `Cmd+R` in the Extension Development Host to reload
3. Set breakpoints in TypeScript files for debugging
4. Use the Output panel to view extension logs