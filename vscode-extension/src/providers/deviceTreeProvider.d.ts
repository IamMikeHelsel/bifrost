import * as vscode from 'vscode';
import { DeviceManager, Device } from '../services/deviceManager';
export declare class DeviceTreeProvider implements vscode.TreeDataProvider<DeviceTreeItem> {
    private deviceManager;
    private _onDidChangeTreeData;
    readonly onDidChangeTreeData: vscode.Event<DeviceTreeItem | undefined | null | void>;
    constructor(deviceManager: DeviceManager);
    refresh(): void;
    getTreeItem(element: DeviceTreeItem): vscode.TreeItem;
    getChildren(element?: DeviceTreeItem): Thenable<DeviceTreeItem[]>;
    private getDeviceCategories;
    private getDevicesInCategory;
    private getDeviceDetails;
    private getProtocolName;
    private getDeviceIcon;
    private formatDuration;
    // Update device status from WebSocket
    updateDeviceStatus(data: any): void;
}
export declare class DeviceTreeItem extends vscode.TreeItem {
    readonly label: string;
    readonly collapsibleState: vscode.TreeItemCollapsibleState;
    readonly contextValue: string;
    readonly category?: string;
    readonly device?: Device;
    constructor(label: string, collapsibleState: vscode.TreeItemCollapsibleState, contextValue: string, category?: string, device?: Device);
}
