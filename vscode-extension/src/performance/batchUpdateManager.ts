import * as vscode from 'vscode';

export interface Update {
    id: string;
    type: 'device' | 'tag' | 'stats';
    data: any;
    timestamp: number;
}

export class BatchUpdateManager {
    private pendingUpdates = new Map<string, Update>();
    private updateTimer?: NodeJS.Timer;
    private readonly batchDelay = 16; // ~60 FPS
    private readonly maxBatchSize = 1000;
    
    private onUpdate = new vscode.EventEmitter<Update[]>();
    readonly onBatchUpdate = this.onUpdate.event;
    
    scheduleUpdate(update: Update): void {
        // Always keep the latest update for each ID
        this.pendingUpdates.set(update.id, update);
        
        // If batch is full, flush immediately
        if (this.pendingUpdates.size >= this.maxBatchSize) {
            this.flushUpdates();
            return;
        }
        
        // Schedule batched update
        if (!this.updateTimer) {
            this.updateTimer = setTimeout(() => {
                this.flushUpdates();
            }, this.batchDelay);
        }
    }
    
    forceFlush(): void {
        if (this.updateTimer) {
            clearTimeout(this.updateTimer);
            this.updateTimer = undefined;
        }
        this.flushUpdates();
    }
    
    private flushUpdates(): void {
        if (this.pendingUpdates.size === 0) {
            return;
        }
        
        const updates = Array.from(this.pendingUpdates.values());
        this.pendingUpdates.clear();
        this.updateTimer = undefined;
        
        // Sort by timestamp for consistent ordering
        updates.sort((a, b) => a.timestamp - b.timestamp);
        
        // Fire batch update event
        this.onUpdate.fire(updates);
    }
    
    dispose(): void {
        if (this.updateTimer) {
            clearTimeout(this.updateTimer);
        }
        this.onUpdate.dispose();
    }
}

// Debounced function wrapper for expensive operations
export class Debouncer {
    private timers = new Map<string, NodeJS.Timer>();
    
    debounce<T extends (...args: any[]) => any>(
        key: string, 
        fn: T, 
        delay: number = 100
    ): (...args: Parameters<T>) => void {
        return (...args: Parameters<T>) => {
            const existingTimer = this.timers.get(key);
            if (existingTimer) {
                clearTimeout(existingTimer);
            }
            
            const timer = setTimeout(() => {
                this.timers.delete(key);
                fn(...args);
            }, delay);
            
            this.timers.set(key, timer);
        };
    }
    
    dispose(): void {
        this.timers.forEach(timer => clearTimeout(timer));
        this.timers.clear();
    }
}