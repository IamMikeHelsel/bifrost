
Object.defineProperty(exports, "__esModule", { value: true });
exports.MultiplexedConnection = exports.ConnectionPool = void 0;
/**
 * High-performance connection pool for industrial protocol connections
 */
class ConnectionPool {
    connections = new Map();
    maxConnectionsPerHost;
    maxIdleTime;
    maxTotalConnections;
    cleanupTimer;
    stats = {
        totalConnections: 0,
        activeConnections: 0,
        idleConnections: 0,
        connectionsMade: 0,
        connectionsReused: 0,
        connectionErrors: 0
    };
    constructor(options = {}) {
        this.maxConnectionsPerHost = options.maxConnectionsPerHost ?? 5;
        this.maxIdleTime = options.maxIdleTime ?? 60000; // 1 minute
        this.maxTotalConnections = options.maxTotalConnections ?? 100;
        // Start cleanup timer
        this.cleanupTimer = setInterval(() => {
            this.cleanup();
        }, options.cleanupInterval ?? 30000); // 30 seconds
    }
    async getConnection(host, port, protocol, createConnection) {
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
            this.connections.get(hostKey).push(connection);
            this.stats.connectionsMade++;
            this.stats.totalConnections++;
            this.updateStats();
            return connection;
        }
        catch (error) {
            this.stats.connectionErrors++;
            throw error;
        }
    }
    releaseConnection(connection) {
        connection.inUse = false;
        connection.lastUsed = Date.now();
        this.updateStats();
    }
    async closeConnection(connection) {
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
    findAvailableConnection(hostKey) {
        const connections = this.connections.get(hostKey);
        if (!connections) {
            return undefined;
        }
        return connections.find(conn => !conn.inUse &&
            conn.isAlive() &&
            (Date.now() - conn.lastUsed) < this.maxIdleTime);
    }
    cleanup() {
        const now = Date.now();
        const connectionsToClose = [];
        for (const [hostKey, connections] of this.connections.entries()) {
            // Find idle connections that exceeded max idle time
            const idle = connections.filter(conn => !conn.inUse &&
                (now - conn.lastUsed) > this.maxIdleTime);
            // Find dead connections
            const dead = connections.filter(conn => !conn.isAlive());
            connectionsToClose.push(...idle, ...dead);
            // Remove from array
            const remaining = connections.filter(conn => !idle.includes(conn) && !dead.includes(conn));
            if (remaining.length === 0) {
                this.connections.delete(hostKey);
            }
            else {
                this.connections.set(hostKey, remaining);
            }
        }
        // Close connections asynchronously
        connectionsToClose.forEach(async conn => {
            try {
                await conn.close();
                this.stats.totalConnections--;
            }
            catch (error) {
                console.error('Error closing connection:', error);
            }
        });
        this.updateStats();
    }
    updateStats() {
        this.stats.activeConnections = 0;
        this.stats.idleConnections = 0;
        for (const connections of this.connections.values()) {
            for (const conn of connections) {
                if (conn.inUse) {
                    this.stats.activeConnections++;
                }
                else {
                    this.stats.idleConnections++;
                }
            }
        }
    }
    getStats() {
        return { ...this.stats };
    }
    async closeAll() {
        const allConnections = [];
        for (const connections of this.connections.values()) {
            allConnections.push(...connections);
        }
        this.connections.clear();
        await Promise.all(allConnections.map(async conn => {
            try {
                await conn.close();
            }
            catch (error) {
                console.error('Error closing connection:', error);
            }
        }));
        this.stats.totalConnections = 0;
        this.stats.activeConnections = 0;
        this.stats.idleConnections = 0;
    }
    dispose() {
        if (this.cleanupTimer) {
            clearInterval(this.cleanupTimer);
        }
        // Close all connections
        this.closeAll().catch(error => {
            console.error('Error disposing connection pool:', error);
        });
    }
}
exports.ConnectionPool = ConnectionPool;
/**
 * Multiplexed connection for sharing single TCP connection
 */
class MultiplexedConnection {
    baseConnection;
    requests = new Map();
    nextRequestId = 1;
    constructor(baseConnection) {
        this.baseConnection = baseConnection;
    }
    async sendRequest(data, timeoutMs = 5000) {
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
    handleResponse(requestId, data) {
        const request = this.requests.get(requestId);
        if (request) {
            clearTimeout(request.timeout);
            this.requests.delete(requestId);
            request.resolve(data);
        }
    }
    handleError(requestId, error) {
        const request = this.requests.get(requestId);
        if (request) {
            clearTimeout(request.timeout);
            this.requests.delete(requestId);
            request.reject(error);
        }
    }
    dispose() {
        // Cancel all pending requests
        for (const [requestId, request] of this.requests.entries()) {
            clearTimeout(request.timeout);
            request.reject(new Error('Connection disposed'));
        }
        this.requests.clear();
    }
}
exports.MultiplexedConnection = MultiplexedConnection;
//# sourceMappingURL=connectionPool.js.map