/**
 * High-performance ring buffer for time-series data
 * Optimized for real-time industrial data streams
 */
export declare class RingBuffer<T> {
    private buffer;
    private head;
    private size;
    private readonly capacity;
    constructor(capacity: number);
    push(item: T): void;
    toArray(): T[];
    get length(): number;
    get isFull(): boolean;
    clear(): void;
}
/**
 * Memory-efficient storage for tag data using typed arrays
 */
export declare class TagDataBuffer {
    private capacity;
    private values;
    private timestamps;
    private pointer;
    private isFull;
    constructor(capacity: number);
    addDataPoint(value: number, timestamp: number): void;
    getChartData(): Array<{
        x: number;
        y: number;
    }>;
    getLatestValue(): number | undefined;
    clear(): void;
    get memoryUsage(): number;
}
/**
 * LRU Cache for device metadata
 */
export declare class LRUCache<K, V> {
    private cache;
    private readonly maxSize;
    constructor(maxSize?: number);
    get(key: K): V | undefined;
    set(key: K, value: V): void;
    has(key: K): boolean;
    delete(key: K): boolean;
    clear(): void;
    get size(): number;
}
