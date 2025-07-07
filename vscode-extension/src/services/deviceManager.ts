import * as vscode from 'vscode';
import { BifrostAPI } from '../api/bifrostAPI';

export enum DeviceStatus {
    Disconnected = 'Disconnected',
    Connecting = 'Connecting',
    Connected = 'Connected',
    Error = 'Error'
}

export interface DeviceStats {
    totalRequests: number;
    successfulRequests: number;
    failedRequests: number;
    successRate: number;
    averageResponseTime: number;
}

export interface Device {
    id: string;
    name: string;
    protocol: string;
    address: string;
    port: number;
    status: DeviceStatus;
    connectionTime?: number;
    lastError?: string;
    stats?: DeviceStats;
    tags?: Tag[];
}

export interface Tag {
    id: string;
    name: string;
    address: string;
    dataType: string;
    value?: any;
    lastUpdate?: number;
    writable: boolean;
    unit?: string;
    description?: string;
}

export class DeviceManager {
    private devices: Map<string, Device> = new Map();
    private _onDeviceChange = new vscode.EventEmitter<void>();
    private _onConnectionChange = new vscode.EventEmitter<number>();
    private _onDataUpdate = new vscode.EventEmitter<Tag>();
    
    readonly onDeviceChange = this._onDeviceChange.event;
    readonly onConnectionChange = this._onConnectionChange.event;
    readonly onDataUpdate = this._onDataUpdate.event;
    
    constructor(private api: BifrostAPI) {
        // Start periodic status updates
        this.startStatusUpdates();
    }
    
    async discoverDevices(): Promise<Device[]> {
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
        } catch (error) {
            vscode.window.showErrorMessage(`Device discovery failed: ${error}`);
            return [];
        }
    }
    
    async connectToDevice(device: Device): Promise<boolean> {
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
            } else {
                device.status = DeviceStatus.Error;
                device.lastError = 'Connection failed';
            }
            
            this._onDeviceChange.fire();
            this.updateConnectionCount();
            
            return success;
        } catch (error) {
            device.status = DeviceStatus.Error;
            device.lastError = error instanceof Error ? error.message : String(error);
            this._onDeviceChange.fire();
            return false;
        }
    }
    
    async disconnectFromDevice(device: Device): Promise<void> {
        try {
            await this.api.disconnectFromDevice(device);
            device.status = DeviceStatus.Disconnected;
            device.connectionTime = undefined;
            this.stopDataMonitoring(device);
        } catch (error) {
            console.error('Disconnect error:', error);
        } finally {
            this._onDeviceChange.fire();
            this.updateConnectionCount();
        }
    }
    
    getAllDevices(): Device[] {
        return Array.from(this.devices.values());
    }
    
    getDevice(id: string): Device | undefined {
        return this.devices.get(id);
    }
    
    getDevicesByProtocol(protocol: string): Device[] {
        return this.getAllDevices().filter(d => d.protocol === protocol);
    }
    
    getConnectedDevices(): Device[] {
        return this.getAllDevices().filter(d => d.status === DeviceStatus.Connected);
    }
    
    async readTag(device: Device, tag: Tag): Promise<any> {
        try {
            const value = await this.api.readTag(device.id, tag.address);
            tag.value = value;
            tag.lastUpdate = Date.now();
            this._onDataUpdate.fire(tag);
            return value;
        } catch (error) {
            throw error;
        }
    }
    
    async writeTag(device: Device, tag: Tag, value: any): Promise<void> {
        try {
            await this.api.writeTag(device.id, tag.address, value);
            tag.value = value;
            tag.lastUpdate = Date.now();
            this._onDataUpdate.fire(tag);
        } catch (error) {
            throw error;
        }
    }
    
    private dataMonitoringIntervals = new Map<string, NodeJS.Timeout>();
    
    private startDataMonitoring(device: Device): void {
        if (!device.tags || device.tags.length === 0) {
            return;
        }
        
        // Get update interval from settings
        const interval = vscode.workspace.getConfiguration('bifrost')
            .get<number>('dataUpdateInterval', 1000);
        
        const timer = setInterval(async () => {
            if (device.status !== DeviceStatus.Connected) {
                this.stopDataMonitoring(device);
                return;
            }
            
            // Update all tags
            for (const tag of device.tags!) {
                try {
                    await this.readTag(device, tag);
                } catch (error) {
                    console.error(`Failed to read tag ${tag.name}:`, error);
                }
            }
        }, interval);
        
        this.dataMonitoringIntervals.set(device.id, timer);
    }
    
    private stopDataMonitoring(device: Device): void {
        const timer = this.dataMonitoringIntervals.get(device.id);
        if (timer) {
            clearInterval(timer);
            this.dataMonitoringIntervals.delete(device.id);
        }
    }
    
    private startStatusUpdates(): void {
        // Update device statistics every 5 seconds
        setInterval(async () => {
            for (const device of this.getConnectedDevices()) {
                try {
                    const stats = await this.api.getDeviceStats(device.id);
                    device.stats = stats;
                } catch (error) {
                    console.error(`Failed to update stats for ${device.name}:`, error);
                }
            }
            this._onDeviceChange.fire();
        }, 5000);
    }
    
    private updateConnectionCount(): void {
        const count = this.getConnectedDevices().length;
        this._onConnectionChange.fire(count);
    }
    
    dispose(): void {
        // Stop all monitoring
        this.dataMonitoringIntervals.forEach(timer => clearInterval(timer));
        this.dataMonitoringIntervals.clear();
        
        this._onDeviceChange.dispose();
        this._onConnectionChange.dispose();
        this._onDataUpdate.dispose();
    }
}