import * as vscode from 'vscode';
import { Device, Tag } from '../services/deviceManager';

export class MonitorPanel {
    public static currentPanel: MonitorPanel | undefined;
    private readonly _panel: vscode.WebviewPanel;
    private readonly _extensionUri: vscode.Uri;
    private _disposables: vscode.Disposable[] = [];
    private _device: Device;
    private _dataBuffer: Map<string, Array<{time: number, value: number}>> = new Map();
    
    public static createOrShow(extensionUri: vscode.Uri, device: Device) {
        const column = vscode.window.activeTextEditor
            ? vscode.window.activeTextEditor.viewColumn
            : undefined;
        
        // If we already have a panel, show it
        if (MonitorPanel.currentPanel) {
            MonitorPanel.currentPanel._panel.reveal(column);
            MonitorPanel.currentPanel.updateDevice(device);
            return;
        }
        
        // Otherwise, create a new panel
        const panel = vscode.window.createWebviewPanel(
            'bifrostMonitor',
            `Bifrost Monitor: ${device.name}`,
            column || vscode.ViewColumn.One,
            {
                enableScripts: true,
                retainContextWhenHidden: true,
                localResourceRoots: [
                    vscode.Uri.joinPath(extensionUri, 'media'),
                    vscode.Uri.joinPath(extensionUri, 'node_modules')
                ]
            }
        );
        
        MonitorPanel.currentPanel = new MonitorPanel(panel, extensionUri, device);
    }
    
    private constructor(panel: vscode.WebviewPanel, extensionUri: vscode.Uri, device: Device) {
        this._panel = panel;
        this._extensionUri = extensionUri;
        this._device = device;
        
        // Set the webview's initial html content
        this._update();
        
        // Listen for when the panel is disposed
        this._panel.onDidDispose(() => this.dispose(), null, this._disposables);
        
        // Update the content based on view changes
        this._panel.onDidChangeViewState(
            e => {
                if (this._panel.visible) {
                    this._update();
                }
            },
            null,
            this._disposables
        );
        
        // Handle messages from the webview
        this._panel.webview.onDidReceiveMessage(
            message => {
                switch (message.command) {
                    case 'refresh':
                        this._update();
                        return;
                    case 'export':
                        this.exportData();
                        return;
                    case 'writeTag':
                        this.writeTag(message.tagId, message.value);
                        return;
                }
            },
            null,
            this._disposables
        );
    }
    
    public updateDevice(device: Device) {
        this._device = device;
        this._panel.title = `Bifrost Monitor: ${device.name}`;
        this._update();
    }
    
    public updateData(tag: Tag) {
        // Buffer data for charts
        const buffer = this._dataBuffer.get(tag.id) || [];
        buffer.push({
            time: Date.now(),
            value: parseFloat(tag.value) || 0
        });
        
        // Keep only last N points
        const maxPoints = vscode.workspace.getConfiguration('bifrost')
            .get<number>('maxDataPoints', 100);
        if (buffer.length > maxPoints) {
            buffer.shift();
        }
        
        this._dataBuffer.set(tag.id, buffer);
        
        // Send update to webview
        this._panel.webview.postMessage({
            command: 'dataUpdate',
            tag: tag,
            data: buffer
        });
    }
    
    private async exportData() {
        const format = await vscode.window.showQuickPick(['CSV', 'JSON'], {
            placeHolder: 'Select export format'
        });
        
        if (!format) {
            return;
        }
        
        // TODO: Implement actual export
        vscode.window.showInformationMessage(`Exporting data as ${format}...`);
    }
    
    private async writeTag(tagId: string, value: any) {
        // TODO: Implement tag writing
        vscode.window.showInformationMessage(`Writing ${value} to ${tagId}...`);
    }
    
    public dispose() {
        MonitorPanel.currentPanel = undefined;
        
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
        this._panel.webview.html = this._getHtmlForWebview(webview);
    }
    
    private _getHtmlForWebview(webview: vscode.Webview) {
        // Get resource URIs
        const scriptUri = webview.asWebviewUri(
            vscode.Uri.joinPath(this._extensionUri, 'media', 'monitor.js')
        );
        const styleUri = webview.asWebviewUri(
            vscode.Uri.joinPath(this._extensionUri, 'media', 'monitor.css')
        );
        const chartUri = webview.asWebviewUri(
            vscode.Uri.joinPath(this._extensionUri, 'node_modules', 'chart.js', 'dist', 'chart.umd.js')
        );
        
        // Use a nonce to only allow specific scripts to be run
        const nonce = getNonce();
        
        return `<!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src ${webview.cspSource} 'unsafe-inline'; script-src 'nonce-${nonce}';">
                <link href="${styleUri}" rel="stylesheet">
                <title>Bifrost Monitor</title>
            </head>
            <body>
                <div class="container">
                    <header>
                        <h1>${this._device.name}</h1>
                        <div class="device-info">
                            <span class="protocol">${this._device.protocol.toUpperCase()}</span>
                            <span class="address">${this._device.address}:${this._device.port}</span>
                            <span class="status status-${this._device.status.toLowerCase()}">${this._device.status}</span>
                        </div>
                        <div class="actions">
                            <button onclick="refresh()">Refresh</button>
                            <button onclick="exportData()">Export</button>
                        </div>
                    </header>
                    
                    <div class="stats" id="stats">
                        ${this._device.stats ? `
                            <div class="stat">
                                <div class="stat-label">Total Requests</div>
                                <div class="stat-value">${this._device.stats.totalRequests}</div>
                            </div>
                            <div class="stat">
                                <div class="stat-label">Success Rate</div>
                                <div class="stat-value">${this._device.stats.successRate}%</div>
                            </div>
                            <div class="stat">
                                <div class="stat-label">Avg Response</div>
                                <div class="stat-value">${this._device.stats.averageResponseTime.toFixed(1)}ms</div>
                            </div>
                        ` : '<div class="loading">Loading statistics...</div>'}
                    </div>
                    
                    <div class="tags" id="tags">
                        <h2>Data Points</h2>
                        ${this._device.tags ? this._device.tags.map(tag => `
                            <div class="tag" data-tag-id="${tag.id}">
                                <div class="tag-header">
                                    <h3>${tag.name}</h3>
                                    <span class="tag-address">${tag.address}</span>
                                </div>
                                <div class="tag-value">
                                    <span class="value">${tag.value !== undefined ? tag.value : '---'}</span>
                                    ${tag.unit ? `<span class="unit">${tag.unit}</span>` : ''}
                                </div>
                                ${tag.writable ? `
                                    <div class="tag-controls">
                                        <input type="number" id="input-${tag.id}" placeholder="New value">
                                        <button onclick="writeTag('${tag.id}')">Write</button>
                                    </div>
                                ` : ''}
                                <div class="chart-container">
                                    <canvas id="chart-${tag.id}"></canvas>
                                </div>
                            </div>
                        `).join('') : '<div class="loading">Loading tags...</div>'}
                    </div>
                </div>
                
                <script nonce="${nonce}" src="${chartUri}"></script>
                <script nonce="${nonce}" src="${scriptUri}"></script>
                <script nonce="${nonce}">
                    // Initialize with device data
                    const device = ${JSON.stringify(this._device)};
                    const vscode = acquireVsCodeApi();
                    
                    function refresh() {
                        vscode.postMessage({ command: 'refresh' });
                    }
                    
                    function exportData() {
                        vscode.postMessage({ command: 'export' });
                    }
                    
                    function writeTag(tagId) {
                        const input = document.getElementById('input-' + tagId);
                        if (input && input.value) {
                            vscode.postMessage({
                                command: 'writeTag',
                                tagId: tagId,
                                value: parseFloat(input.value)
                            });
                            input.value = '';
                        }
                    }
                    
                    // Handle messages from extension
                    window.addEventListener('message', event => {
                        const message = event.data;
                        switch (message.command) {
                            case 'dataUpdate':
                                updateTagData(message.tag, message.data);
                                break;
                        }
                    });
                </script>
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