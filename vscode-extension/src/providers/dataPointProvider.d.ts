import * as vscode from 'vscode';
import { DeviceManager, Device, Tag } from '../services/deviceManager';
export declare class DataPointProvider implements vscode.TreeDataProvider<DataPointItem> {
    private deviceManager;
    private _onDidChangeTreeData;
    readonly onDidChangeTreeData: vscode.Event<DataPointItem | undefined | null | void>;
    constructor(deviceManager: DeviceManager);
    refresh(): void;
    getTreeItem(element: DataPointItem): vscode.TreeItem;
    getChildren(element?: DataPointItem): Thenable<DataPointItem[]>;
    // Update real-time data from WebSocket
    updateRealTimeData(data: any): void;
}
export declare class DataPointItem extends vscode.TreeItem {
    readonly label: string;
    readonly collapsibleState: vscode.TreeItemCollapsibleState;
    readonly contextValue: string;
    readonly device?: Device;
    readonly tag?: Tag;
    constructor(label: string, collapsibleState: vscode.TreeItemCollapsibleState, contextValue: string, device?: Device, tag?: Tag);
}
