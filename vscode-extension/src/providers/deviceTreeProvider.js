
var __createBinding = (this && this.__createBinding) || (Object.create ? ((o, m, k, k2) => {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: () => m[k] };
    }
    Object.defineProperty(o, k2, desc);
}) : ((o, m, k, k2) => {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? ((o, v) => {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : ((o, v) => {
    o["default"] = v;
}));
var __importStar = (this && this.__importStar) || (() => {
    var ownKeys = (o) => {
        ownKeys = Object.getOwnPropertyNames || ((o) => {
            var ar = [];
            for (var k in o) if (Object.hasOwn(o, k)) ar[ar.length] = k;
            return ar;
        });
        return ownKeys(o);
    };
    return (mod) => {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.DeviceTreeItem = exports.DeviceTreeProvider = void 0;
const vscode = __importStar(require("vscode"));
const deviceManager_1 = require("../services/deviceManager");
class DeviceTreeProvider {
    deviceManager;
    _onDidChangeTreeData = new vscode.EventEmitter();
    onDidChangeTreeData = this._onDidChangeTreeData.event;
    constructor(deviceManager) {
        this.deviceManager = deviceManager;
        // Listen for device changes
        deviceManager.onDeviceChange(() => this.refresh());
    }
    refresh() {
        this._onDidChangeTreeData.fire();
    }
    getTreeItem(element) {
        return element;
    }
    getChildren(element) {
        if (!element) {
            // Root level - show device categories
            return Promise.resolve(this.getDeviceCategories());
        }
        else if (element.contextValue === 'category') {
            // Show devices in category
            return Promise.resolve(this.getDevicesInCategory(element.category));
        }
        else if (element.contextValue?.startsWith('device-')) {
            // Show device details
            return Promise.resolve(this.getDeviceDetails(element.device));
        }
        return Promise.resolve([]);
    }
    getDeviceCategories() {
        const devices = this.deviceManager.getAllDevices();
        const categories = new Set(devices.map(d => d.protocol));
        return Array.from(categories).map(category => {
            const count = devices.filter(d => d.protocol === category).length;
            const connected = devices.filter(d => d.protocol === category && d.status === deviceManager_1.DeviceStatus.Connected).length;
            return new DeviceTreeItem(`${this.getProtocolName(category)} (${connected}/${count})`, vscode.TreeItemCollapsibleState.Expanded, 'category', category);
        });
    }
    getDevicesInCategory(protocol) {
        const devices = this.deviceManager.getDevicesByProtocol(protocol);
        return devices.map(device => {
            const icon = this.getDeviceIcon(device);
            const contextValue = device.status === deviceManager_1.DeviceStatus.Connected
                ? 'device-connected'
                : 'device-disconnected';
            const item = new DeviceTreeItem(device.name, vscode.TreeItemCollapsibleState.Collapsed, contextValue, undefined, device);
            item.iconPath = new vscode.ThemeIcon(icon);
            item.description = `${device.address}:${device.port}`;
            if (device.status === deviceManager_1.DeviceStatus.Connected) {
                item.tooltip = `Connected to ${device.name}\nProtocol: ${device.protocol}\nAddress: ${device.address}:${device.port}`;
            }
            else if (device.status === deviceManager_1.DeviceStatus.Error) {
                item.tooltip = `Error: ${device.lastError}`;
                item.iconPath = new vscode.ThemeIcon('error');
            }
            return item;
        });
    }
    getDeviceDetails(device) {
        const details = [];
        // Status
        const statusItem = new DeviceTreeItem(`Status: ${device.status}`, vscode.TreeItemCollapsibleState.None, 'detail');
        statusItem.iconPath = new vscode.ThemeIcon(device.status === deviceManager_1.DeviceStatus.Connected ? 'check' : 'x');
        details.push(statusItem);
        // Connection info
        if (device.connectionTime) {
            const duration = this.formatDuration(Date.now() - device.connectionTime);
            const connItem = new DeviceTreeItem(`Connected: ${duration}`, vscode.TreeItemCollapsibleState.None, 'detail');
            connItem.iconPath = new vscode.ThemeIcon('clock');
            details.push(connItem);
        }
        // Statistics
        if (device.stats) {
            const statsItem = new DeviceTreeItem(`Requests: ${device.stats.totalRequests} (${device.stats.successRate}% success)`, vscode.TreeItemCollapsibleState.None, 'detail');
            statsItem.iconPath = new vscode.ThemeIcon('graph');
            details.push(statsItem);
        }
        // Last error
        if (device.lastError) {
            const errorItem = new DeviceTreeItem(`Last Error: ${device.lastError}`, vscode.TreeItemCollapsibleState.None, 'detail');
            errorItem.iconPath = new vscode.ThemeIcon('warning');
            details.push(errorItem);
        }
        return details;
    }
    getProtocolName(protocol) {
        const names = {
            'modbus-tcp': 'Modbus TCP',
            'modbus-rtu': 'Modbus RTU',
            'opcua': 'OPC UA',
            'ethernet-ip': 'Ethernet/IP',
            's7': 'Siemens S7'
        };
        return names[protocol] || protocol;
    }
    getDeviceIcon(device) {
        if (device.status === deviceManager_1.DeviceStatus.Connected) {
            return 'server-environment';
        }
        else if (device.status === deviceManager_1.DeviceStatus.Error) {
            return 'server-process';
        }
        return 'server';
    }
    formatDuration(ms) {
        const seconds = Math.floor(ms / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        if (hours > 0) {
            return `${hours}h ${minutes % 60}m`;
        }
        else if (minutes > 0) {
            return `${minutes}m ${seconds % 60}s`;
        }
        return `${seconds}s`;
    }
    // Update device status from WebSocket
    updateDeviceStatus(data) {
        // Process device status updates from Go gateway
        if (data.device_id && data.status) {
            const device = this.deviceManager.getDevice(data.device_id);
            if (device) {
                const newStatus = data.status === 'connected' ?
                    deviceManager_1.DeviceStatus.Connected :
                    data.status === 'error' ? deviceManager_1.DeviceStatus.Error : deviceManager_1.DeviceStatus.Disconnected;
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
exports.DeviceTreeProvider = DeviceTreeProvider;
class DeviceTreeItem extends vscode.TreeItem {
    label;
    collapsibleState;
    contextValue;
    category;
    device;
    constructor(label, collapsibleState, contextValue, category, device) {
        super(label, collapsibleState);
        this.label = label;
        this.collapsibleState = collapsibleState;
        this.contextValue = contextValue;
        this.category = category;
        this.device = device;
    }
}
exports.DeviceTreeItem = DeviceTreeItem;
//# sourceMappingURL=deviceTreeProvider.js.map