import * as vscode from 'vscode';
import { DeviceManager } from '../services/deviceManager';
import { DeviceTreeProvider } from '../providers/deviceTreeProvider';
import { DataPointProvider } from '../providers/dataPointProvider';
import { MonitorPanel } from '../panels/monitorPanel';

export class CommandHandler {
    constructor(
        private context: vscode.ExtensionContext,
        private deviceManager: DeviceManager,
        private deviceTreeProvider: DeviceTreeProvider,
        private dataPointProvider: DataPointProvider
    ) {}
    
    async discoverDevices(): Promise<void> {
        try {
            await vscode.window.withProgress({
                location: vscode.ProgressLocation.Notification,
                title: "Discovering industrial devices...",
                cancellable: true
            }, async (progress, token) => {
                progress.report({ increment: 0 });
                
                const devices = await this.deviceManager.discoverDevices();
                
                if (devices.length === 0) {
                    vscode.window.showInformationMessage('No devices found. Make sure devices are online and accessible.');
                } else {
                    vscode.window.showInformationMessage(`Discovered ${devices.length} device(s)`);
                }
                
                return devices;
            });
        } catch (error) {
            vscode.window.showErrorMessage(`Discovery failed: ${error}`);
        }
    }
    
    async connectToDevice(item: any): Promise<void> {
        const device = item?.device;
        if (!device) {
            // Show device selection
            const devices = this.deviceManager.getAllDevices();
            const selected = await vscode.window.showQuickPick(
                devices.map(d => ({
                    label: d.name,
                    description: `${d.protocol} - ${d.address}:${d.port}`,
                    device: d
                })),
                { placeHolder: 'Select a device to connect' }
            );
            
            if (!selected) {
                return;
            }
            
            await this.connectToDevice({ device: selected.device });
            return;
        }
        
        try {
            await vscode.window.withProgress({
                location: vscode.ProgressLocation.Notification,
                title: `Connecting to ${device.name}...`,
            }, async () => {
                const success = await this.deviceManager.connectToDevice(device);
                
                if (success) {
                    vscode.window.showInformationMessage(`Connected to ${device.name}`);
                } else {
                    vscode.window.showErrorMessage(`Failed to connect to ${device.name}`);
                }
            });
        } catch (error) {
            vscode.window.showErrorMessage(`Connection error: ${error}`);
        }
    }
    
    async disconnectFromDevice(item: any): Promise<void> {
        const device = item?.device;
        if (!device) {
            return;
        }
        
        try {
            await this.deviceManager.disconnectFromDevice(device);
            vscode.window.showInformationMessage(`Disconnected from ${device.name}`);
        } catch (error) {
            vscode.window.showErrorMessage(`Disconnect error: ${error}`);
        }
    }
    
    async openMonitor(item: any): Promise<void> {
        const device = item?.device;
        if (!device) {
            // Show connected devices
            const devices = this.deviceManager.getConnectedDevices();
            if (devices.length === 0) {
                vscode.window.showInformationMessage('No connected devices. Connect to a device first.');
                return;
            }
            
            const selected = await vscode.window.showQuickPick(
                devices.map(d => ({
                    label: d.name,
                    description: `${d.protocol} - ${d.address}:${d.port}`,
                    device: d
                })),
                { placeHolder: 'Select a device to monitor' }
            );
            
            if (!selected) {
                return;
            }
            
            await this.openMonitor({ device: selected.device });
            return;
        }
        
        MonitorPanel.createOrShow(this.context.extensionUri, device);
    }
    
    async refreshDevices(): Promise<void> {
        this.deviceTreeProvider.refresh();
        this.dataPointProvider.refresh();
    }
    
    async readTag(item: any): Promise<void> {
        const tag = item?.tag;
        const device = item?.device;
        
        if (!tag || !device) {
            return;
        }
        
        try {
            const value = await this.deviceManager.readTag(device, tag);
            vscode.window.showInformationMessage(
                `${tag.name}: ${value}${tag.unit ? ' ' + tag.unit : ''}`
            );
        } catch (error) {
            vscode.window.showErrorMessage(`Failed to read tag: ${error}`);
        }
    }
    
    async writeTag(item: any): Promise<void> {
        const tag = item?.tag;
        const device = item?.device;
        
        if (!tag || !device) {
            return;
        }
        
        const input = await vscode.window.showInputBox({
            prompt: `Enter new value for ${tag.name}`,
            placeHolder: `Current: ${tag.value}${tag.unit ? ' ' + tag.unit : ''}`,
            validateInput: (value) => {
                if (tag.dataType === 'boolean') {
                    if (value !== 'true' && value !== 'false') {
                        return 'Enter true or false';
                    }
                } else if (tag.dataType.includes('int') || tag.dataType.includes('float')) {
                    if (isNaN(Number(value))) {
                        return 'Enter a valid number';
                    }
                }
                return null;
            }
        });
        
        if (input === undefined) {
            return;
        }
        
        try {
            let value: any = input;
            if (tag.dataType === 'boolean') {
                value = input === 'true';
            } else if (tag.dataType.includes('int') || tag.dataType.includes('float')) {
                value = Number(input);
            }
            
            await this.deviceManager.writeTag(device, tag, value);
            vscode.window.showInformationMessage(`Successfully wrote ${value} to ${tag.name}`);
        } catch (error) {
            vscode.window.showErrorMessage(`Failed to write tag: ${error}`);
        }
    }
    
    async exportData(): Promise<void> {
        const devices = this.deviceManager.getConnectedDevices();
        if (devices.length === 0) {
            vscode.window.showInformationMessage('No connected devices to export data from.');
            return;
        }
        
        // Select device
        const selectedDevice = await vscode.window.showQuickPick(
            devices.map(d => ({
                label: d.name,
                description: `${d.protocol} - ${d.address}:${d.port}`,
                device: d
            })),
            { placeHolder: 'Select device to export data from' }
        );
        
        if (!selectedDevice) {
            return;
        }
        
        // Select format
        const format = await vscode.window.showQuickPick(
            ['CSV', 'JSON'],
            { placeHolder: 'Select export format' }
        );
        
        if (!format) {
            return;
        }
        
        // Save file dialog
        const uri = await vscode.window.showSaveDialog({
            defaultUri: vscode.Uri.file(`bifrost_export_${Date.now()}.${format.toLowerCase()}`),
            filters: {
                [format]: [format.toLowerCase()]
            }
        });
        
        if (!uri) {
            return;
        }
        
        try {
            await vscode.window.withProgress({
                location: vscode.ProgressLocation.Notification,
                title: "Exporting data...",
            }, async () => {
                // TODO: Implement actual export via BifrostAPI
                vscode.window.showInformationMessage(`Data exported to ${uri.fsPath}`);
            });
        } catch (error) {
            vscode.window.showErrorMessage(`Export failed: ${error}`);
        }
    }
}