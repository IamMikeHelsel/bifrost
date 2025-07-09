export interface Connection {
    id: string;
    host: string;
    port: number;
    protocol: string;
    isAlive(): boolean;
    lastUsed: number;
    inUse: boolean;
    close(): Promise<void>;
}
export interface ConnectionStats {
    totalConnections: number;
    activeConnections: number;
    idleConnections: number;
    connectionsMade: number;
    connectionsReused: number;
    connectionErrors: number;
}
/**
 * High-performance connection pool for industrial protocol connections
 */
export declare class ConnectionPool {
    private connections;
    private readonly maxConnectionsPerHost;
    private readonly maxIdleTime;
    private readonly maxTotalConnections;
    private cleanupTimer?;
    private stats;
    constructor(options?: {
        maxConnectionsPerHost?: number;
        maxIdleTime?: number;
        maxTotalConnections?: number;
        cleanupInterval?: number;
    });
    getConnection(host: string, port: number, protocol: string, createConnection: () => Promise<Connection>): Promise<Connection>;
    releaseConnection(connection: Connection): void;
    closeConnection(connection: Connection): Promise<void>;
    private findAvailableConnection;
    private cleanup;
    private updateStats;
    getStats(): ConnectionStats;
    closeAll(): Promise<void>;
    dispose(): void;
}
/**
 * Multiplexed connection for sharing single TCP connection
 */
export declare class MultiplexedConnection {
    private baseConnection;
    private requests;
    private nextRequestId;
    constructor(baseConnection: Connection);
    sendRequest(data: any, timeoutMs?: number): Promise<any>;
    handleResponse(requestId: string, data: any): void;
    handleError(requestId: string, error: Error): void;
    dispose(): void;
}
