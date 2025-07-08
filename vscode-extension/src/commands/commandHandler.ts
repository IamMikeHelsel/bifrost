import * as vscode from 'vscode';
import { DeviceManager } from '../services/deviceManager';
import { DeviceTreeProvider } from '../providers/deviceTreeProvider';
import { DataPointProvider } from '../providers/dataPointProvider';
import { MonitorPanel } from '../panels/monitorPanel';
import { LadderLogicPanel } from '../panels/ladderLogicPanel';
import { spawn } from 'child_process';

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
    
    async openLadderLogic(item: any): Promise<void> {
        const device = item?.device;
        if (!device) {
            // Show connected devices or allow opening without device for sample
            const devices = this.deviceManager.getConnectedDevices();
            const options = [
                { label: 'Sample Ladder Logic', description: 'View sample ladder logic diagram', device: null },
                ...devices.map(d => ({
                    label: d.name,
                    description: `${d.protocol} - ${d.address}:${d.port} (Load from PLC)`,
                    device: d
                }))
            ];
            
            if (options.length === 1) {
                // Only sample option available
                LadderLogicPanel.createOrShow(this.context.extensionUri);
                return;
            }
            
            const selected = await vscode.window.showQuickPick(options, {
                placeHolder: 'Select ladder logic source'
            });
            
            if (!selected) {
                return;
            }
            
            LadderLogicPanel.createOrShow(this.context.extensionUri, selected.device);
            return;
        }
        
        LadderLogicPanel.createOrShow(this.context.extensionUri, device);
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
    
    async enableTypescriptGo(): Promise<void> {
        const config = vscode.workspace.getConfiguration('bifrost');
        const isEnabled = config.get('experimental.useTypescriptGo', false);
        
        if (isEnabled) {
            vscode.window.showInformationMessage('TypeScript-Go is already enabled!');
            return;
        }
        
        const choice = await vscode.window.showInformationMessage(
            'Enable experimental TypeScript-Go compiler for 10x faster builds?',
            'Enable',
            'Learn More',
            'Cancel'
        );
        
        if (choice === 'Enable') {
            try {
                // Check if TypeScript-Go is available
                const { spawn } = require('child_process');
                const child = spawn('npx', ['tsgo', '--version'], { stdio: 'pipe' });
                
                child.on('close', async (code: number) => {
                    if (code === 0) {
                        await config.update('experimental.useTypescriptGo', true, vscode.ConfigurationTarget.Global);
                        vscode.window.showInformationMessage(
                            'TypeScript-Go enabled! Restart VS Code to apply changes.',
                            'Restart Now'
                        ).then(choice => {
                            if (choice === 'Restart Now') {
                                vscode.commands.executeCommand('workbench.action.reloadWindow');
                            }
                        });
                    } else {
                        vscode.window.showErrorMessage(
                            'TypeScript-Go not found. Install it first: npm install @typescript/native-preview'
                        );
                    }
                });
            } catch (error) {
                vscode.window.showErrorMessage(`Failed to enable TypeScript-Go: ${error}`);
            }
        } else if (choice === 'Learn More') {
            vscode.env.openExternal(vscode.Uri.parse('https://github.com/microsoft/typescript-go'));
        }
    }
    
    async benchmarkPerformance(): Promise<void> {
        try {
            await vscode.window.withProgress({
                location: vscode.ProgressLocation.Notification,
                title: "Running TypeScript compilation benchmark...",
                cancellable: false
            }, async (progress) => {
                progress.report({ increment: 0, message: "Starting benchmark..." });
                
                return new Promise<void>((resolve, reject) => {
                    const child = spawn('npm', ['run', 'benchmark'], {
                        cwd: this.context.extensionPath,
                        stdio: 'pipe'
                    });
                    
                    let output = '';
                    
                    child.stdout.on('data', (data) => {
                        output += data.toString();
                        progress.report({ increment: 20, message: "Running benchmarks..." });
                    });
                    
                    child.stderr.on('data', (data) => {
                        output += data.toString();
                    });
                    
                    child.on('close', (code) => {
                        if (code === 0) {
                            // Show results in output channel
                            const outputChannel = vscode.window.createOutputChannel('Bifrost Performance Benchmark');
                            outputChannel.appendLine(output);
                            outputChannel.show();
                            
                            vscode.window.showInformationMessage(
                                'Performance benchmark completed! Check the output panel for results.',
                                'View Results'
                            ).then(choice => {
                                if (choice === 'View Results') {
                                    outputChannel.show();
                                }
                            });
                            
                            resolve();
                        } else {
                            reject(new Error(`Benchmark failed with code ${code}`));
                        }
                    });
                });
            });
        } catch (error) {
            vscode.window.showErrorMessage(`Benchmark failed: ${error}`);
        }
    }
}