import * as vscode from 'vscode';
import { DeviceManager, Device, Tag } from '../services/deviceManager';

export class DataPointProvider implements vscode.TreeDataProvider<DataPointItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<DataPointItem | undefined | null | void> = 
        new vscode.EventEmitter<DataPointItem | undefined | null | void>();
    readonly onDidChangeTreeData: vscode.Event<DataPointItem | undefined | null | void> = 
        this._onDidChangeTreeData.event;

    constructor(private deviceManager: DeviceManager) {
        // Listen for data updates
        deviceManager.onDataUpdate(() => this.refresh());
        deviceManager.onDeviceChange(() => this.refresh());
    }

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: DataPointItem): vscode.TreeItem {
        return element;
    }

    getChildren(element?: DataPointItem): Thenable<DataPointItem[]> {
        if (!element) {
            // Root level - show connected devices
            const devices = this.deviceManager.getConnectedDevices();
            if (devices.length === 0) {
                return Promise.resolve([]);
            }
            
            return Promise.resolve(devices.map(device => 
                new DataPointItem(
                    device.name,
                    vscode.TreeItemCollapsibleState.Expanded,
                    'device',
                    device
                )
            ));
        } else if (element.contextValue === 'device' && element.device) {
            // Show tags for device
            const tags = element.device.tags || [];
            return Promise.resolve(tags.map(tag => {
                const item = new DataPointItem(
                    tag.name,
                    vscode.TreeItemCollapsibleState.None,
                    tag.writable ? 'tag-writable' : 'tag',
                    element.device,
                    tag
                );
                
                // Format value display
                let valueStr = '---';
                if (tag.value !== undefined) {
                    if (typeof tag.value === 'number') {
                        valueStr = tag.value.toFixed(2);
                    } else {
                        valueStr = String(tag.value);
                    }
                }
                
                item.description = `${valueStr}${tag.unit ? ' ' + tag.unit : ''}`;
                
                // Set icon based on data type
                if (tag.dataType === 'boolean') {
                    item.iconPath = new vscode.ThemeIcon(tag.value ? 'check' : 'x');
                } else if (tag.writable) {
                    item.iconPath = new vscode.ThemeIcon('edit');
                } else {
                    item.iconPath = new vscode.ThemeIcon('symbol-numeric');
                }
                
                // Tooltip with details
                item.tooltip = `${tag.name}\nAddress: ${tag.address}\nType: ${tag.dataType}\nValue: ${valueStr}${tag.unit ? ' ' + tag.unit : ''}\n${tag.description || ''}`;
                
                return item;
            }));
        }
        
        return Promise.resolve([]);
    }
}

export class DataPointItem extends vscode.TreeItem {
    constructor(
        public readonly label: string,
        public readonly collapsibleState: vscode.TreeItemCollapsibleState,
        public readonly contextValue: string,
        public readonly device?: Device,
        public readonly tag?: Tag
    ) {
        super(label, collapsibleState);
    }
}