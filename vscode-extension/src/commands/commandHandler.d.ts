import * as vscode from 'vscode';
import { DeviceManager } from '../services/deviceManager';
import { DeviceTreeProvider } from '../providers/deviceTreeProvider';
import { DataPointProvider } from '../providers/dataPointProvider';
export declare class CommandHandler {
    private context;
    private deviceManager;
    private deviceTreeProvider;
    private dataPointProvider;
    constructor(context: vscode.ExtensionContext, deviceManager: DeviceManager, deviceTreeProvider: DeviceTreeProvider, dataPointProvider: DataPointProvider);
    discoverDevices(): Promise<void>;
    connectToDevice(item: any): Promise<void>;
    disconnectFromDevice(item: any): Promise<void>;
    openMonitor(item: any): Promise<void>;
    refreshDevices(): Promise<void>;
    readTag(item: any): Promise<void>;
    writeTag(item: any): Promise<void>;
    exportData(): Promise<void>;
    enableTypescriptGo(): Promise<void>;
    benchmarkPerformance(): Promise<void>;
}
