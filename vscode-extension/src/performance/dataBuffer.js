
Object.defineProperty(exports, "__esModule", { value: true });
exports.LRUCache = exports.TagDataBuffer = exports.RingBuffer = void 0;
/**
 * High-performance ring buffer for time-series data
 * Optimized for real-time industrial data streams
 */
class RingBuffer {
    buffer;
    head = 0;
    size = 0;
    capacity;
    constructor(capacity) {
        this.capacity = capacity;
        this.buffer = new Array(capacity);
    }
    push(item) {
        this.buffer[this.head] = item;
        this.head = (this.head + 1) % this.capacity;
        this.size = Math.min(this.size + 1, this.capacity);
    }
    toArray() {
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
    get length() {
        return this.size;
    }
    get isFull() {
        return this.size === this.capacity;
    }
    clear() {
        this.head = 0;
        this.size = 0;
    }
}
exports.RingBuffer = RingBuffer;
/**
 * Memory-efficient storage for tag data using typed arrays
 */
class TagDataBuffer {
    capacity;
    values;
    timestamps;
    pointer = 0;
    isFull = false;
    constructor(capacity) {
        this.capacity = capacity;
        this.values = new Float32Array(capacity);
        this.timestamps = new Uint32Array(capacity);
    }
    addDataPoint(value, timestamp) {
        this.values[this.pointer] = value;
        this.timestamps[this.pointer] = Math.floor(timestamp / 1000); // Store as seconds
        this.pointer = (this.pointer + 1) % this.capacity;
        if (this.pointer === 0) {
            this.isFull = true;
        }
    }
    getChartData() {
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
        }
        else {
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
    getLatestValue() {
        if (this.pointer === 0 && !this.isFull) {
            return undefined;
        }
        const lastIndex = this.pointer === 0 ? this.capacity - 1 : this.pointer - 1;
        return this.values[lastIndex];
    }
    clear() {
        this.pointer = 0;
        this.isFull = false;
    }
    get memoryUsage() {
        // Return memory usage in bytes
        return this.values.byteLength + this.timestamps.byteLength;
    }
}
exports.TagDataBuffer = TagDataBuffer;
/**
 * LRU Cache for device metadata
 */
class LRUCache {
    cache = new Map();
    maxSize;
    constructor(maxSize = 1000) {
        this.maxSize = maxSize;
    }
    get(key) {
        const value = this.cache.get(key);
        if (value !== undefined) {
            // Move to end (most recently used)
            this.cache.delete(key);
            this.cache.set(key, value);
        }
        return value;
    }
    set(key, value) {
        if (this.cache.has(key)) {
            // Update existing
            this.cache.delete(key);
        }
        else if (this.cache.size >= this.maxSize) {
            // Remove least recently used (first item)
            const firstKey = this.cache.keys().next().value;
            if (firstKey !== undefined) {
                this.cache.delete(firstKey);
            }
        }
        this.cache.set(key, value);
    }
    has(key) {
        return this.cache.has(key);
    }
    delete(key) {
        return this.cache.delete(key);
    }
    clear() {
        this.cache.clear();
    }
    get size() {
        return this.cache.size;
    }
}
exports.LRUCache = LRUCache;
//# sourceMappingURL=dataBuffer.js.map