import * as vscode from 'vscode';
import { BifrostExtension } from '../extension';
import { DeviceManager } from '../deviceManager';
import { ConfigurationManager } from '../configurationManager';

// Mock external dependencies
jest.mock('../deviceManager');
jest.mock('../configurationManager');

describe('Bifrost Commands', () => {
  let extension: BifrostExtension;
  let mockDeviceManager: jest.Mocked<DeviceManager>;
  let mockConfigManager: jest.Mocked<ConfigurationManager>;
  let mockContext: vscode.ExtensionContext;

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Create mock context
    mockContext = {
      subscriptions: [],
      extensionPath: '/test/path',
      globalState: {
        get: jest.fn(),
        update: jest.fn(),
      },
      workspaceState: {
        get: jest.fn(),
        update: jest.fn(),
      },
    } as any;

    // Create mocked dependencies
    mockDeviceManager = new DeviceManager() as jest.Mocked<DeviceManager>;
    mockConfigManager = new ConfigurationManager() as jest.Mocked<ConfigurationManager>;

    // Initialize extension
    extension = new BifrostExtension(mockContext);
    (extension as any).deviceManager = mockDeviceManager;
    (extension as any).configManager = mockConfigManager;
  });

  afterEach(() => {
    extension.dispose();
  });

  describe('Device Discovery', () => {
    it('should execute discovery command successfully', async () => {
      const mockDevices = [
        {
          id: 'device-1',
          name: 'Modbus Device 1',
          protocol: 'modbus',
          host: '192.168.1.100',
          port: 502,
        },
        {
          id: 'device-2',
          name: 'EtherNet/IP Device',
          protocol: 'ethernetip',
          host: '192.168.1.101',
          port: 44818,
        },
      ];

      mockDeviceManager.discoverDevices.mockResolvedValue(mockDevices);
      
      // Mock user input for network range
      (vscode.window.showInputBox as jest.Mock).mockResolvedValue('192.168.1.0/24');
      
      // Mock progress reporting
      const mockProgress = {
        report: jest.fn(),
      };
      (vscode.window.withProgress as jest.Mock).mockImplementation((options, callback) => {
        return callback(mockProgress, {} as vscode.CancellationToken);
      });

      // Execute discovery command
      await extension.executeCommand('bifrost.discover');

      // Verify discovery was called with correct parameters
      expect(mockDeviceManager.discoverDevices).toHaveBeenCalledWith({
        networkRange: '192.168.1.0/24',
        protocols: ['modbus', 'ethernetip'],
        timeout: 30000,
      });

      // Verify progress was reported
      expect(mockProgress.report).toHaveBeenCalledWith({
        message: 'Scanning network for devices...',
      });

      // Verify success message was shown
      expect(vscode.window.showInformationMessage).toHaveBeenCalledWith(
        `Discovered 2 devices`
      );
    });

    it('should handle discovery errors gracefully', async () => {
      const error = new Error('Network timeout');
      mockDeviceManager.discoverDevices.mockRejectedValue(error);
      
      (vscode.window.showInputBox as jest.Mock).mockResolvedValue('192.168.1.0/24');
      (vscode.window.withProgress as jest.Mock).mockImplementation((options, callback) => {
        return callback({ report: jest.fn() }, {} as vscode.CancellationToken);
      });

      await extension.executeCommand('bifrost.discover');

      expect(vscode.window.showErrorMessage).toHaveBeenCalledWith(
        'Discovery failed: Network timeout'
      );
    });

    it('should handle user cancellation during network input', async () => {
      (vscode.window.showInputBox as jest.Mock).mockResolvedValue(undefined);

      await extension.executeCommand('bifrost.discover');

      expect(mockDeviceManager.discoverDevices).not.toHaveBeenCalled();
      expect(vscode.window.showInformationMessage).toHaveBeenCalledWith(
        'Discovery cancelled'
      );
    });

    it('should validate network range input', async () => {
      // Invalid network range
      (vscode.window.showInputBox as jest.Mock).mockResolvedValue('invalid-range');

      await extension.executeCommand('bifrost.discover');

      expect(vscode.window.showErrorMessage).toHaveBeenCalledWith(
        'Invalid network range format'
      );
      expect(mockDeviceManager.discoverDevices).not.toHaveBeenCalled();
    });
  });

  describe('Device Connection', () => {
    it('should connect to selected device', async () => {
      const mockDevice = {
        id: 'device-1',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      mockDeviceManager.getDevices.mockReturnValue([mockDevice]);
      mockDeviceManager.connectToDevice.mockResolvedValue(true);

      // Mock device selection
      (vscode.window.showQuickPick as jest.Mock).mockResolvedValue({
        label: 'Test Device',
        detail: 'modbus://192.168.1.100:502',
        device: mockDevice,
      });

      await extension.executeCommand('bifrost.connect');

      expect(mockDeviceManager.connectToDevice).toHaveBeenCalledWith(mockDevice);
      expect(vscode.window.showInformationMessage).toHaveBeenCalledWith(
        'Connected to Test Device'
      );
    });

    it('should handle connection failures', async () => {
      const mockDevice = {
        id: 'device-1',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      mockDeviceManager.getDevices.mockReturnValue([mockDevice]);
      mockDeviceManager.connectToDevice.mockRejectedValue(new Error('Connection timeout'));

      (vscode.window.showQuickPick as jest.Mock).mockResolvedValue({
        label: 'Test Device',
        device: mockDevice,
      });

      await extension.executeCommand('bifrost.connect');

      expect(vscode.window.showErrorMessage).toHaveBeenCalledWith(
        'Connection failed: Connection timeout'
      );
    });

    it('should show message when no devices available', async () => {
      mockDeviceManager.getDevices.mockReturnValue([]);

      await extension.executeCommand('bifrost.connect');

      expect(vscode.window.showInformationMessage).toHaveBeenCalledWith(
        'No devices available. Please run discovery first.'
      );
    });
  });

  describe('Tag Reading', () => {
    it('should read tags from connected device', async () => {
      const mockDevice = {
        id: 'device-1',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      const mockTags = [
        { name: 'temperature', address: '40001', dataType: 'float32' },
        { name: 'pressure', address: '40002', dataType: 'float32' },
      ];

      const mockReadResults = {
        temperature: { value: 25.5, timestamp: new Date(), quality: 'good' },
        pressure: { value: 101.3, timestamp: new Date(), quality: 'good' },
      };

      mockDeviceManager.getConnectedDevice.mockReturnValue(mockDevice);
      mockDeviceManager.readTags.mockResolvedValue(mockReadResults);

      // Mock tag selection
      (vscode.window.showQuickPick as jest.Mock).mockResolvedValue([
        { label: 'temperature', tag: mockTags[0] },
        { label: 'pressure', tag: mockTags[1] },
      ]);

      await extension.executeCommand('bifrost.readTags');

      expect(mockDeviceManager.readTags).toHaveBeenCalledWith(mockDevice, mockTags);
      
      // Should show results in output channel
      expect(vscode.window.createOutputChannel).toHaveBeenCalledWith('Bifrost');
    });

    it('should handle read errors', async () => {
      const mockDevice = {
        id: 'device-1',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      mockDeviceManager.getConnectedDevice.mockReturnValue(mockDevice);
      mockDeviceManager.readTags.mockRejectedValue(new Error('Read timeout'));

      (vscode.window.showQuickPick as jest.Mock).mockResolvedValue([
        { label: 'temperature', tag: { name: 'temperature', address: '40001' } },
      ]);

      await extension.executeCommand('bifrost.readTags');

      expect(vscode.window.showErrorMessage).toHaveBeenCalledWith(
        'Read failed: Read timeout'
      );
    });
  });

  describe('Configuration Management', () => {
    it('should save device configuration', async () => {
      const mockDevice = {
        id: 'device-1',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      mockDeviceManager.getDevices.mockReturnValue([mockDevice]);
      mockConfigManager.saveDeviceConfig.mockResolvedValue(true);

      (vscode.window.showQuickPick as jest.Mock).mockResolvedValue({
        label: 'Test Device',
        device: mockDevice,
      });

      await extension.executeCommand('bifrost.saveConfig');

      expect(mockConfigManager.saveDeviceConfig).toHaveBeenCalledWith(mockDevice);
      expect(vscode.window.showInformationMessage).toHaveBeenCalledWith(
        'Configuration saved for Test Device'
      );
    });

    it('should load device configuration', async () => {
      const mockConfigs = [
        {
          id: 'config-1',
          name: 'Saved Config 1',
          device: {
            id: 'device-1',
            name: 'Test Device',
            protocol: 'modbus',
            host: '192.168.1.100',
            port: 502,
          },
        },
      ];

      mockConfigManager.getConfigurations.mockReturnValue(mockConfigs);
      mockDeviceManager.addDevice.mockResolvedValue(true);

      (vscode.window.showQuickPick as jest.Mock).mockResolvedValue({
        label: 'Saved Config 1',
        config: mockConfigs[0],
      });

      await extension.executeCommand('bifrost.loadConfig');

      expect(mockDeviceManager.addDevice).toHaveBeenCalledWith(mockConfigs[0].device);
      expect(vscode.window.showInformationMessage).toHaveBeenCalledWith(
        'Configuration loaded: Saved Config 1'
      );
    });
  });

  describe('Status Bar', () => {
    it('should update status bar when device connects', async () => {
      const mockStatusBarItem = {
        text: '',
        tooltip: '',
        show: jest.fn(),
        hide: jest.fn(),
        color: '',
      };

      (vscode.window.createStatusBarItem as jest.Mock).mockReturnValue(mockStatusBarItem);

      // Initialize status bar
      extension.initializeStatusBar();

      // Simulate device connection
      const mockDevice = {
        id: 'device-1',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      extension.updateStatusBar(mockDevice, 'connected');

      expect(mockStatusBarItem.text).toBe('$(plug) Test Device');
      expect(mockStatusBarItem.tooltip).toBe('Connected to Test Device (modbus)');
      expect(mockStatusBarItem.show).toHaveBeenCalled();
    });

    it('should hide status bar when no device connected', async () => {
      const mockStatusBarItem = {
        text: '',
        tooltip: '',
        show: jest.fn(),
        hide: jest.fn(),
        color: '',
      };

      (vscode.window.createStatusBarItem as jest.Mock).mockReturnValue(mockStatusBarItem);

      extension.initializeStatusBar();
      extension.updateStatusBar(null, 'disconnected');

      expect(mockStatusBarItem.hide).toHaveBeenCalled();
    });
  });

  describe('WebView Panel', () => {
    it('should create webview panel for device monitoring', async () => {
      const mockPanel = {
        webview: {
          html: '',
          postMessage: jest.fn(),
          onDidReceiveMessage: jest.fn(),
        },
        onDidDispose: jest.fn(),
        dispose: jest.fn(),
      };

      (vscode.window.createWebviewPanel as jest.Mock).mockReturnValue(mockPanel);

      const mockDevice = {
        id: 'device-1',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      mockDeviceManager.getConnectedDevice.mockReturnValue(mockDevice);

      await extension.executeCommand('bifrost.openMonitor');

      expect(vscode.window.createWebviewPanel).toHaveBeenCalledWith(
        'bifrostMonitor',
        'Bifrost Device Monitor',
        vscode.ViewColumn.One,
        expect.objectContaining({
          enableScripts: true,
        })
      );
    });

    it('should handle webview messages', async () => {
      const mockPanel = {
        webview: {
          html: '',
          postMessage: jest.fn(),
          onDidReceiveMessage: jest.fn(),
        },
        onDidDispose: jest.fn(),
        dispose: jest.fn(),
      };

      (vscode.window.createWebviewPanel as jest.Mock).mockReturnValue(mockPanel);

      mockDeviceManager.getConnectedDevice.mockReturnValue({
        id: 'device-1',
        name: 'Test Device',
      });

      await extension.executeCommand('bifrost.openMonitor');

      // Get the message handler
      const messageHandler = (mockPanel.webview.onDidReceiveMessage as jest.Mock).mock.calls[0][0];

      // Simulate webview message
      const message = {
        command: 'readTag',
        tagName: 'temperature',
        address: '40001',
      };

      mockDeviceManager.readTags.mockResolvedValue({
        temperature: { value: 25.5, timestamp: new Date(), quality: 'good' },
      });

      await messageHandler(message);

      expect(mockPanel.webview.postMessage).toHaveBeenCalledWith({
        command: 'tagData',
        data: expect.objectContaining({
          temperature: expect.objectContaining({ value: 25.5 }),
        }),
      });
    });
  });

  describe('Tree View Provider', () => {
    it('should provide device tree structure', () => {
      const mockDevices = [
        {
          id: 'device-1',
          name: 'Test Device 1',
          protocol: 'modbus',
          host: '192.168.1.100',
          port: 502,
          connected: true,
        },
        {
          id: 'device-2',
          name: 'Test Device 2',
          protocol: 'ethernetip',
          host: '192.168.1.101',
          port: 44818,
          connected: false,
        },
      ];

      mockDeviceManager.getDevices.mockReturnValue(mockDevices);

      const treeDataProvider = extension.getTreeDataProvider();
      const children = treeDataProvider.getChildren();

      expect(children).toHaveLength(2);
      expect(children[0].label).toBe('Test Device 1');
      expect(children[0].iconPath).toMatch(/connected/);
      expect(children[1].label).toBe('Test Device 2');
      expect(children[1].iconPath).toMatch(/disconnected/);
    });

    it('should handle tree item selection', async () => {
      const mockDevice = {
        id: 'device-1',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      const treeDataProvider = extension.getTreeDataProvider();
      
      // Simulate tree item click
      await treeDataProvider.onTreeItemSelected(mockDevice);

      expect(vscode.commands.executeCommand).toHaveBeenCalledWith(
        'bifrost.selectDevice',
        mockDevice
      );
    });
  });

  describe('Error Handling', () => {
    it('should handle command execution errors', async () => {
      mockDeviceManager.discoverDevices.mockRejectedValue(new Error('Unexpected error'));
      
      (vscode.window.showInputBox as jest.Mock).mockResolvedValue('192.168.1.0/24');

      await extension.executeCommand('bifrost.discover');

      expect(vscode.window.showErrorMessage).toHaveBeenCalledWith(
        'Discovery failed: Unexpected error'
      );
    });

    it('should handle missing dependencies gracefully', async () => {
      // Simulate missing device manager
      (extension as any).deviceManager = null;

      await extension.executeCommand('bifrost.discover');

      expect(vscode.window.showErrorMessage).toHaveBeenCalledWith(
        'Extension not properly initialized'
      );
    });
  });

  describe('Configuration Validation', () => {
    it('should validate extension configuration', () => {
      const mockConfig = {
        get: jest.fn(),
      };

      (vscode.workspace.getConfiguration as jest.Mock).mockReturnValue(mockConfig);

      // Set up configuration values
      mockConfig.get.mockImplementation((key: string) => {
        switch (key) {
          case 'discoveryTimeout':
            return 30000;
          case 'connectionTimeout':
            return 5000;
          case 'enableLogging':
            return true;
          default:
            return undefined;
        }
      });

      const config = extension.getConfiguration();

      expect(config.discoveryTimeout).toBe(30000);
      expect(config.connectionTimeout).toBe(5000);
      expect(config.enableLogging).toBe(true);
    });

    it('should use default values for missing configuration', () => {
      const mockConfig = {
        get: jest.fn().mockReturnValue(undefined),
      };

      (vscode.workspace.getConfiguration as jest.Mock).mockReturnValue(mockConfig);

      const config = extension.getConfiguration();

      expect(config.discoveryTimeout).toBe(30000); // Default value
      expect(config.connectionTimeout).toBe(5000); // Default value
      expect(config.enableLogging).toBe(false); // Default value
    });
  });
});