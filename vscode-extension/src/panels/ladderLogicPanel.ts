import * as vscode from 'vscode';
import * as path from 'path';

export class LadderLogicPanel {
    public static readonly viewType = 'ladderLogic';
    private readonly _panel: vscode.WebviewPanel;
    private readonly _extensionUri: vscode.Uri;
    private _disposables: vscode.Disposable[] = [];

    public static createOrShow(extensionUri: vscode.Uri, device?: any) {
        const column = vscode.window.activeTextEditor
            ? vscode.window.activeTextEditor.viewColumn
            : undefined;

        // Create a new panel
        const panel = vscode.window.createWebviewPanel(
            LadderLogicPanel.viewType,
            'Ladder Logic Viewer',
            column || vscode.ViewColumn.One,
            {
                enableScripts: true,
                localResourceRoots: [
                    vscode.Uri.joinPath(extensionUri, 'media'),
                    vscode.Uri.joinPath(extensionUri, 'out'),
                    vscode.Uri.joinPath(extensionUri, 'node_modules')
                ]
            }
        );

        return new LadderLogicPanel(panel, extensionUri, device);
    }

    private constructor(panel: vscode.WebviewPanel, extensionUri: vscode.Uri, private _device?: any) {
        this._panel = panel;
        this._extensionUri = extensionUri;

        // Set the webview's initial html content
        this._update();

        // Listen for when the panel is disposed
        this._panel.onDidDispose(() => this.dispose(), null, this._disposables);

        // Handle messages from the webview
        this._panel.webview.onDidReceiveMessage(
            message => {
                switch (message.command) {
                    case 'alert':
                        vscode.window.showErrorMessage(message.text);
                        return;
                    case 'loadProgram':
                        this._loadProgramFromPLC();
                        return;
                    case 'exportDiagram':
                        this._exportDiagram();
                        return;
                    case 'toggleRealTime':
                        this._toggleRealTimeMonitoring();
                        return;
                }
            },
            null,
            this._disposables
        );
    }

    public dispose() {
        // Clean up our resources
        this._panel.dispose();

        while (this._disposables.length) {
            const x = this._disposables.pop();
            if (x) {
                x.dispose();
            }
        }
    }

    private _update() {
        const webview = this._panel.webview;
        this._panel.title = this._device ? `Ladder Logic - ${this._device.name}` : 'Ladder Logic Viewer';
        this._panel.webview.html = this._getHtmlForWebview(webview);
    }

    private _loadProgramFromPLC() {
        // TODO: Implement program upload from connected PLC
        vscode.window.showInformationMessage('Loading ladder logic program from PLC...');
        
        // Simulate loading a basic ladder logic program
        this._panel.webview.postMessage({
            command: 'updateProgram',
            program: this._getSampleLadderLogic()
        });
    }

    private _exportDiagram() {
        // TODO: Implement export functionality
        vscode.window.showInformationMessage('Exporting ladder logic diagram...');
    }

    private _toggleRealTimeMonitoring() {
        // TODO: Implement real-time monitoring toggle
        vscode.window.showInformationMessage('Toggling real-time monitoring...');
    }

    private _getSampleLadderLogic() {
        return {
            rungs: [
                {
                    id: 'rung_001',
                    elements: [
                        { type: 'contact', id: 'input_001', address: 'I:0/0', position: { x: 50, y: 100 }, state: true },
                        { type: 'contact', id: 'input_002', address: 'I:0/1', position: { x: 150, y: 100 }, state: false },
                        { type: 'coil', id: 'output_001', address: 'O:0/0', position: { x: 300, y: 100 }, state: true }
                    ],
                    connections: [
                        { from: 'input_001', to: 'input_002' },
                        { from: 'input_002', to: 'output_001' }
                    ]
                },
                {
                    id: 'rung_002',
                    elements: [
                        { type: 'contact', id: 'input_003', address: 'I:0/2', position: { x: 50, y: 200 }, state: false },
                        { type: 'timer', id: 'timer_001', address: 'T4:0', position: { x: 150, y: 200 }, preset: 5000, accumulated: 1250 },
                        { type: 'coil', id: 'output_002', address: 'O:0/1', position: { x: 300, y: 200 }, state: false }
                    ],
                    connections: [
                        { from: 'input_003', to: 'timer_001' },
                        { from: 'timer_001', to: 'output_002' }
                    ]
                }
            ]
        };
    }

    private _getHtmlForWebview(webview: vscode.Webview) {
        // Local path to CSS and JS files
        const styleResetUri = webview.asWebviewUri(vscode.Uri.joinPath(this._extensionUri, 'media', 'reset.css'));
        const styleVSCodeUri = webview.asWebviewUri(vscode.Uri.joinPath(this._extensionUri, 'media', 'vscode.css'));
        const styleLadderUri = webview.asWebviewUri(vscode.Uri.joinPath(this._extensionUri, 'media', 'ladderLogic.css'));
        
        // JointJS and D3 from node_modules
        const jointjsUri = webview.asWebviewUri(vscode.Uri.joinPath(this._extensionUri, 'node_modules', '@joint/core', 'dist', 'joint.min.js'));
        const d3Uri = webview.asWebviewUri(vscode.Uri.joinPath(this._extensionUri, 'node_modules', 'd3', 'dist', 'd3.min.js'));
        
        const scriptUri = webview.asWebviewUri(vscode.Uri.joinPath(this._extensionUri, 'media', 'ladderLogic.js'));

        // Use a nonce to only allow specific scripts to be run
        const nonce = getNonce();

        return `<!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src ${webview.cspSource}; script-src 'nonce-${nonce}';">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                
                <link href="${styleResetUri}" rel="stylesheet">
                <link href="${styleVSCodeUri}" rel="stylesheet">
                <link href="${styleLadderUri}" rel="stylesheet">
                
                <title>Ladder Logic Viewer</title>
            </head>
            <body>
                <div class="ladder-logic-container">
                    <div class="toolbar">
                        <button id="load-program" class="button-primary">Load from PLC</button>
                        <button id="export-diagram" class="button-secondary">Export</button>
                        <button id="toggle-realtime" class="button-secondary">Real-time Monitor</button>
                        <div class="device-info">
                            ${this._device ? `Connected to: ${this._device.name} (${this._device.address})` : 'No device connected'}
                        </div>
                    </div>
                    
                    <div class="ladder-diagram-container">
                        <div id="ladder-diagram" class="ladder-diagram"></div>
                    </div>
                    
                    <div class="status-bar">
                        <div class="status-item">Status: <span id="connection-status">Ready</span></div>
                        <div class="status-item">Rungs: <span id="rung-count">0</span></div>
                        <div class="status-item">Elements: <span id="element-count">0</span></div>
                    </div>
                </div>
                
                <script nonce="${nonce}" src="${d3Uri}"></script>
                <script nonce="${nonce}" src="${jointjsUri}"></script>
                <script nonce="${nonce}" src="${scriptUri}"></script>
            </body>
            </html>`;
    }
}

function getNonce() {
    let text = '';
    const possible = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    for (let i = 0; i < 32; i++) {
        text += possible.charAt(Math.floor(Math.random() * possible.length));
    }
    return text;
}