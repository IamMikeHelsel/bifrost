import * as vscode from 'vscode';
import type { DeviceManager } from '../services/deviceManager';

export class DiagnosticsProvider implements vscode.TreeDataProvider<DiagnosticItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<DiagnosticItem | undefined | null | void> = 
        new vscode.EventEmitter<DiagnosticItem | undefined | null | void>();
    readonly onDidChangeTreeData: vscode.Event<DiagnosticItem | undefined | null | void> = 
        this._onDidChangeTreeData.event;

    constructor(private deviceManager: DeviceManager) {
        // Refresh periodically
        setInterval(() => this.refresh(), 5000);
    }

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: DiagnosticItem): vscode.TreeItem {
        return element;
    }

    getChildren(element?: DiagnosticItem): Thenable<DiagnosticItem[]> {
        const items: DiagnosticItem[] = [];
        
        // System status
        const connectedCount = this.deviceManager.getConnectedDevices().length;
        const totalCount = this.deviceManager.getAllDevices().length;
        
        const systemItem = new DiagnosticItem(
            'System Status',
            vscode.TreeItemCollapsibleState.None,
            'system'
        );
        systemItem.description = `${connectedCount}/${totalCount} devices connected`;
        systemItem.iconPath = new vscode.ThemeIcon(
            connectedCount === totalCount ? 'check' : 
            connectedCount > 0 ? 'warning' : 'error'
        );
        items.push(systemItem);
        
        // Python/Bifrost status
        const pythonItem = new DiagnosticItem(
            'Python Environment',
            vscode.TreeItemCollapsibleState.None,
            'python'
        );
        pythonItem.description = 'Python 3.13+ with Bifrost';
        pythonItem.iconPath = new vscode.ThemeIcon('check');
        items.push(pythonItem);
        
        // Performance metrics
        const devices = this.deviceManager.getConnectedDevices();
        let totalRequests = 0;
        let avgResponseTime = 0;
        let errorCount = 0;
        
        devices.forEach(device => {
            if (device.stats) {
                totalRequests += device.stats.totalRequests;
                avgResponseTime += device.stats.averageResponseTime;
                errorCount += device.stats.failedRequests;
            }
        });
        
        if (devices.length > 0) {
            avgResponseTime /= devices.length;
        }
        
        const perfItem = new DiagnosticItem(
            'Performance',
            vscode.TreeItemCollapsibleState.None,
            'performance'
        );
        perfItem.description = `${avgResponseTime.toFixed(1)}ms avg response`;
        perfItem.iconPath = new vscode.ThemeIcon(
            avgResponseTime < 50 ? 'check' : 
            avgResponseTime < 100 ? 'warning' : 'error'
        );
        items.push(perfItem);
        
        // Error summary
        const errorItem = new DiagnosticItem(
            'Errors',
            vscode.TreeItemCollapsibleState.None,
            'errors'
        );
        errorItem.description = errorCount > 0 ? `${errorCount} total errors` : 'No errors';
        errorItem.iconPath = new vscode.ThemeIcon(errorCount > 0 ? 'warning' : 'check');
        items.push(errorItem);
        
        // Recent issues
        const devicesWithErrors = devices.filter(d => d.lastError);
        if (devicesWithErrors.length > 0) {
            const issuesItem = new DiagnosticItem(
                'Recent Issues',
                vscode.TreeItemCollapsibleState.None,
                'issues'
            );
            issuesItem.description = `${devicesWithErrors.length} devices with errors`;
            issuesItem.iconPath = new vscode.ThemeIcon('warning');
            items.push(issuesItem);
        }
        
        return Promise.resolve(items);
    }
}

export class DiagnosticItem extends vscode.TreeItem {
    constructor(
        public readonly label: string,
        public readonly collapsibleState: vscode.TreeItemCollapsibleState,
        public readonly contextValue: string
    ) {
        super(label, collapsibleState);
    }
}