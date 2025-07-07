package performance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// ConnectionPool manages a pool of connections with circuit breaker support
type ConnectionPool struct {
	logger    *zap.Logger
	config    *PoolConfig
	
	// Pool management
	pools     sync.Map // map[string]*DevicePool
	metrics   *PoolMetrics
	
	// Circuit breaker
	breakers  sync.Map // map[string]*gobreaker.CircuitBreaker
	
	// Cleanup and monitoring
	cleanup   chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
}

// PoolConfig defines connection pool configuration
type PoolConfig struct {
	MaxConnectionsPerDevice int           `yaml:"max_connections_per_device"`
	MaxTotalConnections     int           `yaml:"max_total_connections"`
	ConnectionTimeout       time.Duration `yaml:"connection_timeout"`
	IdleTimeout            time.Duration `yaml:"idle_timeout"`
	HealthCheckInterval    time.Duration `yaml:"health_check_interval"`
	RetryAttempts          int           `yaml:"retry_attempts"`
	RetryDelay             time.Duration `yaml:"retry_delay"`
	
	// Circuit breaker configuration
	CircuitBreakerConfig CircuitBreakerConfig `yaml:"circuit_breaker"`
}

// CircuitBreakerConfig defines circuit breaker settings
type CircuitBreakerConfig struct {
	MaxRequests    uint32        `yaml:"max_requests"`
	Interval       time.Duration `yaml:"interval"`
	Timeout        time.Duration `yaml:"timeout"`
	FailureRate    float64       `yaml:"failure_rate"`
	MinRequests    uint32        `yaml:"min_requests"`
}

// DevicePool manages connections for a specific device
type DevicePool struct {
	deviceID    string
	connections chan *PooledConnection
	active      int32
	total       int32
	config      *PoolConfig
	logger      *zap.Logger
	mutex       sync.RWMutex
	
	// Health monitoring
	lastHealthCheck time.Time
	healthy         bool
	
	// Statistics
	stats *DevicePoolStats
}

// PooledConnection wraps a connection with pool metadata
type PooledConnection struct {
	conn        Connection
	pool        *DevicePool
	createdAt   time.Time
	lastUsed    time.Time
	inUse       bool
	healthy     bool
	useCount    int64
	
	// Performance tracking
	totalLatency time.Duration
	requestCount int64
}

// Connection interface for pooled connections
type Connection interface {
	Connect() error
	Disconnect() error
	IsHealthy() bool
	Execute(ctx context.Context, request interface{}) (interface{}, error)
	GetStats() ConnectionStats
}

// ConnectionStats holds performance statistics for a connection
type ConnectionStats struct {
	RequestsTotal     int64
	RequestsSuccessful int64
	RequestsFailed    int64
	AverageLatency    time.Duration
	LastActivity      time.Time
}

// PoolMetrics tracks pool-wide performance metrics
type PoolMetrics struct {
	TotalConnections    int64
	ActiveConnections   int64
	FailedConnections   int64
	ConnectionsCreated  int64
	ConnectionsDestroyed int64
	
	// Performance metrics
	AverageLatency      time.Duration
	P95Latency         time.Duration
	P99Latency         time.Duration
	ThroughputPerSecond float64
	
	// Circuit breaker metrics
	CircuitBreakerTrips int64
	CircuitBreakerResets int64
}

// DevicePoolStats holds statistics for a device pool
type DevicePoolStats struct {
	TotalConnections   int32
	ActiveConnections  int32
	IdleConnections    int32
	FailedConnections  int32
	
	// Performance metrics
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	AverageLatency     time.Duration
	
	// Health metrics
	LastHealthCheck    time.Time
	HealthChecksPassed int64
	HealthChecksFailed int64
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *PoolConfig, logger *zap.Logger) *ConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &ConnectionPool{
		logger:  logger,
		config:  config,
		metrics: &PoolMetrics{},
		cleanup: make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
	
	// Start background maintenance
	go pool.maintenanceLoop()
	
	return pool
}

// GetConnection retrieves a connection from the pool
func (p *ConnectionPool) GetConnection(deviceID string, factory func() (Connection, error)) (*PooledConnection, error) {
	// Check circuit breaker
	breaker := p.getCircuitBreaker(deviceID)
	
	conn, err := breaker.Execute(func() (interface{}, error) {
		return p.getConnectionFromPool(deviceID, factory)
	})
	
	if err != nil {
		return nil, fmt.Errorf("circuit breaker: %w", err)
	}
	
	return conn.(*PooledConnection), nil
}

// getConnectionFromPool internal method to get connection from pool
func (p *ConnectionPool) getConnectionFromPool(deviceID string, factory func() (Connection, error)) (*PooledConnection, error) {
	pool := p.getDevicePool(deviceID)
	
	// Try to get existing connection
	select {
	case conn := <-pool.connections:
		if conn.healthy && time.Since(conn.lastUsed) < p.config.IdleTimeout {
			conn.inUse = true
			conn.lastUsed = time.Now()
			atomic.AddInt32(&pool.active, 1)
			return conn, nil
		}
		// Connection is stale, close it
		conn.Close()
		atomic.AddInt32(&pool.total, -1)
		
	default:
		// No available connections
	}
	
	// Create new connection if under limit
	if atomic.LoadInt32(&pool.total) < int32(p.config.MaxConnectionsPerDevice) {
		return p.createNewConnection(pool, factory)
	}
	
	// Wait for connection to become available
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ConnectionTimeout)
	defer cancel()
	
	select {
	case conn := <-pool.connections:
		if conn.healthy {
			conn.inUse = true
			conn.lastUsed = time.Now()
			atomic.AddInt32(&pool.active, 1)
			return conn, nil
		}
		// Connection is unhealthy, close it
		conn.Close()
		atomic.AddInt32(&pool.total, -1)
		return p.createNewConnection(pool, factory)
		
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for connection to %s", deviceID)
	}
}

// createNewConnection creates a new pooled connection
func (p *ConnectionPool) createNewConnection(pool *DevicePool, factory func() (Connection, error)) (*PooledConnection, error) {
	conn, err := factory()
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}
	
	if err := conn.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	
	pooledConn := &PooledConnection{
		conn:      conn,
		pool:      pool,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		inUse:     true,
		healthy:   true,
	}
	
	atomic.AddInt32(&pool.total, 1)
	atomic.AddInt32(&pool.active, 1)
	atomic.AddInt64(&p.metrics.ConnectionsCreated, 1)
	
	return pooledConn, nil
}

// ReturnConnection returns a connection to the pool
func (p *ConnectionPool) ReturnConnection(conn *PooledConnection) {
	if conn == nil {
		return
	}
	
	conn.inUse = false
	conn.lastUsed = time.Now()
	atomic.AddInt32(&conn.pool.active, -1)
	
	// Return to pool if healthy
	if conn.healthy {
		select {
		case conn.pool.connections <- conn:
			// Successfully returned to pool
		default:
			// Pool is full, close connection
			conn.Close()
			atomic.AddInt32(&conn.pool.total, -1)
		}
	} else {
		// Connection is unhealthy, close it
		conn.Close()
		atomic.AddInt32(&conn.pool.total, -1)
	}
}

// getDevicePool retrieves or creates a device pool
func (p *ConnectionPool) getDevicePool(deviceID string) *DevicePool {
	if pool, exists := p.pools.Load(deviceID); exists {
		return pool.(*DevicePool)
	}
	
	// Create new device pool
	pool := &DevicePool{
		deviceID:    deviceID,
		connections: make(chan *PooledConnection, p.config.MaxConnectionsPerDevice),
		config:      p.config,
		logger:      p.logger,
		healthy:     true,
		stats:       &DevicePoolStats{},
	}
	
	p.pools.Store(deviceID, pool)
	return pool
}

// getCircuitBreaker retrieves or creates a circuit breaker for a device
func (p *ConnectionPool) getCircuitBreaker(deviceID string) *gobreaker.CircuitBreaker {
	if breaker, exists := p.breakers.Load(deviceID); exists {
		return breaker.(*gobreaker.CircuitBreaker)
	}
	
	settings := gobreaker.Settings{
		Name:        fmt.Sprintf("device-%s", deviceID),
		MaxRequests: p.config.CircuitBreakerConfig.MaxRequests,
		Interval:    p.config.CircuitBreakerConfig.Interval,
		Timeout:     p.config.CircuitBreakerConfig.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRate := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= p.config.CircuitBreakerConfig.MinRequests &&
				   failureRate >= p.config.CircuitBreakerConfig.FailureRate
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			p.logger.Warn("Circuit breaker state changed",
				zap.String("device", deviceID),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
			
			if to == gobreaker.StateOpen {
				atomic.AddInt64(&p.metrics.CircuitBreakerTrips, 1)
			} else if to == gobreaker.StateClosed {
				atomic.AddInt64(&p.metrics.CircuitBreakerResets, 1)
			}
		},
	}
	
	breaker := gobreaker.NewCircuitBreaker(settings)
	p.breakers.Store(deviceID, breaker)
	return breaker
}

// maintenanceLoop runs background maintenance tasks
func (p *ConnectionPool) maintenanceLoop() {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.performMaintenance()
		}
	}
}

// performMaintenance performs pool maintenance tasks
func (p *ConnectionPool) performMaintenance() {
	now := time.Now()
	
	p.pools.Range(func(key, value interface{}) bool {
		pool := value.(*DevicePool)
		
		// Health check
		if now.Sub(pool.lastHealthCheck) > p.config.HealthCheckInterval {
			p.performHealthCheck(pool)
			pool.lastHealthCheck = now
		}
		
		// Remove idle connections
		p.removeIdleConnections(pool)
		
		return true
	})
}

// performHealthCheck checks the health of connections in a pool
func (p *ConnectionPool) performHealthCheck(pool *DevicePool) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	// Check connections in pool
	var healthyConnections []*PooledConnection
	
	for len(pool.connections) > 0 {
		select {
		case conn := <-pool.connections:
			if conn.conn.IsHealthy() {
				healthyConnections = append(healthyConnections, conn)
				pool.stats.HealthChecksPassed++
			} else {
				conn.Close()
				atomic.AddInt32(&pool.total, -1)
				pool.stats.HealthChecksFailed++
			}
		default:
			break
		}
	}
	
	// Return healthy connections to pool
	for _, conn := range healthyConnections {
		select {
		case pool.connections <- conn:
		default:
			// Pool is full, close connection
			conn.Close()
			atomic.AddInt32(&pool.total, -1)
		}
	}
}

// removeIdleConnections removes connections that have been idle too long
func (p *ConnectionPool) removeIdleConnections(pool *DevicePool) {
	var activeConnections []*PooledConnection
	idleThreshold := time.Now().Add(-p.config.IdleTimeout)
	
	for len(pool.connections) > 0 {
		select {
		case conn := <-pool.connections:
			if conn.lastUsed.After(idleThreshold) {
				activeConnections = append(activeConnections, conn)
			} else {
				conn.Close()
				atomic.AddInt32(&pool.total, -1)
			}
		default:
			break
		}
	}
	
	// Return active connections to pool
	for _, conn := range activeConnections {
		select {
		case pool.connections <- conn:
		default:
			// Pool is full, close connection
			conn.Close()
			atomic.AddInt32(&pool.total, -1)
		}
	}
}

// GetMetrics returns pool performance metrics
func (p *ConnectionPool) GetMetrics() *PoolMetrics {
	return p.metrics
}

// GetDeviceStats returns statistics for a specific device
func (p *ConnectionPool) GetDeviceStats(deviceID string) *DevicePoolStats {
	if pool, exists := p.pools.Load(deviceID); exists {
		return pool.(*DevicePool).stats
	}
	return nil
}

// Close gracefully shuts down the connection pool
func (p *ConnectionPool) Close() error {
	p.cancel()
	close(p.cleanup)
	
	// Close all connections
	p.pools.Range(func(key, value interface{}) bool {
		pool := value.(*DevicePool)
		
		// Close all connections in pool
		for len(pool.connections) > 0 {
			select {
			case conn := <-pool.connections:
				conn.Close()
			default:
				break
			}
		}
		
		close(pool.connections)
		return true
	})
	
	return nil
}

// Execute executes a request using a pooled connection
func (conn *PooledConnection) Execute(ctx context.Context, request interface{}) (interface{}, error) {
	start := time.Now()
	
	result, err := conn.conn.Execute(ctx, request)
	
	// Update statistics
	duration := time.Since(start)
	atomic.AddInt64(&conn.requestCount, 1)
	atomic.AddInt64(&conn.pool.stats.TotalRequests, 1)
	
	if err != nil {
		atomic.AddInt64(&conn.pool.stats.FailedRequests, 1)
		conn.healthy = false
	} else {
		atomic.AddInt64(&conn.pool.stats.SuccessfulRequests, 1)
		conn.totalLatency += duration
	}
	
	conn.lastUsed = time.Now()
	return result, err
}

// Close closes the pooled connection
func (conn *PooledConnection) Close() error {
	if conn.conn != nil {
		return conn.conn.Disconnect()
	}
	return nil
}

// IsHealthy checks if the connection is healthy
func (conn *PooledConnection) IsHealthy() bool {
	return conn.healthy && conn.conn.IsHealthy()
}

// GetStats returns connection statistics
func (conn *PooledConnection) GetStats() ConnectionStats {
	stats := conn.conn.GetStats()
	
	// Add pool-specific stats
	if conn.requestCount > 0 {
		stats.AverageLatency = time.Duration(int64(conn.totalLatency) / conn.requestCount)
	}
	
	return stats
}