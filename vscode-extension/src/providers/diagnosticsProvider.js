
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
exports.DiagnosticItem = exports.DiagnosticsProvider = void 0;
const vscode = __importStar(require("vscode"));
class DiagnosticsProvider {
    deviceManager;
    _onDidChangeTreeData = new vscode.EventEmitter();
    onDidChangeTreeData = this._onDidChangeTreeData.event;
    constructor(deviceManager) {
        this.deviceManager = deviceManager;
        // Refresh periodically
        setInterval(() => this.refresh(), 5000);
    }
    refresh() {
        this._onDidChangeTreeData.fire();
    }
    getTreeItem(element) {
        return element;
    }
    getChildren(element) {
        const items = [];
        // System status
        const connectedCount = this.deviceManager.getConnectedDevices().length;
        const totalCount = this.deviceManager.getAllDevices().length;
        const systemItem = new DiagnosticItem('System Status', vscode.TreeItemCollapsibleState.None, 'system');
        systemItem.description = `${connectedCount}/${totalCount} devices connected`;
        systemItem.iconPath = new vscode.ThemeIcon(connectedCount === totalCount ? 'check' :
            connectedCount > 0 ? 'warning' : 'error');
        items.push(systemItem);
        // Python/Bifrost status
        const pythonItem = new DiagnosticItem('Python Environment', vscode.TreeItemCollapsibleState.None, 'python');
        pythonItem.description = 'Python 3.13+ with Bifrost';
        pythonItem.iconPath = new vscode.ThemeIcon('check');
        items.push(pythonItem);
        // Performance metrics
        const devices = this.deviceManager.getConnectedDevices();
        let totalRequests = 0;
        let avgResponseTime = 0;
        let errorCount = 0;
        devices.forEach(device => {
            if (device.stats) {
                totalRequests += device.stats.totalRequests;
                avgResponseTime += device.stats.averageResponseTime;
                errorCount += device.stats.failedRequests;
            }
        });
        if (devices.length > 0) {
            avgResponseTime /= devices.length;
        }
        const perfItem = new DiagnosticItem('Performance', vscode.TreeItemCollapsibleState.None, 'performance');
        perfItem.description = `${avgResponseTime.toFixed(1)}ms avg response`;
        perfItem.iconPath = new vscode.ThemeIcon(avgResponseTime < 50 ? 'check' :
            avgResponseTime < 100 ? 'warning' : 'error');
        items.push(perfItem);
        // Error summary
        const errorItem = new DiagnosticItem('Errors', vscode.TreeItemCollapsibleState.None, 'errors');
        errorItem.description = errorCount > 0 ? `${errorCount} total errors` : 'No errors';
        errorItem.iconPath = new vscode.ThemeIcon(errorCount > 0 ? 'warning' : 'check');
        items.push(errorItem);
        // Recent issues
        const devicesWithErrors = devices.filter(d => d.lastError);
        if (devicesWithErrors.length > 0) {
            const issuesItem = new DiagnosticItem('Recent Issues', vscode.TreeItemCollapsibleState.None, 'issues');
            issuesItem.description = `${devicesWithErrors.length} devices with errors`;
            issuesItem.iconPath = new vscode.ThemeIcon('warning');
            items.push(issuesItem);
        }
        return Promise.resolve(items);
    }
}
exports.DiagnosticsProvider = DiagnosticsProvider;
class DiagnosticItem extends vscode.TreeItem {
    label;
    collapsibleState;
    contextValue;
    constructor(label, collapsibleState, contextValue) {
        super(label, collapsibleState);
        this.label = label;
        this.collapsibleState = collapsibleState;
        this.contextValue = contextValue;
    }
}
exports.DiagnosticItem = DiagnosticItem;
//# sourceMappingURL=diagnosticsProvider.js.map