import { Device, Tag, DeviceStats } from '../services/deviceManager';
export declare class BifrostAPI {
    private readonly gatewayUrl;
    private websocket?;
    private reconnectTimer?;
    private readonly maxReconnectAttempts = 5;
    private reconnectAttempts;
    constructor();
    private getGatewayUrl;
    private getWebSocketUrl;
    discoverDevices(): Promise<Device[]>;
    private getMockDevices;
    connectToDevice(device: Device): Promise<boolean>;
    disconnectFromDevice(device: Device): Promise<void>;
    getDeviceTags(deviceId: string): Promise<Tag[]>;
    private getMockTags;
    readTag(deviceId: string, address: string): Promise<any>;
    writeTag(deviceId: string, address: string, value: any): Promise<void>;
    getDeviceStats(deviceId: string): Promise<DeviceStats>;
    exportData(deviceId: string, tags: string[], format: 'csv' | 'json'): Promise<string>;
    // WebSocket real-time data streaming
    startRealTimeDataStream(onDataUpdate: (data: any) => void): void;
    stopRealTimeDataStream(): void;
    private connectWebSocket;
    // Check if Go gateway is available
    isGatewayAvailable(): Promise<boolean>;
    // Cleanup method
    dispose(): void;
}
