// Mock VSCode API for testing
declare const vscode: {
    window: {
        showInformationMessage: jest.Mock<any, any, any>;
        showWarningMessage: jest.Mock<any, any, any>;
        showErrorMessage: jest.Mock<any, any, any>;
        createOutputChannel: jest.Mock<{
            appendLine: jest.Mock<any, any, any>;
            show: jest.Mock<any, any, any>;
            dispose: jest.Mock<any, any, any>;
        }, [], any>;
        createWebviewPanel: jest.Mock<any, any, any>;
        registerTreeDataProvider: jest.Mock<any, any, any>;
    };
    commands: {
        registerCommand: jest.Mock<any, any, any>;
        executeCommand: jest.Mock<any, any, any>;
    };
    workspace: {
        getConfiguration: jest.Mock<{
            get: jest.Mock<any, any, any>;
            update: jest.Mock<any, any, any>;
        }, [], any>;
        onDidChangeConfiguration: jest.Mock<any, any, any>;
    };
    Uri: {
        file: jest.Mock<{
            fsPath: string;
        }, [path: string], any>;
        parse: jest.Mock<any, any, any>;
    };
    ViewColumn: {
        One: number;
        Two: number;
        Three: number;
    };
    TreeItemCollapsibleState: {
        None: number;
        Collapsed: number;
        Expanded: number;
    };
    EventEmitter: jest.Mock<{
        fire: jest.Mock<any, any, any>;
        event: jest.Mock<any, any, any>;
    }, [], any>;
    Disposable: {
        from: jest.Mock<any, any, any>;
    };
};
export default vscode;
