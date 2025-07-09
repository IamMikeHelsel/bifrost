
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
exports.DataPointItem = exports.DataPointProvider = void 0;
const vscode = __importStar(require("vscode"));
class DataPointProvider {
    deviceManager;
    _onDidChangeTreeData = new vscode.EventEmitter();
    onDidChangeTreeData = this._onDidChangeTreeData.event;
    constructor(deviceManager) {
        this.deviceManager = deviceManager;
        // Listen for data updates
        deviceManager.onDataUpdate(() => this.refresh());
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
            // Root level - show connected devices
            const devices = this.deviceManager.getConnectedDevices();
            if (devices.length === 0) {
                return Promise.resolve([]);
            }
            return Promise.resolve(devices.map(device => new DataPointItem(device.name, vscode.TreeItemCollapsibleState.Expanded, 'device', device)));
        }
        else if (element.contextValue === 'device' && element.device) {
            // Show tags for device
            const tags = element.device.tags || [];
            return Promise.resolve(tags.map(tag => {
                const item = new DataPointItem(tag.name, vscode.TreeItemCollapsibleState.None, tag.writable ? 'tag-writable' : 'tag', element.device, tag);
                // Format value display
                let valueStr = '---';
                if (tag.value !== undefined) {
                    if (typeof tag.value === 'number') {
                        valueStr = tag.value.toFixed(2);
                    }
                    else {
                        valueStr = String(tag.value);
                    }
                }
                item.description = `${valueStr}${tag.unit ? ' ' + tag.unit : ''}`;
                // Set icon based on data type
                if (tag.dataType === 'boolean') {
                    item.iconPath = new vscode.ThemeIcon(tag.value ? 'check' : 'x');
                }
                else if (tag.writable) {
                    item.iconPath = new vscode.ThemeIcon('edit');
                }
                else {
                    item.iconPath = new vscode.ThemeIcon('symbol-numeric');
                }
                // Tooltip with details
                item.tooltip = `${tag.name}\nAddress: ${tag.address}\nType: ${tag.dataType}\nValue: ${valueStr}${tag.unit ? ' ' + tag.unit : ''}\n${tag.description || ''}`;
                return item;
            }));
        }
        return Promise.resolve([]);
    }
    // Update real-time data from WebSocket
    updateRealTimeData(data) {
        // Process real-time data updates from Go gateway
        if (data.device_id && data.tag_data) {
            const device = this.deviceManager.getDevice(data.device_id);
            if (device && device.tags) {
                // Update tag values with real-time data
                for (const tagUpdate of data.tag_data) {
                    const tag = device.tags.find(t => t.address === tagUpdate.address);
                    if (tag) {
                        tag.value = tagUpdate.value;
                        tag.timestamp = new Date(tagUpdate.timestamp);
                    }
                }
                // Refresh the tree to show updated values
                this.refresh();
            }
        }
    }
}
exports.DataPointProvider = DataPointProvider;
class DataPointItem extends vscode.TreeItem {
    label;
    collapsibleState;
    contextValue;
    device;
    tag;
    constructor(label, collapsibleState, contextValue, device, tag) {
        super(label, collapsibleState);
        this.label = label;
        this.collapsibleState = collapsibleState;
        this.contextValue = contextValue;
        this.device = device;
        this.tag = tag;
    }
}
exports.DataPointItem = DataPointItem;
//# sourceMappingURL=dataPointProvider.js.map