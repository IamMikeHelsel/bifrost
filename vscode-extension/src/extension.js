
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
exports.activate = activate;
exports.deactivate = deactivate;
const vscode = __importStar(require("vscode"));
const deviceTreeProvider_1 = require("./providers/deviceTreeProvider");
const dataPointProvider_1 = require("./providers/dataPointProvider");
const diagnosticsProvider_1 = require("./providers/diagnosticsProvider");
const deviceManager_1 = require("./services/deviceManager");
const bifrostAPI_1 = require("./api/bifrostAPI");
const commandHandler_1 = require("./commands/commandHandler");
function activate(context) {
    console.log('Bifrost Industrial IoT extension is now active');
    // Initialize services
    const bifrostAPI = new bifrostAPI_1.BifrostAPI();
    const deviceManager = new deviceManager_1.DeviceManager(bifrostAPI);
    // Initialize providers
    const deviceTreeProvider = new deviceTreeProvider_1.DeviceTreeProvider(deviceManager);
    const dataPointProvider = new dataPointProvider_1.DataPointProvider(deviceManager);
    const diagnosticsProvider = new diagnosticsProvider_1.DiagnosticsProvider(deviceManager);
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
    const commandHandler = new commandHandler_1.CommandHandler(context, deviceManager, deviceTreeProvider, dataPointProvider);
    // Register commands
    context.subscriptions.push(vscode.commands.registerCommand('bifrost.discover', () => commandHandler.discoverDevices()), vscode.commands.registerCommand('bifrost.connect', (device) => commandHandler.connectToDevice(device)), vscode.commands.registerCommand('bifrost.disconnect', (device) => commandHandler.disconnectFromDevice(device)), vscode.commands.registerCommand('bifrost.monitor', (device) => commandHandler.openMonitor(device)), vscode.commands.registerCommand('bifrost.refreshDevices', () => commandHandler.refreshDevices()), vscode.commands.registerCommand('bifrost.readTag', (tag) => commandHandler.readTag(tag)), vscode.commands.registerCommand('bifrost.writeTag', (tag) => commandHandler.writeTag(tag)), vscode.commands.registerCommand('bifrost.exportData', () => commandHandler.exportData()), vscode.commands.registerCommand('bifrost.enableTypescriptGo', () => commandHandler.enableTypescriptGo()), vscode.commands.registerCommand('bifrost.benchmarkPerformance', () => commandHandler.benchmarkPerformance()));
    // Status bar items
    const statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    statusBarItem.command = 'bifrost.monitor';
    context.subscriptions.push(statusBarItem);
    // Update status bar based on connections
    deviceManager.onConnectionChange((connected) => {
        if (connected > 0) {
            statusBarItem.text = `$(plug) Bifrost: ${connected} connected`;
            statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.prominentBackground');
            statusBarItem.show();
        }
        else {
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
            }
            else if (data.type === 'device_status') {
                deviceTreeProvider.updateDeviceStatus(data);
            }
        });
    }
}
function deactivate() {
    console.log('Bifrost Industrial IoT extension is now deactivated');
    // Clean up API connections
    const bifrostAPI = new bifrostAPI_1.BifrostAPI();
    bifrostAPI.dispose();
}
//# sourceMappingURL=extension.js.map