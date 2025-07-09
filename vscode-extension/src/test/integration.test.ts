import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import { BifrostExtension } from '../extension';

describe('Bifrost Extension Integration Tests', () => {
  let extension: BifrostExtension;
  let workspaceRoot: string;

  beforeAll(async () => {
    // Set up test workspace
    workspaceRoot = path.join(__dirname, '..', '..', 'test-workspace');
    if (!fs.existsSync(workspaceRoot)) {
      fs.mkdirSync(workspaceRoot, { recursive: true });
    }

    // Create mock workspace folder
    const workspaceFolder: vscode.WorkspaceFolder = {
      uri: vscode.Uri.file(workspaceRoot),
      name: 'test-workspace',
      index: 0,
    };

    // Mock workspace
    (vscode.workspace.workspaceFolders as any) = [workspaceFolder];
    (vscode.workspace.getWorkspaceFolder as jest.Mock).mockReturnValue(workspaceFolder);
  });

  beforeEach(async () => {
    jest.clearAllMocks();
    
    const mockContext: vscode.ExtensionContext = {
      subscriptions: [],
      extensionPath: path.join(__dirname, '..', '..'),
      globalState: {
        get: jest.fn(),
        update: jest.fn(),
      },
      workspaceState: {
        get: jest.fn(),
        update: jest.fn(),
      },
    } as any;

    extension = new BifrostExtension(mockContext);
    await extension.activate();
  });

  afterEach(async () => {
    if (extension) {
      extension.dispose();
    }
  });

  afterAll(async () => {
    // Clean up test workspace
    if (fs.existsSync(workspaceRoot)) {
      fs.rmSync(workspaceRoot, { recursive: true, force: true });
    }
  });

  describe('Extension Lifecycle', () => {
    it('should activate successfully', async () => {
      expect(extension).toBeDefined();
      expect(extension.isActive()).toBe(true);
    });

    it('should register all commands', async () => {
      const expectedCommands = [
        'bifrost.discover',
        'bifrost.connect',
        'bifrost.disconnect',
        'bifrost.readTags',
        'bifrost.writeTags',
        'bifrost.saveConfig',
        'bifrost.loadConfig',
        'bifrost.openMonitor',
        'bifrost.refreshDevices',
      ];

      for (const command of expectedCommands) {
        expect(vscode.commands.registerCommand).toHaveBeenCalledWith(
          command,
          expect.any(Function)
        );
      }
    });

    it('should register tree data provider', async () => {
      expect(vscode.window.registerTreeDataProvider).toHaveBeenCalledWith(
        'bifrostDevices',
        expect.any(Object)
      );
    });

    it('should create status bar item', async () => {
      expect(vscode.window.createStatusBarItem).toHaveBeenCalled();
    });

    it('should dispose resources on deactivation', async () => {
      const disposeSpy = jest.spyOn(extension, 'dispose');
      
      extension.dispose();
      
      expect(disposeSpy).toHaveBeenCalled();
    });
  });

  describe('Configuration File Handling', () => {
    it('should create configuration file in workspace', async () => {
      const configPath = path.join(workspaceRoot, '.vscode', 'bifrost.json');
      
      // Mock file system operations
      const writeFileSpy = jest.spyOn(fs, 'writeFileSync').mockImplementation();
      const mkdirSpy = jest.spyOn(fs, 'mkdirSync').mockImplementation();

      const testConfig = {
        devices: [
          {
            id: 'test-device',
            name: 'Test Device',
            protocol: 'modbus',
            host: '192.168.1.100',
            port: 502,
          },
        ],
      };

      await extension.saveWorkspaceConfig(testConfig);

      expect(mkdirSpy).toHaveBeenCalledWith(
        path.dirname(configPath),
        { recursive: true }
      );
      expect(writeFileSpy).toHaveBeenCalledWith(
        configPath,
        JSON.stringify(testConfig, null, 2)
      );

      writeFileSpy.mockRestore();
      mkdirSpy.mockRestore();
    });

    it('should load configuration file from workspace', async () => {
      const configPath = path.join(workspaceRoot, '.vscode', 'bifrost.json');
      const testConfig = {
        devices: [
          {
            id: 'test-device',
            name: 'Test Device',
            protocol: 'modbus',
            host: '192.168.1.100',
            port: 502,
          },
        ],
      };

      // Mock file system operations
      const existsSpy = jest.spyOn(fs, 'existsSync').mockReturnValue(true);
      const readFileSpy = jest.spyOn(fs, 'readFileSync').mockReturnValue(
        JSON.stringify(testConfig)
      );

      const loadedConfig = await extension.loadWorkspaceConfig();

      expect(existsSpy).toHaveBeenCalledWith(configPath);
      expect(readFileSpy).toHaveBeenCalledWith(configPath, 'utf8');
      expect(loadedConfig).toEqual(testConfig);

      existsSpy.mockRestore();
      readFileSpy.mockRestore();
    });

    it('should handle missing configuration file gracefully', async () => {
      const existsSpy = jest.spyOn(fs, 'existsSync').mockReturnValue(false);

      const loadedConfig = await extension.loadWorkspaceConfig();

      expect(loadedConfig).toEqual({ devices: [] });

      existsSpy.mockRestore();
    });

    it('should handle corrupted configuration file', async () => {
      const existsSpy = jest.spyOn(fs, 'existsSync').mockReturnValue(true);
      const readFileSpy = jest.spyOn(fs, 'readFileSync').mockReturnValue(
        'invalid json content'
      );

      const loadedConfig = await extension.loadWorkspaceConfig();

      expect(loadedConfig).toEqual({ devices: [] });
      expect(vscode.window.showWarningMessage).toHaveBeenCalledWith(
        'Failed to load Bifrost configuration: Invalid JSON format'
      );

      existsSpy.mockRestore();
      readFileSpy.mockRestore();
    });
  });

  describe('Device Discovery Workflow', () => {
    it('should complete full discovery workflow', async () => {
      // Mock successful discovery
      const mockDevices = [
        {
          id: 'modbus-device-1',
          name: 'Modbus Device 1',
          protocol: 'modbus',
          host: '192.168.1.100',
          port: 502,
        },
        {
          id: 'eip-device-1',
          name: 'EtherNet/IP Device 1',
          protocol: 'ethernetip',
          host: '192.168.1.101',
          port: 44818,
        },
      ];

      // Mock device manager discovery
      const deviceManager = (extension as any).deviceManager;
      deviceManager.discoverDevices = jest.fn().mockResolvedValue(mockDevices);

      // Mock user inputs
      (vscode.window.showInputBox as jest.Mock)
        .mockResolvedValueOnce('192.168.1.0/24') // Network range
        .mockResolvedValueOnce('30'); // Timeout

      (vscode.window.showQuickPick as jest.Mock).mockResolvedValue([
        'modbus',
        'ethernetip',
      ]);

      // Mock progress dialog
      (vscode.window.withProgress as jest.Mock).mockImplementation(
        (options, callback) => {
          const progress = { report: jest.fn() };
          const token = { isCancellationRequested: false };
          return callback(progress, token);
        }
      );

      // Execute discovery
      await vscode.commands.executeCommand('bifrost.discover');

      // Verify discovery was called
      expect(deviceManager.discoverDevices).toHaveBeenCalledWith({
        networkRange: '192.168.1.0/24',
        protocols: ['modbus', 'ethernetip'],
        timeout: 30000,
      });

      // Verify devices were added
      expect(deviceManager.addDevice).toHaveBeenCalledTimes(2);
      expect(deviceManager.addDevice).toHaveBeenCalledWith(mockDevices[0]);
      expect(deviceManager.addDevice).toHaveBeenCalledWith(mockDevices[1]);

      // Verify tree view was refreshed
      expect(vscode.commands.executeCommand).toHaveBeenCalledWith(
        'bifrost.refreshTreeView'
      );
    });

    it('should handle discovery cancellation', async () => {
      // Mock user cancelling input
      (vscode.window.showInputBox as jest.Mock).mockResolvedValue(undefined);

      await vscode.commands.executeCommand('bifrost.discover');

      // Verify discovery was not called
      const deviceManager = (extension as any).deviceManager;
      expect(deviceManager.discoverDevices).not.toHaveBeenCalled();

      expect(vscode.window.showInformationMessage).toHaveBeenCalledWith(
        'Discovery cancelled'
      );
    });
  });

  describe('Device Connection Workflow', () => {
    it('should complete full connection workflow', async () => {
      const mockDevice = {
        id: 'test-device',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      const deviceManager = (extension as any).deviceManager;
      deviceManager.getDevices = jest.fn().mockReturnValue([mockDevice]);
      deviceManager.connectToDevice = jest.fn().mockResolvedValue(true);
      deviceManager.getDeviceInfo = jest.fn().mockResolvedValue({
        vendor: 'Test Vendor',
        model: 'Test Model',
        version: '1.0',
      });

      // Mock device selection
      (vscode.window.showQuickPick as jest.Mock).mockResolvedValue({
        label: 'Test Device',
        detail: 'modbus://192.168.1.100:502',
        device: mockDevice,
      });

      // Execute connection
      await vscode.commands.executeCommand('bifrost.connect');

      // Verify connection was attempted
      expect(deviceManager.connectToDevice).toHaveBeenCalledWith(mockDevice);

      // Verify success message
      expect(vscode.window.showInformationMessage).toHaveBeenCalledWith(
        'Connected to Test Device'
      );

      // Verify status bar was updated
      const statusBar = (extension as any).statusBarItem;
      expect(statusBar.text).toContain('Test Device');
      expect(statusBar.show).toHaveBeenCalled();
    });

    it('should handle connection timeout', async () => {
      const mockDevice = {
        id: 'test-device',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      const deviceManager = (extension as any).deviceManager;
      deviceManager.getDevices = jest.fn().mockReturnValue([mockDevice]);
      deviceManager.connectToDevice = jest.fn().mockRejectedValue(
        new Error('Connection timeout')
      );

      (vscode.window.showQuickPick as jest.Mock).mockResolvedValue({
        label: 'Test Device',
        device: mockDevice,
      });

      await vscode.commands.executeCommand('bifrost.connect');

      expect(vscode.window.showErrorMessage).toHaveBeenCalledWith(
        'Connection failed: Connection timeout'
      );
    });
  });

  describe('Tag Operations Workflow', () => {
    it('should read tags from connected device', async () => {
      const mockDevice = {
        id: 'test-device',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      const mockTags = [
        { name: 'temperature', address: '40001', dataType: 'float32' },
        { name: 'pressure', address: '40002', dataType: 'float32' },
      ];

      const mockResults = {
        temperature: {
          value: 25.5,
          timestamp: new Date(),
          quality: 'good',
        },
        pressure: {
          value: 101.3,
          timestamp: new Date(),
          quality: 'good',
        },
      };

      const deviceManager = (extension as any).deviceManager;
      deviceManager.getConnectedDevice = jest.fn().mockReturnValue(mockDevice);
      deviceManager.getDeviceTags = jest.fn().mockReturnValue(mockTags);
      deviceManager.readTags = jest.fn().mockResolvedValue(mockResults);

      // Mock tag selection
      (vscode.window.showQuickPick as jest.Mock).mockResolvedValue([
        { label: 'temperature', tag: mockTags[0] },
        { label: 'pressure', tag: mockTags[1] },
      ]);

      // Mock output channel
      const mockOutputChannel = {
        appendLine: jest.fn(),
        show: jest.fn(),
      };
      (vscode.window.createOutputChannel as jest.Mock).mockReturnValue(
        mockOutputChannel
      );

      await vscode.commands.executeCommand('bifrost.readTags');

      expect(deviceManager.readTags).toHaveBeenCalledWith(mockDevice, mockTags);

      // Verify results were displayed
      expect(mockOutputChannel.appendLine).toHaveBeenCalledWith(
        expect.stringContaining('temperature: 25.5')
      );
      expect(mockOutputChannel.appendLine).toHaveBeenCalledWith(
        expect.stringContaining('pressure: 101.3')
      );
      expect(mockOutputChannel.show).toHaveBeenCalled();
    });

    it('should write tags to connected device', async () => {
      const mockDevice = {
        id: 'test-device',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      const mockTag = {
        name: 'setpoint',
        address: '40100',
        dataType: 'float32',
        writable: true,
      };

      const deviceManager = (extension as any).deviceManager;
      deviceManager.getConnectedDevice = jest.fn().mockReturnValue(mockDevice);
      deviceManager.getWritableTags = jest.fn().mockReturnValue([mockTag]);
      deviceManager.writeTag = jest.fn().mockResolvedValue(true);

      // Mock tag and value selection
      (vscode.window.showQuickPick as jest.Mock).mockResolvedValue({
        label: 'setpoint',
        tag: mockTag,
      });
      (vscode.window.showInputBox as jest.Mock).mockResolvedValue('30.0');

      await vscode.commands.executeCommand('bifrost.writeTags');

      expect(deviceManager.writeTag).toHaveBeenCalledWith(
        mockDevice,
        mockTag,
        30.0
      );

      expect(vscode.window.showInformationMessage).toHaveBeenCalledWith(
        'Successfully wrote value 30 to setpoint'
      );
    });
  });

  describe('WebView Integration', () => {
    it('should create and configure webview panel', async () => {
      const mockPanel = {
        webview: {
          html: '',
          postMessage: jest.fn(),
          onDidReceiveMessage: jest.fn(),
          asWebviewUri: jest.fn((uri) => uri),
        },
        onDidDispose: jest.fn(),
        reveal: jest.fn(),
        dispose: jest.fn(),
      };

      (vscode.window.createWebviewPanel as jest.Mock).mockReturnValue(mockPanel);

      const deviceManager = (extension as any).deviceManager;
      deviceManager.getConnectedDevice = jest.fn().mockReturnValue({
        id: 'test-device',
        name: 'Test Device',
      });

      await vscode.commands.executeCommand('bifrost.openMonitor');

      expect(vscode.window.createWebviewPanel).toHaveBeenCalledWith(
        'bifrostMonitor',
        'Bifrost Device Monitor',
        vscode.ViewColumn.One,
        {
          enableScripts: true,
          retainContextWhenHidden: true,
          localResourceRoots: [
            vscode.Uri.file(path.join(extension.extensionPath, 'media')),
          ],
        }
      );

      // Verify HTML content was set
      expect(mockPanel.webview.html).toContain('Bifrost Device Monitor');
    });

    it('should handle webview message communication', async () => {
      const mockPanel = {
        webview: {
          html: '',
          postMessage: jest.fn(),
          onDidReceiveMessage: jest.fn(),
          asWebviewUri: jest.fn((uri) => uri),
        },
        onDidDispose: jest.fn(),
        dispose: jest.fn(),
      };

      (vscode.window.createWebviewPanel as jest.Mock).mockReturnValue(mockPanel);

      const deviceManager = (extension as any).deviceManager;
      deviceManager.getConnectedDevice = jest.fn().mockReturnValue({
        id: 'test-device',
        name: 'Test Device',
      });

      await vscode.commands.executeCommand('bifrost.openMonitor');

      // Get the message handler
      const messageHandler = mockPanel.webview.onDidReceiveMessage.mock.calls[0][0];

      // Test different message types
      const testMessages = [
        {
          command: 'ready',
          expected: { command: 'deviceInfo', data: expect.any(Object) },
        },
        {
          command: 'readTag',
          tagName: 'temperature',
          expected: { command: 'tagData', data: expect.any(Object) },
        },
        {
          command: 'startPolling',
          interval: 1000,
          expected: { command: 'pollingStarted' },
        },
      ];

      for (const testMessage of testMessages) {
        await messageHandler(testMessage);
        expect(mockPanel.webview.postMessage).toHaveBeenCalledWith(
          expect.objectContaining(testMessage.expected)
        );
      }
    });
  });

  describe('Error Recovery', () => {
    it('should recover from device disconnection', async () => {
      const mockDevice = {
        id: 'test-device',
        name: 'Test Device',
        protocol: 'modbus',
        host: '192.168.1.100',
        port: 502,
      };

      const deviceManager = (extension as any).deviceManager;
      deviceManager.getConnectedDevice = jest.fn().mockReturnValue(mockDevice);

      // Simulate connection initially working
      deviceManager.isConnected = jest.fn().mockReturnValue(true);

      // Then simulate disconnection
      setTimeout(() => {
        deviceManager.isConnected.mockReturnValue(false);
        // Trigger disconnection event
        extension.onDeviceDisconnected(mockDevice);
      }, 100);

      // Wait for disconnection handling
      await new Promise((resolve) => setTimeout(resolve, 200));

      // Verify status bar was updated
      const statusBar = (extension as any).statusBarItem;
      expect(statusBar.hide).toHaveBeenCalled();

      // Verify notification was shown
      expect(vscode.window.showWarningMessage).toHaveBeenCalledWith(
        'Device Test Device disconnected'
      );
    });

    it('should handle extension initialization errors', async () => {
      // Create extension with invalid context
      const invalidContext = null as any;

      expect(() => {
        new BifrostExtension(invalidContext);
      }).toThrow();
    });
  });

  describe('Performance', () => {
    it('should handle large number of devices efficiently', async () => {
      const deviceManager = (extension as any).deviceManager;
      
      // Create 1000 mock devices
      const mockDevices = Array.from({ length: 1000 }, (_, i) => ({
        id: `device-${i}`,
        name: `Device ${i}`,
        protocol: 'modbus',
        host: `192.168.${Math.floor(i / 256)}.${i % 256}`,
        port: 502,
      }));

      deviceManager.getDevices = jest.fn().mockReturnValue(mockDevices);

      const startTime = performance.now();
      
      // Get tree data provider and request children
      const treeProvider = extension.getTreeDataProvider();
      const children = await treeProvider.getChildren();
      
      const endTime = performance.now();
      const duration = endTime - startTime;

      expect(children).toHaveLength(1000);
      expect(duration).toBeLessThan(100); // Should complete in under 100ms
    });

    it('should handle rapid command execution', async () => {
      const deviceManager = (extension as any).deviceManager;
      deviceManager.getDevices = jest.fn().mockReturnValue([]);

      // Execute multiple commands rapidly
      const promises = Array.from({ length: 100 }, () =>
        vscode.commands.executeCommand('bifrost.refreshDevices')
      );

      const startTime = performance.now();
      await Promise.all(promises);
      const endTime = performance.now();
      const duration = endTime - startTime;

      expect(duration).toBeLessThan(1000); // Should complete in under 1 second
    });
  });
});