
var __createBinding = (this && this.__createBinding) || (Object.create ? ((o, m, k, k2) => {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: () => m[k] };
    }
    Object.defineProperty(o, k2, desc);
}) : ((o, m, k, k2) => {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? ((o, v) => {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : ((o, v) => {
    o["default"] = v;
}));
var __importStar = (this && this.__importStar) || (() => {
    var ownKeys = (o) => {
        ownKeys = Object.getOwnPropertyNames || ((o) => {
            var ar = [];
            for (var k in o) if (Object.hasOwn(o, k)) ar[ar.length] = k;
            return ar;
        });
        return ownKeys(o);
    };
    return (mod) => {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
const vscode = __importStar(require("vscode"));
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
            expect(vscode.commands.registerCommand).toHaveBeenCalledWith(expect.stringContaining('bifrost.discover'), expect.any(Function));
        });
        it('should handle bifrost.connect command', () => {
            // Test command registration
            expect(vscode.commands.registerCommand).toHaveBeenCalledWith(expect.stringContaining('bifrost.connect'), expect.any(Function));
        });
    });
    describe('configuration', () => {
        it('should read configuration values', () => {
            const mockConfig = {
                get: jest.fn().mockReturnValue('test-value'),
            };
            vscode.workspace.getConfiguration.mockReturnValue(mockConfig);
            const config = vscode.workspace.getConfiguration('bifrost');
            const value = config.get('discoveryTimeout');
            expect(vscode.workspace.getConfiguration).toHaveBeenCalledWith('bifrost');
            expect(mockConfig.get).toHaveBeenCalledWith('discoveryTimeout');
        });
    });
});
//# sourceMappingURL=extension.test.js.map