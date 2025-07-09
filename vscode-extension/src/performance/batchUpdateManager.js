
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
exports.Debouncer = exports.BatchUpdateManager = void 0;
const vscode = __importStar(require("vscode"));
class BatchUpdateManager {
    pendingUpdates = new Map();
    updateTimer;
    batchDelay = 16; // ~60 FPS
    maxBatchSize = 1000;
    onUpdate = new vscode.EventEmitter();
    onBatchUpdate = this.onUpdate.event;
    scheduleUpdate(update) {
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
    forceFlush() {
        if (this.updateTimer) {
            clearTimeout(this.updateTimer);
            this.updateTimer = undefined;
        }
        this.flushUpdates();
    }
    flushUpdates() {
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
    dispose() {
        if (this.updateTimer) {
            clearTimeout(this.updateTimer);
        }
        this.onUpdate.dispose();
    }
}
exports.BatchUpdateManager = BatchUpdateManager;
// Debounced function wrapper for expensive operations
class Debouncer {
    timers = new Map();
    debounce(key, fn, delay = 100) {
        return (...args) => {
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
    dispose() {
        this.timers.forEach(timer => clearTimeout(timer));
        this.timers.clear();
    }
}
exports.Debouncer = Debouncer;
//# sourceMappingURL=batchUpdateManager.js.map