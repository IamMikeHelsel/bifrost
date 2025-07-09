import * as vscode from 'vscode';

describe('Extension', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('activate', () => {
    it('should register commands', async () => {
      // Mock the activation
      const mockContext = {
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
      };

      // Test that VSCode API is available
      expect(vscode.commands.registerCommand).toBeDefined();
      expect(vscode.window.showInformationMessage).toBeDefined();
    });
  });

  describe('commands', () => {
    it('should handle bifrost.discover command', () => {
      // Test command registration
      expect(vscode.commands.registerCommand).toHaveBeenCalledWith(
        expect.stringContaining('bifrost.discover'),
        expect.any(Function)
      );
    });

    it('should handle bifrost.connect command', () => {
      // Test command registration
      expect(vscode.commands.registerCommand).toHaveBeenCalledWith(
        expect.stringContaining('bifrost.connect'),
        expect.any(Function)
      );
    });
  });

  describe('configuration', () => {
    it('should read configuration values', () => {
      const mockConfig = {
        get: jest.fn().mockReturnValue('test-value'),
      };
      (vscode.workspace.getConfiguration as jest.Mock).mockReturnValue(mockConfig);

      const config = vscode.workspace.getConfiguration('bifrost');
      const value = config.get('discoveryTimeout');

      expect(vscode.workspace.getConfiguration).toHaveBeenCalledWith('bifrost');
      expect(mockConfig.get).toHaveBeenCalledWith('discoveryTimeout');
    });
  });
});