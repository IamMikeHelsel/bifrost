import * as vscode from 'vscode';
import { Device, Tag } from '../services/deviceManager';
export declare class MonitorPanel {
    static currentPanel: MonitorPanel | undefined;
    private readonly _panel;
    private readonly _extensionUri;
    private _disposables;
    private _device;
    private _dataBuffer;
    static createOrShow(extensionUri: vscode.Uri, device: Device): void;
    private constructor();
    updateDevice(device: Device): void;
    updateData(tag: Tag): void;
    private exportData;
    private writeTag;
    dispose(): void;
    private _update;
    private _getHtmlForWebview;
}
