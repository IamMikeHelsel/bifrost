import * as vscode from 'vscode';
export interface Update {
    id: string;
    type: 'device' | 'tag' | 'stats';
    data: any;
    timestamp: number;
}
export declare class BatchUpdateManager {
    private pendingUpdates;
    private updateTimer?;
    private readonly batchDelay = 16; // ~60 FPS
    private readonly maxBatchSize = 1000;
    private onUpdate;
    readonly onBatchUpdate: vscode.Event<Update[]>;
    scheduleUpdate(update: Update): void;
    forceFlush(): void;
    private flushUpdates;
    dispose(): void;
}
// Debounced function wrapper for expensive operations
export declare class Debouncer {
    private timers;
    debounce<T extends (...args: any[]) => any>(key: string, fn: T, delay?: number): (...args: Parameters<T>) => void;
    dispose(): void;
}
