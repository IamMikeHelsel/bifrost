
Object.defineProperty(exports, "__esModule", { value: true });
// Mock VSCode API for testing
const vscode = {
    window: {
        showInformationMessage: jest.fn(),
        showWarningMessage: jest.fn(),
        showErrorMessage: jest.fn(),
        createOutputChannel: jest.fn(() => ({
            appendLine: jest.fn(),
            show: jest.fn(),
            dispose: jest.fn(),
        })),
        createWebviewPanel: jest.fn(),
        registerTreeDataProvider: jest.fn(),
    },
    commands: {
        registerCommand: jest.fn(),
        executeCommand: jest.fn(),
    },
    workspace: {
        getConfiguration: jest.fn(() => ({
            get: jest.fn(),
            update: jest.fn(),
        })),
        onDidChangeConfiguration: jest.fn(),
    },
    Uri: {
        file: jest.fn((path) => ({ fsPath: path })),
        parse: jest.fn(),
    },
    ViewColumn: {
        One: 1,
        Two: 2,
        Three: 3,
    },
    TreeItemCollapsibleState: {
        None: 0,
        Collapsed: 1,
        Expanded: 2,
    },
    EventEmitter: jest.fn(() => ({
        fire: jest.fn(),
        event: jest.fn(),
    })),
    Disposable: {
        from: jest.fn(),
    },
};
// Mock the vscode module
jest.mock('vscode', () => vscode, { virtual: true });
exports.default = vscode;
//# sourceMappingURL=setup.js.map