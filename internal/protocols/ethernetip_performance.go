package protocols

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// EtherNetIPPerformanceOptimizer provides performance optimizations for EtherNet/IP communication
type EtherNetIPPerformanceOptimizer struct {
	handler        *EtherNetIPHandler
	logger         *zap.Logger
	connectionPool *EtherNetIPConnectionPool
	batchProcessor *CIPBatchProcessor
	tagCache       *EtherNetIPTagCache
	metrics        *EtherNetIPMetrics
}

// EtherNetIPConnectionPool manages connection pooling and reuse
type EtherNetIPConnectionPool struct {
	connections sync.Map // map[string]*PooledConnection
	maxIdle     int
	maxActive   int
	idleTimeout time.Duration
	mutex       sync.RWMutex
	logger      *zap.Logger
}

// PooledConnection represents a pooled EtherNet/IP connection
type PooledConnection struct {
	*EtherNetIPConnection
	inUse          bool
	pooledAt       time.Time
	lastActivity   time.Time
	usageCount     uint64
	maxUsageCount  uint64
	keepAliveTimer *time.Timer
}

// CIPBatchProcessor handles optimized batch operations
type CIPBatchProcessor struct {
	maxBatchSize   int
	maxWaitTime    time.Duration
	batchBuffer    map[string][]*BatchedRequest
	bufferMutex    sync.RWMutex
	flushTimers    map[string]*time.Timer
	timerMutex     sync.RWMutex
	requestCounter uint64
	logger         *zap.Logger
}

// BatchedRequest represents a request in a batch
type BatchedRequest struct {
	Tag        *Tag
	Operation  string // "read" or "write"
	Value      interface{}
	ResponseCh chan *BatchedResponse
	RequestID  uint64
	Timestamp  time.Time
}

// BatchedResponse represents a response from a batch operation
type BatchedResponse struct {
	Value     interface{}
	Error     error
	RequestID uint64
	Duration  time.Duration
}

// EtherNetIPTagCache provides intelligent tag caching
type EtherNetIPTagCache struct {
	cache       sync.Map // map[string]*CachedTag
	metrics     *CacheMetrics
	maxSize     int
	defaultTTL  time.Duration
	cleanupStop chan struct{}
	logger      *zap.Logger
}

// CachedTag represents a cached tag value
type CachedTag struct {
	Value       interface{}
	Quality     Quality
	Timestamp   time.Time
	ExpiresAt   time.Time
	AccessCount uint64
	LastAccess  time.Time
	Mutex       sync.RWMutex
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	Hits        uint64
	Misses      uint64
	Evictions   uint64
	Size        uint64
	TotalReads  uint64
	TotalWrites uint64
}

// EtherNetIPMetrics provides comprehensive performance metrics
type EtherNetIPMetrics struct {
	ConnectionCount     uint64
	ActiveConnections   uint64
	TotalRequests       uint64
	SuccessfulRequests  uint64
	FailedRequests      uint64
	AverageResponseTime time.Duration
	MinResponseTime     time.Duration
	MaxResponseTime     time.Duration
	BytesSent           uint64
	BytesReceived       uint64
	SessionsCreated     uint64
	SessionsDestroyed   uint64
	CacheMetrics        *CacheMetrics
	LastReset           time.Time
	mutex               sync.RWMutex
}

// NewEtherNetIPPerformanceOptimizer creates a new performance optimizer
func NewEtherNetIPPerformanceOptimizer(handler *EtherNetIPHandler, logger *zap.Logger) *EtherNetIPPerformanceOptimizer {
	optimizer := &EtherNetIPPerformanceOptimizer{
		handler: handler,
		logger:  logger,
		metrics: &EtherNetIPMetrics{
			MinResponseTime: time.Hour, // Initialize to high value
			LastReset:       time.Now(),
			CacheMetrics:    &CacheMetrics{},
		},
	}

	// Initialize connection pool
	optimizer.connectionPool = &EtherNetIPConnectionPool{
		maxIdle:     10,
		maxActive:   100,
		idleTimeout: 5 * time.Minute,
		logger:      logger,
	}

	// Initialize batch processor
	optimizer.batchProcessor = &CIPBatchProcessor{
		maxBatchSize: 50,
		maxWaitTime:  10 * time.Millisecond,
		batchBuffer:  make(map[string][]*BatchedRequest),
		flushTimers:  make(map[string]*time.Timer),
		logger:       logger,
	}

	// Initialize tag cache
	optimizer.tagCache = &EtherNetIPTagCache{
		maxSize:     10000,
		defaultTTL:  5 * time.Second,
		cleanupStop: make(chan struct{}),
		logger:      logger,
		metrics:     optimizer.metrics.CacheMetrics,
	}

	// Start background processes
	go optimizer.connectionPool.cleanupIdleConnections()
	go optimizer.tagCache.cleanupExpiredEntries()

	return optimizer
}

// OptimizedReadTag reads a tag with performance optimizations
func (opt *EtherNetIPPerformanceOptimizer) OptimizedReadTag(device *Device, tag *Tag) (interface{}, error) {
	startTime := time.Now()
	defer func() {
		opt.updateMetrics(time.Since(startTime), true)
	}()

	// Check cache first
	if cachedValue, found := opt.tagCache.Get(device.ID, tag.ID); found {
		atomic.AddUint64(&opt.metrics.CacheMetrics.Hits, 1)
		return cachedValue, nil
	}

	atomic.AddUint64(&opt.metrics.CacheMetrics.Misses, 1)

	// Get pooled connection
	conn, err := opt.connectionPool.GetConnection(device)
	if err != nil {
		opt.updateMetrics(time.Since(startTime), false)
		return nil, err
	}
	defer opt.connectionPool.ReleaseConnection(device.ID, conn)

	// Perform optimized read
	value, err := opt.performOptimizedRead(conn, device, tag)
	if err != nil {
		opt.updateMetrics(time.Since(startTime), false)
		return nil, err
	}

	// Cache the result
	opt.tagCache.Set(device.ID, tag.ID, value, opt.tagCache.defaultTTL)

	return value, nil
}

// OptimizedReadMultipleTags reads multiple tags with advanced batching and caching
func (opt *EtherNetIPPerformanceOptimizer) OptimizedReadMultipleTags(device *Device, tags []*Tag) (map[string]interface{}, error) {
	startTime := time.Now()
	defer func() {
		opt.updateMetrics(time.Since(startTime), true)
	}()

	results := make(map[string]interface{})
	var uncachedTags []*Tag

	// Check cache for all tags first
	for _, tag := range tags {
		if cachedValue, found := opt.tagCache.Get(device.ID, tag.ID); found {
			results[tag.ID] = cachedValue
			atomic.AddUint64(&opt.metrics.CacheMetrics.Hits, 1)
		} else {
			uncachedTags = append(uncachedTags, tag)
			atomic.AddUint64(&opt.metrics.CacheMetrics.Misses, 1)
		}
	}

	// If all tags were cached, return immediately
	if len(uncachedTags) == 0 {
		return results, nil
	}

	// Get pooled connection
	conn, err := opt.connectionPool.GetConnection(device)
	if err != nil {
		opt.updateMetrics(time.Since(startTime), false)
		return nil, err
	}
	defer opt.connectionPool.ReleaseConnection(device.ID, conn)

	// Group uncached tags for optimal batch reading
	optimizedBatches := opt.optimizeTagBatches(uncachedTags)

	for _, batch := range optimizedBatches {
		batchResults, err := opt.performOptimizedBatchRead(conn, device, batch)
		if err != nil {
			// Fall back to individual reads for this batch
			for _, tag := range batch {
				if value, readErr := opt.performOptimizedRead(conn, device, tag); readErr == nil {
					results[tag.ID] = value
					opt.tagCache.Set(device.ID, tag.ID, value, opt.tagCache.defaultTTL)
				}
			}
		} else {
			for tagID, value := range batchResults {
				results[tagID] = value
				// Find the tag to cache with proper TTL
				for _, tag := range batch {
					if tag.ID == tagID {
						opt.tagCache.Set(device.ID, tag.ID, value, opt.tagCache.defaultTTL)
						break
					}
				}
			}
		}
	}

	return results, nil
}

// GetConnection retrieves a connection from the pool
func (pool *EtherNetIPConnectionPool) GetConnection(device *Device) (*PooledConnection, error) {
	connectionKey := fmt.Sprintf("%s:%d", device.Address, device.Port)

	// Try to get existing connection from pool
	if connInterface, exists := pool.connections.Load(connectionKey); exists {
		pooled := connInterface.(*PooledConnection)
		pooled.EtherNetIPConnection.mutex.Lock()
		defer pooled.EtherNetIPConnection.mutex.Unlock()

		if !pooled.inUse && pooled.isConnected && time.Since(pooled.lastActivity) < pool.idleTimeout {
			pooled.inUse = true
			pooled.lastActivity = time.Now()
			atomic.AddUint64(&pooled.usageCount, 1)
			return pooled, nil
		}
	}

	// Create new connection
	return pool.createNewConnection(device)
}

// ReleaseConnection returns a connection to the pool
func (pool *EtherNetIPConnectionPool) ReleaseConnection(deviceID string, conn *PooledConnection) {
	conn.EtherNetIPConnection.mutex.Lock()
	defer conn.EtherNetIPConnection.mutex.Unlock()

	conn.inUse = false
	conn.lastActivity = time.Now()

	// Check if connection should be discarded
	if conn.usageCount >= conn.maxUsageCount || !conn.isConnected {
		pool.connections.Delete(deviceID)
		if conn.tcpConn != nil {
			conn.tcpConn.Close()
		}
		return
	}

	// Set up keep-alive if needed
	if conn.keepAliveTimer != nil {
		conn.keepAliveTimer.Stop()
	}

	conn.keepAliveTimer = time.AfterFunc(pool.idleTimeout, func() {
		pool.connections.Delete(deviceID)
		if conn.tcpConn != nil {
			conn.tcpConn.Close()
		}
	})
}

// createNewConnection creates a new pooled connection
func (pool *EtherNetIPConnectionPool) createNewConnection(device *Device) (*PooledConnection, error) {
	// This would integrate with the main handler's Connect method
	// For now, return a placeholder
	return nil, fmt.Errorf("connection creation not implemented in pool")
}

// cleanupIdleConnections periodically cleans up idle connections
func (pool *EtherNetIPConnectionPool) cleanupIdleConnections() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pool.connections.Range(func(key, value interface{}) bool {
			pooled := value.(*PooledConnection)
			pooled.EtherNetIPConnection.mutex.RLock()
			shouldRemove := !pooled.inUse && time.Since(pooled.lastActivity) > pool.idleTimeout
			pooled.EtherNetIPConnection.mutex.RUnlock()

			if shouldRemove {
				pool.connections.Delete(key)
				if pooled.tcpConn != nil {
					pooled.tcpConn.Close()
				}
				pool.logger.Debug("Cleaned up idle connection", zap.String("key", key.(string)))
			}
			return true
		})
	}
}

// optimizeTagBatches groups tags for optimal batch reading based on addressing patterns
func (opt *EtherNetIPPerformanceOptimizer) optimizeTagBatches(tags []*Tag) [][]*Tag {
	// Group tags by addressing type and proximity
	symbolicTags := make([]*Tag, 0)
	instanceTags := make([]*Tag, 0)

	for _, tag := range tags {
		addr, err := opt.handler.parseAddress(tag.Address)
		if err != nil {
			continue
		}

		if addr.IsSymbolic {
			symbolicTags = append(symbolicTags, tag)
		} else {
			instanceTags = append(instanceTags, tag)
		}
	}

	var batches [][]*Tag

	// Create batches for symbolic tags (group by similar tag names)
	batches = append(batches, opt.createSymbolicBatches(symbolicTags)...)

	// Create batches for instance tags (group by consecutive instances)
	batches = append(batches, opt.createInstanceBatches(instanceTags)...)

	return batches
}

// createSymbolicBatches creates optimized batches for symbolic addressing
func (opt *EtherNetIPPerformanceOptimizer) createSymbolicBatches(tags []*Tag) [][]*Tag {
	var batches [][]*Tag

	// Simple batching - group by tag name prefix
	tagGroups := make(map[string][]*Tag)

	for _, tag := range tags {
		prefix := opt.getTagPrefix(tag.Address)
		tagGroups[prefix] = append(tagGroups[prefix], tag)
	}

	for _, group := range tagGroups {
		// Split large groups into smaller batches
		for i := 0; i < len(group); i += opt.batchProcessor.maxBatchSize {
			end := i + opt.batchProcessor.maxBatchSize
			if end > len(group) {
				end = len(group)
			}
			batches = append(batches, group[i:end])
		}
	}

	return batches
}

// createInstanceBatches creates optimized batches for instance addressing
func (opt *EtherNetIPPerformanceOptimizer) createInstanceBatches(tags []*Tag) [][]*Tag {
	if len(tags) == 0 {
		return nil
	}

	// Sort tags by instance ID for consecutive batching
	sort.Slice(tags, func(i, j int) bool {
		addrI, _ := opt.handler.parseAddress(tags[i].Address)
		addrJ, _ := opt.handler.parseAddress(tags[j].Address)
		return addrI.InstanceID < addrJ.InstanceID
	})

	var batches [][]*Tag
	currentBatch := []*Tag{tags[0]}

	for i := 1; i < len(tags); i++ {
		addrCurrent, _ := opt.handler.parseAddress(tags[i].Address)
		addrPrev, _ := opt.handler.parseAddress(tags[i-1].Address)

		// Check if instances are consecutive and batch isn't too large
		if addrCurrent.InstanceID == addrPrev.InstanceID+1 && len(currentBatch) < opt.batchProcessor.maxBatchSize {
			currentBatch = append(currentBatch, tags[i])
		} else {
			batches = append(batches, currentBatch)
			currentBatch = []*Tag{tags[i]}
		}
	}

	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}

// getTagPrefix extracts a prefix from a tag address for grouping
func (opt *EtherNetIPPerformanceOptimizer) getTagPrefix(address string) string {
	// Simple implementation - return first part before array index or dot
	if idx := strings.IndexAny(address, "[."); idx > 0 {
		return address[:idx]
	}
	return address
}

// performOptimizedRead performs an optimized single tag read
func (opt *EtherNetIPPerformanceOptimizer) performOptimizedRead(conn *PooledConnection, device *Device, tag *Tag) (interface{}, error) {
	// Use the main handler's ReadTag method but with pooled connection
	// This would need integration with the main handler
	return opt.handler.ReadTag(device, tag)
}

// performOptimizedBatchRead performs an optimized batch read using Multiple Service Packet
func (opt *EtherNetIPPerformanceOptimizer) performOptimizedBatchRead(conn *PooledConnection, device *Device, tags []*Tag) (map[string]interface{}, error) {
	// Implementation would use CIP Multiple Service Packet for true batch reading
	// For now, fall back to multiple individual reads
	results := make(map[string]interface{})

	for _, tag := range tags {
		if value, err := opt.performOptimizedRead(conn, device, tag); err == nil {
			results[tag.ID] = value
		}
	}

	return results, nil
}

// Tag Cache Implementation

// Get retrieves a value from the cache
func (cache *EtherNetIPTagCache) Get(deviceID, tagID string) (interface{}, bool) {
	key := fmt.Sprintf("%s:%s", deviceID, tagID)

	if valueInterface, exists := cache.cache.Load(key); exists {
		cached := valueInterface.(*CachedTag)
		cached.Mutex.RLock()
		defer cached.Mutex.RUnlock()

		if time.Now().Before(cached.ExpiresAt) {
			atomic.AddUint64(&cached.AccessCount, 1)
			cached.LastAccess = time.Now()
			atomic.AddUint64(&cache.metrics.TotalReads, 1)
			return cached.Value, true
		}

		// Expired entry
		cache.cache.Delete(key)
		atomic.AddUint64(&cache.metrics.Evictions, 1)
		atomic.AddUint64(&cache.metrics.Size, ^uint64(0)) // Decrement
	}

	return nil, false
}

// Set stores a value in the cache
func (cache *EtherNetIPTagCache) Set(deviceID, tagID string, value interface{}, ttl time.Duration) {
	key := fmt.Sprintf("%s:%s", deviceID, tagID)

	cached := &CachedTag{
		Value:       value,
		Quality:     QualityGood,
		Timestamp:   time.Now(),
		ExpiresAt:   time.Now().Add(ttl),
		AccessCount: 1,
		LastAccess:  time.Now(),
	}

	// Check cache size limit
	if atomic.LoadUint64(&cache.metrics.Size) >= uint64(cache.maxSize) {
		cache.evictLeastRecentlyUsed()
	}

	cache.cache.Store(key, cached)
	atomic.AddUint64(&cache.metrics.Size, 1)
	atomic.AddUint64(&cache.metrics.TotalWrites, 1)
}

// evictLeastRecentlyUsed removes the least recently used entry
func (cache *EtherNetIPTagCache) evictLeastRecentlyUsed() {
	var oldestKey interface{}
	var oldestTime time.Time = time.Now()

	cache.cache.Range(func(key, value interface{}) bool {
		cached := value.(*CachedTag)
		cached.Mutex.RLock()
		lastAccess := cached.LastAccess
		cached.Mutex.RUnlock()

		if lastAccess.Before(oldestTime) {
			oldestTime = lastAccess
			oldestKey = key
		}
		return true
	})

	if oldestKey != nil {
		cache.cache.Delete(oldestKey)
		atomic.AddUint64(&cache.metrics.Evictions, 1)
		atomic.AddUint64(&cache.metrics.Size, ^uint64(0)) // Decrement
	}
}

// cleanupExpiredEntries periodically removes expired cache entries
func (cache *EtherNetIPTagCache) cleanupExpiredEntries() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			var keysToDelete []interface{}

			cache.cache.Range(func(key, value interface{}) bool {
				cached := value.(*CachedTag)
				cached.Mutex.RLock()
				expired := now.After(cached.ExpiresAt)
				cached.Mutex.RUnlock()

				if expired {
					keysToDelete = append(keysToDelete, key)
				}
				return true
			})

			for _, key := range keysToDelete {
				cache.cache.Delete(key)
				atomic.AddUint64(&cache.metrics.Evictions, 1)
				atomic.AddUint64(&cache.metrics.Size, ^uint64(0)) // Decrement
			}

			if len(keysToDelete) > 0 {
				cache.logger.Debug("Cleaned up expired cache entries", zap.Int("count", len(keysToDelete)))
			}

		case <-cache.cleanupStop:
			return
		}
	}
}

// Metrics and Monitoring

// updateMetrics updates performance metrics
func (opt *EtherNetIPPerformanceOptimizer) updateMetrics(duration time.Duration, success bool) {
	opt.metrics.mutex.Lock()
	defer opt.metrics.mutex.Unlock()

	atomic.AddUint64(&opt.metrics.TotalRequests, 1)

	if success {
		atomic.AddUint64(&opt.metrics.SuccessfulRequests, 1)
	} else {
		atomic.AddUint64(&opt.metrics.FailedRequests, 1)
	}

	// Update response time metrics
	if duration < opt.metrics.MinResponseTime {
		opt.metrics.MinResponseTime = duration
	}
	if duration > opt.metrics.MaxResponseTime {
		opt.metrics.MaxResponseTime = duration
	}

	// Calculate running average (simplified)
	totalRequests := atomic.LoadUint64(&opt.metrics.TotalRequests)
	if totalRequests > 0 {
		currentAvg := opt.metrics.AverageResponseTime
		opt.metrics.AverageResponseTime = time.Duration(
			(int64(currentAvg)*int64(totalRequests-1) + int64(duration)) / int64(totalRequests),
		)
	}
}

// GetMetrics returns current performance metrics
func (opt *EtherNetIPPerformanceOptimizer) GetMetrics() *EtherNetIPMetrics {
	opt.metrics.mutex.RLock()
	defer opt.metrics.mutex.RUnlock()

	// Return a copy to avoid race conditions
	return &EtherNetIPMetrics{
		ConnectionCount:     atomic.LoadUint64(&opt.metrics.ConnectionCount),
		ActiveConnections:   atomic.LoadUint64(&opt.metrics.ActiveConnections),
		TotalRequests:       atomic.LoadUint64(&opt.metrics.TotalRequests),
		SuccessfulRequests:  atomic.LoadUint64(&opt.metrics.SuccessfulRequests),
		FailedRequests:      atomic.LoadUint64(&opt.metrics.FailedRequests),
		AverageResponseTime: opt.metrics.AverageResponseTime,
		MinResponseTime:     opt.metrics.MinResponseTime,
		MaxResponseTime:     opt.metrics.MaxResponseTime,
		BytesSent:           atomic.LoadUint64(&opt.metrics.BytesSent),
		BytesReceived:       atomic.LoadUint64(&opt.metrics.BytesReceived),
		SessionsCreated:     atomic.LoadUint64(&opt.metrics.SessionsCreated),
		SessionsDestroyed:   atomic.LoadUint64(&opt.metrics.SessionsDestroyed),
		CacheMetrics:        opt.getCacheMetricsCopy(),
		LastReset:           opt.metrics.LastReset,
	}
}

// getCacheMetricsCopy returns a copy of cache metrics
func (opt *EtherNetIPPerformanceOptimizer) getCacheMetricsCopy() *CacheMetrics {
	return &CacheMetrics{
		Hits:        atomic.LoadUint64(&opt.metrics.CacheMetrics.Hits),
		Misses:      atomic.LoadUint64(&opt.metrics.CacheMetrics.Misses),
		Evictions:   atomic.LoadUint64(&opt.metrics.CacheMetrics.Evictions),
		Size:        atomic.LoadUint64(&opt.metrics.CacheMetrics.Size),
		TotalReads:  atomic.LoadUint64(&opt.metrics.CacheMetrics.TotalReads),
		TotalWrites: atomic.LoadUint64(&opt.metrics.CacheMetrics.TotalWrites),
	}
}

// ResetMetrics resets all performance metrics
func (opt *EtherNetIPPerformanceOptimizer) ResetMetrics() {
	opt.metrics.mutex.Lock()
	defer opt.metrics.mutex.Unlock()

	atomic.StoreUint64(&opt.metrics.TotalRequests, 0)
	atomic.StoreUint64(&opt.metrics.SuccessfulRequests, 0)
	atomic.StoreUint64(&opt.metrics.FailedRequests, 0)
	opt.metrics.AverageResponseTime = 0
	opt.metrics.MinResponseTime = time.Hour
	opt.metrics.MaxResponseTime = 0
	atomic.StoreUint64(&opt.metrics.BytesSent, 0)
	atomic.StoreUint64(&opt.metrics.BytesReceived, 0)

	// Reset cache metrics
	atomic.StoreUint64(&opt.metrics.CacheMetrics.Hits, 0)
	atomic.StoreUint64(&opt.metrics.CacheMetrics.Misses, 0)
	atomic.StoreUint64(&opt.metrics.CacheMetrics.TotalReads, 0)
	atomic.StoreUint64(&opt.metrics.CacheMetrics.TotalWrites, 0)

	opt.metrics.LastReset = time.Now()
}
