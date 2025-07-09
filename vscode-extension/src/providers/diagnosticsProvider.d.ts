import * as vscode from 'vscode';
import { DeviceManager } from '../services/deviceManager';
export declare class DiagnosticsProvider implements vscode.TreeDataProvider<DiagnosticItem> {
    private deviceManager;
    private _onDidChangeTreeData;
    readonly onDidChangeTreeData: vscode.Event<DiagnosticItem | undefined | null | void>;
    constructor(deviceManager: DeviceManager);
    refresh(): void;
    getTreeItem(element: DiagnosticItem): vscode.TreeItem;
    getChildren(element?: DiagnosticItem): Thenable<DiagnosticItem[]>;
}
export declare class DiagnosticItem extends vscode.TreeItem {
    readonly label: string;
    readonly collapsibleState: vscode.TreeItemCollapsibleState;
    readonly contextValue: string;
    constructor(label: string, collapsibleState: vscode.TreeItemCollapsibleState, contextValue: string);
}
