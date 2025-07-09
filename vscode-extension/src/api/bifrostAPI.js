
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
exports.BifrostAPI = void 0;
const vscode = __importStar(require("vscode"));
const deviceManager_1 = require("../services/deviceManager");
class BifrostAPI {
    gatewayUrl;
    websocket;
    reconnectTimer;
    maxReconnectAttempts = 5;
    reconnectAttempts = 0;
    constructor() {
        // Connect to Go gateway instead of Python backend
        this.gatewayUrl = this.getGatewayUrl();
    }
    getGatewayUrl() {
        const config = vscode.workspace.getConfiguration('bifrost');
        const host = config.get('gateway.host', 'localhost');
        const port = config.get('gateway.port', 8080);
        return `http://${host}:${port}`;
    }
    getWebSocketUrl() {
        return this.gatewayUrl.replace('http://', 'ws://').replace('https://', 'wss://') + '/ws';
    }
    async discoverDevices() {
        try {
            const response = await fetch(`${this.gatewayUrl}/api/devices/discover`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    protocols: ['modbus-tcp', 'opcua'],
                    timeout: 5000,
                    network_ranges: ['192.168.1.0/24', '10.0.0.0/24']
                })
            });
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            const devices = await response.json();
            // Convert Go gateway device format to VS Code extension format
            return devices.map((device) => ({
                id: device.id,
                name: device.name,
                protocol: device.protocol,
                address: device.address,
                port: device.port,
                status: device.connected ? deviceManager_1.DeviceStatus.Connected : deviceManager_1.DeviceStatus.Disconnected
            }));
        }
        catch (error) {
            console.error('Device discovery failed:', error);
            // Fallback to mock data for development
            return this.getMockDevices();
        }
    }
    getMockDevices() {
        return [
            {
                id: 'modbus-sim-1',
                name: 'Modbus Simulator',
                protocol: 'modbus-tcp',
                address: 'localhost',
                port: 502,
                status: deviceManager_1.DeviceStatus.Disconnected
            },
            {
                id: 'opcua-sim-1',
                name: 'OPC UA Simulator',
                protocol: 'opcua',
                address: 'localhost',
                port: 4840,
                status: deviceManager_1.DeviceStatus.Disconnected
            }
        ];
    }
    async connectToDevice(device) {
        try {
            const response = await fetch(`${this.gatewayUrl}/api/devices`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    id: device.id,
                    name: device.name,
                    protocol: device.protocol,
                    address: device.address,
                    port: device.port,
                    config: {}
                })
            });
            return response.ok;
        }
        catch (error) {
            console.error('Failed to connect to device:', error);
            return false;
        }
    }
    async disconnectFromDevice(device) {
        try {
            await fetch(`${this.gatewayUrl}/api/devices/${device.id}`, {
                method: 'DELETE'
            });
        }
        catch (error) {
            console.error('Failed to disconnect from device:', error);
        }
    }
    async getDeviceTags(deviceId) {
        try {
            const response = await fetch(`${this.gatewayUrl}/api/devices/${deviceId}/tags`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            const tagsData = await response.json();
            // Convert Go gateway tag format to VS Code extension format
            return Object.values(tagsData).map((tag) => ({
                id: tag.id,
                name: tag.name,
                address: tag.address,
                dataType: tag.data_type,
                writable: tag.writable,
                unit: tag.unit,
                description: tag.description
            }));
        }
        catch (error) {
            console.error('Failed to get device tags:', error);
            // Fallback to mock data
            return this.getMockTags(deviceId);
        }
    }
    getMockTags(deviceId) {
        if (deviceId.includes('modbus')) {
            return [
                {
                    id: 'temp-1',
                    name: 'Temperature Sensor 1',
                    address: '40001',
                    dataType: 'float32',
                    writable: false,
                    unit: '°C',
                    description: 'Primary temperature sensor'
                },
                {
                    id: 'pressure-1',
                    name: 'Pressure Sensor 1',
                    address: '40011',
                    dataType: 'float32',
                    writable: false,
                    unit: 'bar',
                    description: 'System pressure'
                },
                {
                    id: 'setpoint-1',
                    name: 'Temperature Setpoint',
                    address: '40031',
                    dataType: 'float32',
                    writable: true,
                    unit: '°C',
                    description: 'Temperature control setpoint'
                }
            ];
        }
        else if (deviceId.includes('opcua')) {
            return [
                {
                    id: 'ns2-temp-1',
                    name: 'Temperature Sensor 1',
                    address: 'ns=2;s=TempSensor1',
                    dataType: 'float',
                    writable: false,
                    unit: '°C'
                },
                {
                    id: 'ns2-pressure-1',
                    name: 'Pressure Sensor 1',
                    address: 'ns=2;s=PressureSensor1',
                    dataType: 'float',
                    writable: false,
                    unit: 'bar'
                }
            ];
        }
        return [];
    }
    async readTag(deviceId, address) {
        try {
            const response = await fetch(`${this.gatewayUrl}/api/tags/read`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    device_id: deviceId,
                    tag_address: address
                })
            });
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            const result = await response.json();
            return result.value;
        }
        catch (error) {
            console.error('Failed to read tag:', error);
            // Fallback to mock data
            const baseValue = Math.random() * 100;
            const variation = (Math.random() - 0.5) * 10;
            return baseValue + variation;
        }
    }
    async writeTag(deviceId, address, value) {
        try {
            const response = await fetch(`${this.gatewayUrl}/api/tags/write`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    device_id: deviceId,
                    tag_address: address,
                    value: value
                })
            });
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
        }
        catch (error) {
            console.error('Failed to write tag:', error);
            throw error;
        }
    }
    async getDeviceStats(deviceId) {
        try {
            const response = await fetch(`${this.gatewayUrl}/api/devices/${deviceId}/stats`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            const stats = await response.json();
            return {
                totalRequests: stats.requests_total,
                successfulRequests: stats.requests_successful,
                failedRequests: stats.requests_failed,
                successRate: Math.round((stats.requests_successful / stats.requests_total) * 100),
                averageResponseTime: stats.avg_response_time
            };
        }
        catch (error) {
            console.error('Failed to get device stats:', error);
            // Fallback to mock data
            const total = Math.floor(Math.random() * 10000) + 1000;
            const failed = Math.floor(Math.random() * 100);
            const successful = total - failed;
            return {
                totalRequests: total,
                successfulRequests: successful,
                failedRequests: failed,
                successRate: Math.round((successful / total) * 100),
                averageResponseTime: Math.random() * 50 + 10
            };
        }
    }
    async exportData(deviceId, tags, format) {
        // Mock export functionality
        const data = [];
        const timestamp = new Date().toISOString();
        for (const tagId of tags) {
            const value = await this.readTag(deviceId, tagId);
            data.push({
                timestamp,
                tag: tagId,
                value
            });
        }
        if (format === 'json') {
            return JSON.stringify(data, null, 2);
        }
        else {
            // CSV format
            let csv = 'timestamp,tag,value\n';
            for (const row of data) {
                csv += `${row.timestamp},${row.tag},${row.value}\n`;
            }
            return csv;
        }
    }
    // WebSocket real-time data streaming
    startRealTimeDataStream(onDataUpdate) {
        this.connectWebSocket(onDataUpdate);
    }
    stopRealTimeDataStream() {
        if (this.websocket) {
            this.websocket.close();
            this.websocket = undefined;
        }
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = undefined;
        }
    }
    connectWebSocket(onDataUpdate) {
        try {
            const wsUrl = this.getWebSocketUrl();
            // Use dynamic require to avoid TypeScript issues
            const ws = require('ws');
            this.websocket = new ws(wsUrl);
            if (this.websocket) {
                this.websocket.onopen = () => {
                    console.log('WebSocket connected to Go gateway');
                    this.reconnectAttempts = 0;
                    // Subscribe to real-time data updates
                    this.websocket?.send(JSON.stringify({
                        type: 'subscribe',
                        topics: ['device_data', 'device_status']
                    }));
                };
                this.websocket.onmessage = (event) => {
                    try {
                        const data = JSON.parse(event.data);
                        onDataUpdate(data);
                    }
                    catch (error) {
                        console.error('Failed to parse WebSocket message:', error);
                    }
                };
                this.websocket.onclose = () => {
                    console.log('WebSocket disconnected');
                    this.websocket = undefined;
                    // Attempt to reconnect
                    if (this.reconnectAttempts < this.maxReconnectAttempts) {
                        this.reconnectAttempts++;
                        const delay = Math.min(1000 * 2 ** this.reconnectAttempts, 30000);
                        this.reconnectTimer = setTimeout(() => {
                            console.log(`Attempting WebSocket reconnection ${this.reconnectAttempts}/${this.maxReconnectAttempts}`);
                            this.connectWebSocket(onDataUpdate);
                        }, delay);
                    }
                };
                this.websocket.onerror = (error) => {
                    console.error('WebSocket error:', error);
                };
            }
        }
        catch (error) {
            console.error('Failed to create WebSocket connection:', error);
        }
    }
    // Check if Go gateway is available
    async isGatewayAvailable() {
        try {
            const response = await fetch(`${this.gatewayUrl}/health`, {
                method: 'GET',
                signal: AbortSignal.timeout(5000) // 5 second timeout
            });
            return response.ok;
        }
        catch (error) {
            return false;
        }
    }
    // Cleanup method
    dispose() {
        this.stopRealTimeDataStream();
    }
}
exports.BifrostAPI = BifrostAPI;
//# sourceMappingURL=bifrostAPI.js.map