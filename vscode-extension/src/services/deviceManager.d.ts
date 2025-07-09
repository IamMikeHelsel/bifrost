import * as vscode from 'vscode';
import { BifrostAPI } from '../api/bifrostAPI';
export declare enum DeviceStatus {
    Disconnected = "Disconnected",
    Connecting = "Connecting",
    Connected = "Connected",
    Error = "Error"
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
    lastSeen?: Date;
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
    timestamp?: Date;
    writable: boolean;
    unit?: string;
    description?: string;
}
export declare class DeviceManager {
    private api;
    private devices;
    private _onDeviceChange;
    private _onConnectionChange;
    private _onDataUpdate;
    readonly onDeviceChange: vscode.Event<void>;
    readonly onConnectionChange: vscode.Event<number>;
    readonly onDataUpdate: vscode.Event<Tag>;
    constructor(api: BifrostAPI);
    discoverDevices(): Promise<Device[]>;
    connectToDevice(device: Device): Promise<boolean>;
    disconnectFromDevice(device: Device): Promise<void>;
    getAllDevices(): Device[];
    getDevice(id: string): Device | undefined;
    getDevicesByProtocol(protocol: string): Device[];
    getConnectedDevices(): Device[];
    readTag(device: Device, tag: Tag): Promise<any>;
    writeTag(device: Device, tag: Tag, value: any): Promise<void>;
    private dataMonitoringIntervals;
    private startDataMonitoring;
    private stopDataMonitoring;
    private startStatusUpdates;
    private updateConnectionCount;
    dispose(): void;
}
