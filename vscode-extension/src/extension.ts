import * as vscode from 'vscode';
import { DeviceTreeProvider } from './providers/deviceTreeProvider';
import { DataPointProvider } from './providers/dataPointProvider';
import { DiagnosticsProvider } from './providers/diagnosticsProvider';
import { MonitorPanel } from './panels/monitorPanel';
import { LadderLogicPanel } from './panels/ladderLogicPanel';
import { DeviceManager } from './services/deviceManager';
import { BifrostAPI } from './api/bifrostAPI';
import { CommandHandler } from './commands/commandHandler';

export function activate(context: vscode.ExtensionContext) {
    console.log('Bifrost Industrial IoT extension is now active');

    // Initialize services
    const bifrostAPI = new BifrostAPI();
    const deviceManager = new DeviceManager(bifrostAPI);
    
    // Initialize providers
    const deviceTreeProvider = new DeviceTreeProvider(deviceManager);
    const dataPointProvider = new DataPointProvider(deviceManager);
    const diagnosticsProvider = new DiagnosticsProvider(deviceManager);
    
    // Register tree data providers
    vscode.window.createTreeView('bifrostDevices', {
        treeDataProvider: deviceTreeProvider,
        showCollapseAll: true
    });
    
    vscode.window.createTreeView('bifrostData', {
        treeDataProvider: dataPointProvider,
        showCollapseAll: true
    });
    
    vscode.window.createTreeView('bifrostDiagnostics', {
        treeDataProvider: diagnosticsProvider
    });
    
    // Initialize command handler
    const commandHandler = new CommandHandler(
        context,
        deviceManager,
        deviceTreeProvider,
        dataPointProvider
    );
    
    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('bifrost.discover', () => 
            commandHandler.discoverDevices()
        ),
        
        vscode.commands.registerCommand('bifrost.connect', (device) => 
            commandHandler.connectToDevice(device)
        ),
        
        vscode.commands.registerCommand('bifrost.disconnect', (device) => 
            commandHandler.disconnectFromDevice(device)
        ),
        
        vscode.commands.registerCommand('bifrost.monitor', (device) => 
            commandHandler.openMonitor(device)
        ),
        
        vscode.commands.registerCommand('bifrost.ladderLogic', (device) => 
            commandHandler.openLadderLogic(device)
        ),
        
        vscode.commands.registerCommand('bifrost.refreshDevices', () => 
            commandHandler.refreshDevices()
        ),
        
        vscode.commands.registerCommand('bifrost.readTag', (tag) => 
            commandHandler.readTag(tag)
        ),
        
        vscode.commands.registerCommand('bifrost.writeTag', (tag) => 
            commandHandler.writeTag(tag)
        ),
        
        vscode.commands.registerCommand('bifrost.exportData', () => 
            commandHandler.exportData()
        ),
        
        vscode.commands.registerCommand('bifrost.enableTypescriptGo', () => 
            commandHandler.enableTypescriptGo()
        ),
        
        vscode.commands.registerCommand('bifrost.benchmarkPerformance', () => 
            commandHandler.benchmarkPerformance()
        ),
        
        vscode.commands.registerCommand('bifrost.viewLadderLogic', () => 
            commandHandler.openLadderLogic(null)
        )
    );
    
    // Status bar items
    const statusBarItem = vscode.window.createStatusBarItem(
        vscode.StatusBarAlignment.Right,
        100
    );
    statusBarItem.command = 'bifrost.monitor';
    context.subscriptions.push(statusBarItem);
    
    // Update status bar based on connections
    deviceManager.onConnectionChange((connected) => {
        if (connected > 0) {
            statusBarItem.text = `$(plug) Bifrost: ${connected} connected`;
            statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.prominentBackground');
            statusBarItem.show();
        } else {
            statusBarItem.text = '$(debug-disconnect) Bifrost: No connections';
            statusBarItem.backgroundColor = undefined;
            statusBarItem.show();
        }
    });
    
    // Initialize with no connections
    statusBarItem.text = '$(debug-disconnect) Bifrost: Ready';
    statusBarItem.show();
    
    // Start real-time data streaming if enabled
    const config = vscode.workspace.getConfiguration('bifrost');
    if (config.get('realtime.enabled', true)) {
        bifrostAPI.startRealTimeDataStream((data) => {
            // Update device tree and data providers with real-time data
            if (data.type === 'device_data') {
                dataPointProvider.updateRealTimeData(data);
            } else if (data.type === 'device_status') {
                deviceTreeProvider.updateDeviceStatus(data);
            }
        });
    }
}

export function deactivate() {
    console.log('Bifrost Industrial IoT extension is now deactivated');
    
    // Clean up API connections
    const bifrostAPI = new BifrostAPI();
    bifrostAPI.dispose();
}