import * as vscode from 'vscode';

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
export class ConnectionPool {
    private connections = new Map<string, Connection[]>();
    private readonly maxConnectionsPerHost: number;
    private readonly maxIdleTime: number;
    private readonly maxTotalConnections: number;
    private cleanupTimer?: NodeJS.Timeout;
    
    private stats: ConnectionStats = {
        totalConnections: 0,
        activeConnections: 0,
        idleConnections: 0,
        connectionsMade: 0,
        connectionsReused: 0,
        connectionErrors: 0
    };
    
    constructor(options: {
        maxConnectionsPerHost?: number;
        maxIdleTime?: number;
        maxTotalConnections?: number;
        cleanupInterval?: number;
    } = {}) {
        this.maxConnectionsPerHost = options.maxConnectionsPerHost ?? 5;
        this.maxIdleTime = options.maxIdleTime ?? 60000; // 1 minute
        this.maxTotalConnections = options.maxTotalConnections ?? 100;
        
        // Start cleanup timer
        this.cleanupTimer = setInterval(() => {
            this.cleanup();
        }, options.cleanupInterval ?? 30000); // 30 seconds
    }
    
    async getConnection(
        host: string, 
        port: number, 
        protocol: string,
        createConnection: () => Promise<Connection>
    ): Promise<Connection> {
        const hostKey = `${host}:${port}:${protocol}`;
        
        // Try to reuse existing connection
        const existingConnection = this.findAvailableConnection(hostKey);
        if (existingConnection) {
            existingConnection.inUse = true;
            existingConnection.lastUsed = Date.now();
            this.stats.connectionsReused++;
            this.updateStats();
            return existingConnection;
        }
        
        // Check connection limits
        if (this.stats.totalConnections >= this.maxTotalConnections) {
            throw new Error('Maximum total connections reached');
        }
        
        const hostConnections = this.connections.get(hostKey) || [];
        if (hostConnections.length >= this.maxConnectionsPerHost) {
            throw new Error(`Maximum connections per host reached for ${hostKey}`);
        }
        
        // Create new connection
        try {
            const connection = await createConnection();
            connection.inUse = true;
            connection.lastUsed = Date.now();
            
            // Add to pool
            if (!this.connections.has(hostKey)) {
                this.connections.set(hostKey, []);
            }
            this.connections.get(hostKey)!.push(connection);
            
            this.stats.connectionsMade++;
            this.stats.totalConnections++;
            this.updateStats();
            
            return connection;
        } catch (error) {
            this.stats.connectionErrors++;
            throw error;
        }
    }
    
    releaseConnection(connection: Connection): void {
        connection.inUse = false;
        connection.lastUsed = Date.now();
        this.updateStats();
    }
    
    async closeConnection(connection: Connection): Promise<void> {
        // Remove from pool
        for (const [hostKey, connections] of this.connections.entries()) {
            const index = connections.indexOf(connection);
            if (index !== -1) {
                connections.splice(index, 1);
                if (connections.length === 0) {
                    this.connections.delete(hostKey);
                }
                break;
            }
        }
        
        // Close the connection
        await connection.close();
        this.stats.totalConnections--;
        this.updateStats();
    }
    
    private findAvailableConnection(hostKey: string): Connection | undefined {
        const connections = this.connections.get(hostKey);
        if (!connections) {
            return undefined;
        }
        
        return connections.find(conn => 
            !conn.inUse && 
            conn.isAlive() &&
            (Date.now() - conn.lastUsed) < this.maxIdleTime
        );
    }
    
    private cleanup(): void {
        const now = Date.now();
        const connectionsToClose: Connection[] = [];
        
        for (const [hostKey, connections] of this.connections.entries()) {
            // Find idle connections that exceeded max idle time
            const idle = connections.filter(conn => 
                !conn.inUse && 
                (now - conn.lastUsed) > this.maxIdleTime
            );
            
            // Find dead connections
            const dead = connections.filter(conn => !conn.isAlive());
            
            connectionsToClose.push(...idle, ...dead);
            
            // Remove from array
            const remaining = connections.filter(conn => 
                !idle.includes(conn) && !dead.includes(conn)
            );
            
            if (remaining.length === 0) {
                this.connections.delete(hostKey);
            } else {
                this.connections.set(hostKey, remaining);
            }
        }
        
        // Close connections asynchronously
        connectionsToClose.forEach(async conn => {
            try {
                await conn.close();
                this.stats.totalConnections--;
            } catch (error) {
                console.error('Error closing connection:', error);
            }
        });
        
        this.updateStats();
    }
    
    private updateStats(): void {
        this.stats.activeConnections = 0;
        this.stats.idleConnections = 0;
        
        for (const connections of this.connections.values()) {
            for (const conn of connections) {
                if (conn.inUse) {
                    this.stats.activeConnections++;
                } else {
                    this.stats.idleConnections++;
                }
            }
        }
    }
    
    getStats(): ConnectionStats {
        return { ...this.stats };
    }
    
    async closeAll(): Promise<void> {
        const allConnections: Connection[] = [];
        
        for (const connections of this.connections.values()) {
            allConnections.push(...connections);
        }
        
        this.connections.clear();
        
        await Promise.all(
            allConnections.map(async conn => {
                try {
                    await conn.close();
                } catch (error) {
                    console.error('Error closing connection:', error);
                }
            })
        );
        
        this.stats.totalConnections = 0;
        this.stats.activeConnections = 0;
        this.stats.idleConnections = 0;
    }
    
    dispose(): void {
        if (this.cleanupTimer) {
            clearInterval(this.cleanupTimer);
        }
        
        // Close all connections
        this.closeAll().catch(error => {
            console.error('Error disposing connection pool:', error);
        });
    }
}

/**
 * Multiplexed connection for sharing single TCP connection
 */
export class MultiplexedConnection {
    private requests = new Map<string, {
        resolve: (value: any) => void;
        reject: (error: Error) => void;
        timeout: NodeJS.Timeout;
    }>();
    private nextRequestId = 1;
    
    constructor(private baseConnection: Connection) {}
    
    async sendRequest(data: any, timeoutMs: number = 5000): Promise<any> {
        return new Promise((resolve, reject) => {
            const requestId = (this.nextRequestId++).toString();
            
            const timeout = setTimeout(() => {
                this.requests.delete(requestId);
                reject(new Error('Request timeout'));
            }, timeoutMs);
            
            this.requests.set(requestId, { resolve, reject, timeout });
            
            // TODO: Send data with request ID
            // This would be protocol-specific implementation
        });
    }
    
    handleResponse(requestId: string, data: any): void {
        const request = this.requests.get(requestId);
        if (request) {
            clearTimeout(request.timeout);
            this.requests.delete(requestId);
            request.resolve(data);
        }
    }
    
    handleError(requestId: string, error: Error): void {
        const request = this.requests.get(requestId);
        if (request) {
            clearTimeout(request.timeout);
            this.requests.delete(requestId);
            request.reject(error);
        }
    }
    
    dispose(): void {
        // Cancel all pending requests
        for (const [requestId, request] of this.requests.entries()) {
            clearTimeout(request.timeout);
            request.reject(new Error('Connection disposed'));
        }
        this.requests.clear();
    }
}