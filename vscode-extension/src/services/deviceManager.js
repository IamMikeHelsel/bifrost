
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
exports.DeviceManager = exports.DeviceStatus = void 0;
const vscode = __importStar(require("vscode"));
var DeviceStatus;
((DeviceStatus) => {
    DeviceStatus["Disconnected"] = "Disconnected";
    DeviceStatus["Connecting"] = "Connecting";
    DeviceStatus["Connected"] = "Connected";
    DeviceStatus["Error"] = "Error";
})(DeviceStatus || (exports.DeviceStatus = DeviceStatus = {}));
class DeviceManager {
    api;
    devices = new Map();
    _onDeviceChange = new vscode.EventEmitter();
    _onConnectionChange = new vscode.EventEmitter();
    _onDataUpdate = new vscode.EventEmitter();
    onDeviceChange = this._onDeviceChange.event;
    onConnectionChange = this._onConnectionChange.event;
    onDataUpdate = this._onDataUpdate.event;
    constructor(api) {
        this.api = api;
        // Start periodic status updates
        this.startStatusUpdates();
    }
    async discoverDevices() {
        try {
            const discovered = await this.api.discoverDevices();
            // Add discovered devices
            discovered.forEach(device => {
                if (!this.devices.has(device.id)) {
                    this.devices.set(device.id, device);
                }
            });
            this._onDeviceChange.fire();
            return discovered;
        }
        catch (error) {
            vscode.window.showErrorMessage(`Device discovery failed: ${error}`);
            return [];
        }
    }
    async connectToDevice(device) {
        try {
            device.status = DeviceStatus.Connecting;
            this._onDeviceChange.fire();
            const success = await this.api.connectToDevice(device);
            if (success) {
                device.status = DeviceStatus.Connected;
                device.connectionTime = Date.now();
                device.lastError = undefined;
                // Load initial tags
                const tags = await this.api.getDeviceTags(device.id);
                device.tags = tags;
                // Start data monitoring
                this.startDataMonitoring(device);
            }
            else {
                device.status = DeviceStatus.Error;
                device.lastError = 'Connection failed';
            }
            this._onDeviceChange.fire();
            this.updateConnectionCount();
            return success;
        }
        catch (error) {
            device.status = DeviceStatus.Error;
            device.lastError = error instanceof Error ? error.message : String(error);
            this._onDeviceChange.fire();
            return false;
        }
    }
    async disconnectFromDevice(device) {
        try {
            await this.api.disconnectFromDevice(device);
            device.status = DeviceStatus.Disconnected;
            device.connectionTime = undefined;
            this.stopDataMonitoring(device);
        }
        catch (error) {
            console.error('Disconnect error:', error);
        }
        finally {
            this._onDeviceChange.fire();
            this.updateConnectionCount();
        }
    }
    getAllDevices() {
        return Array.from(this.devices.values());
    }
    getDevice(id) {
        return this.devices.get(id);
    }
    getDevicesByProtocol(protocol) {
        return this.getAllDevices().filter(d => d.protocol === protocol);
    }
    getConnectedDevices() {
        return this.getAllDevices().filter(d => d.status === DeviceStatus.Connected);
    }
    async readTag(device, tag) {
        try {
            const value = await this.api.readTag(device.id, tag.address);
            tag.value = value;
            tag.lastUpdate = Date.now();
            this._onDataUpdate.fire(tag);
            return value;
        }
        catch (error) {
            throw error;
        }
    }
    async writeTag(device, tag, value) {
        try {
            await this.api.writeTag(device.id, tag.address, value);
            tag.value = value;
            tag.lastUpdate = Date.now();
            this._onDataUpdate.fire(tag);
        }
        catch (error) {
            throw error;
        }
    }
    dataMonitoringIntervals = new Map();
    startDataMonitoring(device) {
        if (!device.tags || device.tags.length === 0) {
            return;
        }
        // Get update interval from settings
        const interval = vscode.workspace.getConfiguration('bifrost')
            .get('dataUpdateInterval', 1000);
        const timer = setInterval(async () => {
            if (device.status !== DeviceStatus.Connected) {
                this.stopDataMonitoring(device);
                return;
            }
            // Update all tags
            for (const tag of device.tags) {
                try {
                    await this.readTag(device, tag);
                }
                catch (error) {
                    console.error(`Failed to read tag ${tag.name}:`, error);
                }
            }
        }, interval);
        this.dataMonitoringIntervals.set(device.id, timer);
    }
    stopDataMonitoring(device) {
        const timer = this.dataMonitoringIntervals.get(device.id);
        if (timer) {
            clearInterval(timer);
            this.dataMonitoringIntervals.delete(device.id);
        }
    }
    startStatusUpdates() {
        // Update device statistics every 5 seconds
        setInterval(async () => {
            for (const device of this.getConnectedDevices()) {
                try {
                    const stats = await this.api.getDeviceStats(device.id);
                    device.stats = stats;
                }
                catch (error) {
                    console.error(`Failed to update stats for ${device.name}:`, error);
                }
            }
            this._onDeviceChange.fire();
        }, 5000);
    }
    updateConnectionCount() {
        const count = this.getConnectedDevices().length;
        this._onConnectionChange.fire(count);
    }
    dispose() {
        // Stop all monitoring
        this.dataMonitoringIntervals.forEach(timer => clearInterval(timer));
        this.dataMonitoringIntervals.clear();
        this._onDeviceChange.dispose();
        this._onConnectionChange.dispose();
        this._onDataUpdate.dispose();
    }
}
exports.DeviceManager = DeviceManager;
//# sourceMappingURL=deviceManager.js.map