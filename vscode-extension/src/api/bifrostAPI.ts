import * as vscode from 'vscode';
import { Device, DeviceStatus, Tag, DeviceStats } from '../services/deviceManager';

// WebSocket types for Node.js environment
declare global {
    const WebSocket: {
        new(url: string): {
            onopen: ((event: any) => void) | null;
            onmessage: ((event: { data: string }) => void) | null;
            onclose: (() => void) | null;
            onerror: ((error: any) => void) | null;
            send(data: string): void;
            close(): void;
        };
    };
}

export class BifrostAPI {
    private readonly gatewayUrl: string;
    private websocket?: WebSocket;
    private reconnectTimer?: NodeJS.Timeout;
    private readonly maxReconnectAttempts = 5;
    private reconnectAttempts = 0;
    
    constructor() {
        // Connect to Go gateway instead of Python backend
        this.gatewayUrl = this.getGatewayUrl();
    }
    
    private getGatewayUrl(): string {
        const config = vscode.workspace.getConfiguration('bifrost');
        const host = config.get('gateway.host', 'localhost');
        const port = config.get('gateway.port', 8080);
        return `http://${host}:${port}`;
    }
    
    private getWebSocketUrl(): string {
        return this.gatewayUrl.replace('http://', 'ws://').replace('https://', 'wss://') + '/ws';
    }
    
    async discoverDevices(): Promise<Device[]> {
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
            
            const devices = await response.json() as any[];
            
            // Convert Go gateway device format to VS Code extension format
            return devices.map((device: any) => ({
                id: device.id,
                name: device.name,
                protocol: device.protocol,
                address: device.address,
                port: device.port,
                status: device.connected ? DeviceStatus.Connected : DeviceStatus.Disconnected
            }));
            
        } catch (error) {
            console.error('Device discovery failed:', error);
            // Fallback to mock data for development
            return this.getMockDevices();
        }
    }
    
    private getMockDevices(): Device[] {
        return [
            {
                id: 'modbus-sim-1',
                name: 'Modbus Simulator',
                protocol: 'modbus-tcp',
                address: 'localhost',
                port: 502,
                status: DeviceStatus.Disconnected
            },
            {
                id: 'opcua-sim-1',
                name: 'OPC UA Simulator',
                protocol: 'opcua',
                address: 'localhost',
                port: 4840,
                status: DeviceStatus.Disconnected
            }
        ];
    }
    
    async connectToDevice(device: Device): Promise<boolean> {
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
        } catch (error) {
            console.error('Failed to connect to device:', error);
            return false;
        }
    }
    
    async disconnectFromDevice(device: Device): Promise<void> {
        try {
            await fetch(`${this.gatewayUrl}/api/devices/${device.id}`, {
                method: 'DELETE'
            });
        } catch (error) {
            console.error('Failed to disconnect from device:', error);
        }
    }
    
    async getDeviceTags(deviceId: string): Promise<Tag[]> {
        try {
            const response = await fetch(`${this.gatewayUrl}/api/devices/${deviceId}/tags`);
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const tagsData = await response.json() as Record<string, any>;
            
            // Convert Go gateway tag format to VS Code extension format
            return Object.values(tagsData).map((tag: any) => ({
                id: tag.id,
                name: tag.name,
                address: tag.address,
                dataType: tag.data_type,
                writable: tag.writable,
                unit: tag.unit,
                description: tag.description
            }));
            
        } catch (error) {
            console.error('Failed to get device tags:', error);
            // Fallback to mock data
            return this.getMockTags(deviceId);
        }
    }
    
    private getMockTags(deviceId: string): Tag[] {
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
        } else if (deviceId.includes('opcua')) {
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
    
    async readTag(deviceId: string, address: string): Promise<any> {
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
            
            const result = await response.json() as { value: any };
            return result.value;
            
        } catch (error) {
            console.error('Failed to read tag:', error);
            // Fallback to mock data
            const baseValue = Math.random() * 100;
            const variation = (Math.random() - 0.5) * 10;
            return baseValue + variation;
        }
    }
    
    async writeTag(deviceId: string, address: string, value: any): Promise<void> {
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
            
        } catch (error) {
            console.error('Failed to write tag:', error);
            throw error;
        }
    }
    
    async getDeviceStats(deviceId: string): Promise<DeviceStats> {
        try {
            const response = await fetch(`${this.gatewayUrl}/api/devices/${deviceId}/stats`);
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const stats = await response.json() as {
                requests_total: number;
                requests_successful: number;
                requests_failed: number;
                avg_response_time: number;
            };
            
            return {
                totalRequests: stats.requests_total,
                successfulRequests: stats.requests_successful,
                failedRequests: stats.requests_failed,
                successRate: Math.round((stats.requests_successful / stats.requests_total) * 100),
                averageResponseTime: stats.avg_response_time
            };
            
        } catch (error) {
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
    
    async exportData(deviceId: string, tags: string[], format: 'csv' | 'json'): Promise<string> {
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
        } else {
            // CSV format
            let csv = 'timestamp,tag,value\n';
            for (const row of data) {
                csv += `${row.timestamp},${row.tag},${row.value}\n`;
            }
            return csv;
        }
    }
    
    // WebSocket real-time data streaming
    startRealTimeDataStream(onDataUpdate: (data: any) => void): void {
        this.connectWebSocket(onDataUpdate);
    }
    
    stopRealTimeDataStream(): void {
        if (this.websocket) {
            this.websocket.close();
            this.websocket = undefined;
        }
        
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = undefined;
        }
    }
    
    private connectWebSocket(onDataUpdate: (data: any) => void): void {
        try {
            const wsUrl = this.getWebSocketUrl();
            this.websocket = new WebSocket(wsUrl);
            
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
                } catch (error) {
                    console.error('Failed to parse WebSocket message:', error);
                }
            };
            
            this.websocket.onclose = () => {
                console.log('WebSocket disconnected');
                this.websocket = undefined;
                
                // Attempt to reconnect
                if (this.reconnectAttempts < this.maxReconnectAttempts) {
                    this.reconnectAttempts++;
                    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
                    
                    this.reconnectTimer = setTimeout(() => {
                        console.log(`Attempting WebSocket reconnection ${this.reconnectAttempts}/${this.maxReconnectAttempts}`);
                        this.connectWebSocket(onDataUpdate);
                    }, delay);
                }
            };
            
            this.websocket.onerror = (error) => {
                console.error('WebSocket error:', error);
            };
            
        } catch (error) {
            console.error('Failed to create WebSocket connection:', error);
        }
    }
    
    // Check if Go gateway is available
    async isGatewayAvailable(): Promise<boolean> {
        try {
            const response = await fetch(`${this.gatewayUrl}/health`, {
                method: 'GET',
                signal: AbortSignal.timeout(5000) // 5 second timeout
            });
            return response.ok;
        } catch (error) {
            return false;
        }
    }
    
    // Cleanup method
    dispose(): void {
        this.stopRealTimeDataStream();
    }
}