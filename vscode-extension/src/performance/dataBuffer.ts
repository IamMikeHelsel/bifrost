/**
 * High-performance ring buffer for time-series data
 * Optimized for real-time industrial data streams
 */
export class RingBuffer<T> {
    private buffer: T[];
    private head = 0;
    private size = 0;
    private readonly capacity: number;
    
    constructor(capacity: number) {
        this.capacity = capacity;
        this.buffer = new Array(capacity);
    }
    
    push(item: T): void {
        this.buffer[this.head] = item;
        this.head = (this.head + 1) % this.capacity;
        this.size = Math.min(this.size + 1, this.capacity);
    }
    
    toArray(): T[] {
        if (this.size === 0) {
            return [];
        }
        
        if (this.size < this.capacity) {
            return this.buffer.slice(0, this.size);
        }
        
        // Buffer is full, need to reorder
        return [
            ...this.buffer.slice(this.head),
            ...this.buffer.slice(0, this.head)
        ];
    }
    
    get length(): number {
        return this.size;
    }
    
    get isFull(): boolean {
        return this.size === this.capacity;
    }
    
    clear(): void {
        this.head = 0;
        this.size = 0;
    }
}

/**
 * Memory-efficient storage for tag data using typed arrays
 */
export class TagDataBuffer {
    private values: Float32Array;
    private timestamps: Uint32Array;
    private pointer = 0;
    private isFull = false;
    
    constructor(private capacity: number) {
        this.values = new Float32Array(capacity);
        this.timestamps = new Uint32Array(capacity);
    }
    
    addDataPoint(value: number, timestamp: number): void {
        this.values[this.pointer] = value;
        this.timestamps[this.pointer] = Math.floor(timestamp / 1000); // Store as seconds
        
        this.pointer = (this.pointer + 1) % this.capacity;
        
        if (this.pointer === 0) {
            this.isFull = true;
        }
    }
    
    getChartData(): Array<{x: number, y: number}> {
        const size = this.isFull ? this.capacity : this.pointer;
        const result = new Array(size);
        
        if (this.isFull) {
            // Read from pointer position to end, then from start to pointer
            let resultIndex = 0;
            
            for (let i = this.pointer; i < this.capacity; i++) {
                result[resultIndex++] = {
                    x: this.timestamps[i] * 1000, // Convert back to milliseconds
                    y: this.values[i]
                };
            }
            
            for (let i = 0; i < this.pointer; i++) {
                result[resultIndex++] = {
                    x: this.timestamps[i] * 1000,
                    y: this.values[i]
                };
            }
        } else {
            // Simple case - just read from start to pointer
            for (let i = 0; i < this.pointer; i++) {
                result[i] = {
                    x: this.timestamps[i] * 1000,
                    y: this.values[i]
                };
            }
        }
        
        return result;
    }
    
    getLatestValue(): number | undefined {
        if (this.pointer === 0 && !this.isFull) {
            return undefined;
        }
        
        const lastIndex = this.pointer === 0 ? this.capacity - 1 : this.pointer - 1;
        return this.values[lastIndex];
    }
    
    clear(): void {
        this.pointer = 0;
        this.isFull = false;
    }
    
    get memoryUsage(): number {
        // Return memory usage in bytes
        return this.values.byteLength + this.timestamps.byteLength;
    }
}

/**
 * LRU Cache for device metadata
 */
export class LRUCache<K, V> {
    private cache = new Map<K, V>();
    private readonly maxSize: number;
    
    constructor(maxSize: number = 1000) {
        this.maxSize = maxSize;
    }
    
    get(key: K): V | undefined {
        const value = this.cache.get(key);
        if (value !== undefined) {
            // Move to end (most recently used)
            this.cache.delete(key);
            this.cache.set(key, value);
        }
        return value;
    }
    
    set(key: K, value: V): void {
        if (this.cache.has(key)) {
            // Update existing
            this.cache.delete(key);
        } else if (this.cache.size >= this.maxSize) {
            // Remove least recently used (first item)
            const firstKey = this.cache.keys().next().value;
            this.cache.delete(firstKey);
        }
        
        this.cache.set(key, value);
    }
    
    has(key: K): boolean {
        return this.cache.has(key);
    }
    
    delete(key: K): boolean {
        return this.cache.delete(key);
    }
    
    clear(): void {
        this.cache.clear();
    }
    
    get size(): number {
        return this.cache.size;
    }
}