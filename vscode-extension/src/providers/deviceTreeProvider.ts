import * as vscode from 'vscode';
import { DeviceManager, Device, DeviceStatus } from '../services/deviceManager';

export class DeviceTreeProvider implements vscode.TreeDataProvider<DeviceTreeItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<DeviceTreeItem | undefined | null | void> = 
        new vscode.EventEmitter<DeviceTreeItem | undefined | null | void>();
    readonly onDidChangeTreeData: vscode.Event<DeviceTreeItem | undefined | null | void> = 
        this._onDidChangeTreeData.event;

    constructor(private deviceManager: DeviceManager) {
        // Listen for device changes
        deviceManager.onDeviceChange(() => this.refresh());
    }

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: DeviceTreeItem): vscode.TreeItem {
        return element;
    }

    getChildren(element?: DeviceTreeItem): Thenable<DeviceTreeItem[]> {
        if (!element) {
            // Root level - show device categories
            return Promise.resolve(this.getDeviceCategories());
        } else if (element.contextValue === 'category') {
            // Show devices in category
            return Promise.resolve(this.getDevicesInCategory(element.category!));
        } else if (element.contextValue?.startsWith('device-')) {
            // Show device details
            return Promise.resolve(this.getDeviceDetails(element.device!));
        }
        return Promise.resolve([]);
    }

    private getDeviceCategories(): DeviceTreeItem[] {
        const devices = this.deviceManager.getAllDevices();
        const categories = new Set(devices.map(d => d.protocol));
        
        return Array.from(categories).map(category => {
            const count = devices.filter(d => d.protocol === category).length;
            const connected = devices.filter(d => 
                d.protocol === category && d.status === DeviceStatus.Connected
            ).length;
            
            return new DeviceTreeItem(
                `${this.getProtocolName(category)} (${connected}/${count})`,
                vscode.TreeItemCollapsibleState.Expanded,
                'category',
                category
            );
        });
    }

    private getDevicesInCategory(protocol: string): DeviceTreeItem[] {
        const devices = this.deviceManager.getDevicesByProtocol(protocol);
        
        return devices.map(device => {
            const icon = this.getDeviceIcon(device);
            const contextValue = device.status === DeviceStatus.Connected 
                ? 'device-connected' 
                : 'device-disconnected';
            
            const item = new DeviceTreeItem(
                device.name,
                vscode.TreeItemCollapsibleState.Collapsed,
                contextValue,
                undefined,
                device
            );
            
            item.iconPath = new vscode.ThemeIcon(icon);
            item.description = `${device.address}:${device.port}`;
            
            if (device.status === DeviceStatus.Connected) {
                item.tooltip = `Connected to ${device.name}\nProtocol: ${device.protocol}\nAddress: ${device.address}:${device.port}`;
            } else if (device.status === DeviceStatus.Error) {
                item.tooltip = `Error: ${device.lastError}`;
                item.iconPath = new vscode.ThemeIcon('error');
            }
            
            return item;
        });
    }

    private getDeviceDetails(device: Device): DeviceTreeItem[] {
        const details: DeviceTreeItem[] = [];
        
        // Status
        const statusItem = new DeviceTreeItem(
            `Status: ${device.status}`,
            vscode.TreeItemCollapsibleState.None,
            'detail'
        );
        statusItem.iconPath = new vscode.ThemeIcon(
            device.status === DeviceStatus.Connected ? 'check' : 'x'
        );
        details.push(statusItem);
        
        // Connection info
        if (device.connectionTime) {
            const duration = this.formatDuration(Date.now() - device.connectionTime);
            const connItem = new DeviceTreeItem(
                `Connected: ${duration}`,
                vscode.TreeItemCollapsibleState.None,
                'detail'
            );
            connItem.iconPath = new vscode.ThemeIcon('clock');
            details.push(connItem);
        }
        
        // Statistics
        if (device.stats) {
            const statsItem = new DeviceTreeItem(
                `Requests: ${device.stats.totalRequests} (${device.stats.successRate}% success)`,
                vscode.TreeItemCollapsibleState.None,
                'detail'
            );
            statsItem.iconPath = new vscode.ThemeIcon('graph');
            details.push(statsItem);
        }
        
        // Last error
        if (device.lastError) {
            const errorItem = new DeviceTreeItem(
                `Last Error: ${device.lastError}`,
                vscode.TreeItemCollapsibleState.None,
                'detail'
            );
            errorItem.iconPath = new vscode.ThemeIcon('warning');
            details.push(errorItem);
        }
        
        return details;
    }

    private getProtocolName(protocol: string): string {
        const names: Record<string, string> = {
            'modbus-tcp': 'Modbus TCP',
            'modbus-rtu': 'Modbus RTU',
            'opcua': 'OPC UA',
            'ethernet-ip': 'Ethernet/IP',
            's7': 'Siemens S7'
        };
        return names[protocol] || protocol;
    }

    private getDeviceIcon(device: Device): string {
        if (device.status === DeviceStatus.Connected) {
            return 'server-environment';
        } else if (device.status === DeviceStatus.Error) {
            return 'server-process';
        }
        return 'server';
    }

    private formatDuration(ms: number): string {
        const seconds = Math.floor(ms / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        
        if (hours > 0) {
            return `${hours}h ${minutes % 60}m`;
        } else if (minutes > 0) {
            return `${minutes}m ${seconds % 60}s`;
        }
        return `${seconds}s`;
    }
    
    // Update device status from WebSocket
    updateDeviceStatus(data: any): void {
        // Process device status updates from Go gateway
        if (data.device_id && data.status) {
            const device = this.deviceManager.getDevice(data.device_id);
            if (device) {
                const newStatus = data.status === 'connected' ? 
                    DeviceStatus.Connected : 
                    data.status === 'error' ? DeviceStatus.Error : DeviceStatus.Disconnected;
                
                if (device.status !== newStatus) {
                    device.status = newStatus;
                    device.lastSeen = new Date();
                    
                    // Refresh the tree to show updated status
                    this.refresh();
                }
            }
        }
    }
}

export class DeviceTreeItem extends vscode.TreeItem {
    constructor(
        public readonly label: string,
        public readonly collapsibleState: vscode.TreeItemCollapsibleState,
        public readonly contextValue: string,
        public readonly category?: string,
        public readonly device?: Device
    ) {
        super(label, collapsibleState);
    }
}