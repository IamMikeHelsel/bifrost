package performance

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// BatchProcessor manages intelligent request batching for optimal network utilization
type BatchProcessor struct {
	logger         *zap.Logger
	config         *BatchConfig
	batches        sync.Map // map[string]*DeviceBatch
	metrics        *BatchMetrics
	ctx            context.Context
	cancel         context.CancelFunc
	flushTimer     *time.Timer
	timerMutex     sync.Mutex
}

// BatchConfig defines batching configuration
type BatchConfig struct {
	MaxBatchSize     int           `yaml:"max_batch_size"`
	BatchTimeout     time.Duration `yaml:"batch_timeout"`
	FlushInterval    time.Duration `yaml:"flush_interval"`
	MaxConcurrentBatches int       `yaml:"max_concurrent_batches"`
	
	// Adaptive batching parameters
	EnableAdaptiveBatching bool    `yaml:"enable_adaptive_batching"`
	MinBatchSize          int     `yaml:"min_batch_size"`
	LatencyThreshold      time.Duration `yaml:"latency_threshold"`
	ThroughputThreshold   float64 `yaml:"throughput_threshold"`
}

// BatchMetrics tracks batching performance
type BatchMetrics struct {
	TotalBatches        int64
	BatchesProcessed    int64
	BatchesFailed       int64
	TotalRequests       int64
	BatchedRequests     int64
	SingleRequests      int64
	
	AverageBatchSize    float64
	AverageLatency      time.Duration
	ThroughputPerSecond float64
	
	// Adaptive metrics
	BatchSizeAdjustments int64
	OptimalBatchSize     int32
}

// DeviceBatch manages batching for a specific device
type DeviceBatch struct {
	deviceID       string
	requests       []*BatchRequest
	mutex          sync.Mutex
	lastFlush      time.Time
	processor      *BatchProcessor
	
	// Adaptive parameters
	currentBatchSize int
	avgLatency       time.Duration
	throughput       float64
	errorRate        float64
	
	// Statistics
	stats *DeviceBatchStats
}

// BatchRequest represents a request to be batched
type BatchRequest struct {
	ID          string
	DeviceID    string
	Operation   string
	Address     string
	DataType    string
	Value       interface{}
	Timestamp   time.Time
	Callback    func(result interface{}, err error)
	Context     context.Context
	
	// Batching metadata
	CanBatch    bool
	Priority    int
	Deadline    time.Time
}

// BatchResult contains the result of a batch operation
type BatchResult struct {
	RequestID string
	Value     interface{}
	Error     error
	Latency   time.Duration
}

// DeviceBatchStats holds statistics for device batching
type DeviceBatchStats struct {
	TotalBatches       int64
	SuccessfulBatches  int64
	FailedBatches      int64
	AverageBatchSize   float64
	AverageLatency     time.Duration
	ThroughputPerSecond float64
	OptimalBatchSize   int32
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(config *BatchConfig, logger *zap.Logger) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	processor := &BatchProcessor{
		logger:  logger,
		config:  config,
		metrics: &BatchMetrics{OptimalBatchSize: int32(config.MaxBatchSize)},
		ctx:     ctx,
		cancel:  cancel,
	}
	
	// Start background flush timer
	processor.startFlushTimer()
	
	return processor
}

// AddRequest adds a request to the batch processor
func (bp *BatchProcessor) AddRequest(request *BatchRequest) {
	if !request.CanBatch {
		// Execute immediately for non-batchable requests
		bp.executeSingle(request)
		atomic.AddInt64(&bp.metrics.SingleRequests, 1)
		return
	}
	
	batch := bp.getOrCreateBatch(request.DeviceID)
	batch.addRequest(request)
	atomic.AddInt64(&bp.metrics.TotalRequests, 1)
}

// getOrCreateBatch retrieves or creates a batch for a device
func (bp *BatchProcessor) getOrCreateBatch(deviceID string) *DeviceBatch {
	if batch, exists := bp.batches.Load(deviceID); exists {
		return batch.(*DeviceBatch)
	}
	
	batch := &DeviceBatch{
		deviceID:         deviceID,
		requests:         make([]*BatchRequest, 0, bp.config.MaxBatchSize),
		lastFlush:        time.Now(),
		processor:        bp,
		currentBatchSize: bp.config.MaxBatchSize,
		stats:            &DeviceBatchStats{},
	}
	
	bp.batches.Store(deviceID, batch)
	return batch
}

// addRequest adds a request to the device batch
func (db *DeviceBatch) addRequest(request *BatchRequest) {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	
	db.requests = append(db.requests, request)
	
	// Check if batch should be flushed
	shouldFlush := len(db.requests) >= db.currentBatchSize ||
		time.Since(db.lastFlush) >= db.processor.config.BatchTimeout ||
		db.hasHighPriorityRequest() ||
		db.hasDeadlineApproaching()
	
	if shouldFlush {
		db.flush()
	}
}

// hasHighPriorityRequest checks if batch contains high priority requests
func (db *DeviceBatch) hasHighPriorityRequest() bool {
	for _, req := range db.requests {
		if req.Priority > 5 { // High priority threshold
			return true
		}
	}
	return false
}

// hasDeadlineApproaching checks if any request has an approaching deadline
func (db *DeviceBatch) hasDeadlineApproaching() bool {
	now := time.Now()
	for _, req := range db.requests {
		if !req.Deadline.IsZero() && req.Deadline.Sub(now) < db.processor.config.BatchTimeout {
			return true
		}
	}
	return false
}

// flush executes all requests in the batch
func (db *DeviceBatch) flush() {
	if len(db.requests) == 0 {
		return
	}
	
	requests := make([]*BatchRequest, len(db.requests))
	copy(requests, db.requests)
	db.requests = db.requests[:0] // Clear the slice
	db.lastFlush = time.Now()
	
	// Execute batch asynchronously
	go db.executeBatch(requests)
}

// executeBatch executes a batch of requests
func (db *DeviceBatch) executeBatch(requests []*BatchRequest) {
	start := time.Now()
	batchSize := len(requests)
	
	atomic.AddInt64(&db.processor.metrics.TotalBatches, 1)
	atomic.AddInt64(&db.processor.metrics.BatchedRequests, int64(batchSize))
	atomic.AddInt64(&db.stats.TotalBatches, 1)
	
	// Group requests by operation type for optimal batching
	batches := db.groupRequestsByOperation(requests)
	
	var wg sync.WaitGroup
	var successCount, failCount int64
	
	for opType, opRequests := range batches {
		wg.Add(1)
		go func(operation string, reqs []*BatchRequest) {
			defer wg.Done()
			
			results, err := db.executeOperationBatch(operation, reqs)
			
			// Process results
			for i, req := range reqs {
				var result interface{}
				var reqErr error
				
				if err != nil {
					reqErr = err
					atomic.AddInt64(&failCount, 1)
				} else if i < len(results) {
					result = results[i]
					atomic.AddInt64(&successCount, 1)
				}
				
				if req.Callback != nil {
					req.Callback(result, reqErr)
				}
			}
		}(opType, opRequests)
	}
	
	wg.Wait()
	
	// Update metrics
	latency := time.Since(start)
	db.updateMetrics(batchSize, latency, successCount, failCount)
	
	// Adaptive batch size adjustment
	if db.processor.config.EnableAdaptiveBatching {
		db.adjustBatchSize(latency, float64(successCount)/float64(batchSize))
	}
}

// groupRequestsByOperation groups requests by operation type for optimal batching
func (db *DeviceBatch) groupRequestsByOperation(requests []*BatchRequest) map[string][]*BatchRequest {
	groups := make(map[string][]*BatchRequest)
	
	for _, req := range requests {
		opKey := db.getOperationKey(req)
		groups[opKey] = append(groups[opKey], req)
	}
	
	return groups
}

// getOperationKey generates a key for grouping similar operations
func (db *DeviceBatch) getOperationKey(req *BatchRequest) string {
	// Group by operation type and data type for optimal batching
	return req.Operation + ":" + req.DataType
}

// executeOperationBatch executes a batch of similar operations
func (db *DeviceBatch) executeOperationBatch(operation string, requests []*BatchRequest) ([]interface{}, error) {
	// This would integrate with the actual protocol handler
	// For now, simulate batch execution
	
	results := make([]interface{}, len(requests))
	
	// Simulate batch processing time based on operation
	switch operation {
	case "read:holding_register":
		// Batch read holding registers
		results = db.executeBatchRead(requests)
	case "read:input_register":
		// Batch read input registers
		results = db.executeBatchRead(requests)
	case "write:holding_register":
		// Batch write holding registers
		results = db.executeBatchWrite(requests)
	default:
		// Fall back to individual execution
		for i, req := range requests {
			result, err := db.executeSingleRequest(req)
			if err != nil {
				return nil, err
			}
			results[i] = result
		}
	}
	
	return results, nil
}

// executeBatchRead executes a batch read operation
func (db *DeviceBatch) executeBatchRead(requests []*BatchRequest) []interface{} {
	// Optimize by grouping consecutive addresses
	addressGroups := db.groupConsecutiveAddresses(requests)
	results := make([]interface{}, len(requests))
	
	for _, group := range addressGroups {
		// Execute optimized read for consecutive addresses
		groupResults := db.executeConsecutiveRead(group)
		
		// Map results back to original requests
		for i, req := range group {
			if i < len(groupResults) {
				results[db.findRequestIndex(requests, req)] = groupResults[i]
			}
		}
	}
	
	return results
}

// executeBatchWrite executes a batch write operation
func (db *DeviceBatch) executeBatchWrite(requests []*BatchRequest) []interface{} {
	results := make([]interface{}, len(requests))
	
	// Group consecutive addresses for multi-register writes
	addressGroups := db.groupConsecutiveAddresses(requests)
	
	for _, group := range addressGroups {
		// Execute optimized write for consecutive addresses
		groupResults := db.executeConsecutiveWrite(group)
		
		// Map results back to original requests
		for i, req := range group {
			if i < len(groupResults) {
				results[db.findRequestIndex(requests, req)] = groupResults[i]
			}
		}
	}
	
	return results
}

// groupConsecutiveAddresses groups requests with consecutive addresses
func (db *DeviceBatch) groupConsecutiveAddresses(requests []*BatchRequest) [][]*BatchRequest {
	// This would implement intelligent address grouping
	// For now, return individual requests
	groups := make([][]*BatchRequest, len(requests))
	for i, req := range requests {
		groups[i] = []*BatchRequest{req}
	}
	return groups
}

// executeConsecutiveRead executes a read for consecutive addresses
func (db *DeviceBatch) executeConsecutiveRead(requests []*BatchRequest) []interface{} {
	// This would implement optimized consecutive read
	// For now, simulate results
	results := make([]interface{}, len(requests))
	for i := range requests {
		results[i] = float64(i * 100) // Simulated value
	}
	return results
}

// executeConsecutiveWrite executes a write for consecutive addresses
func (db *DeviceBatch) executeConsecutiveWrite(requests []*BatchRequest) []interface{} {
	// This would implement optimized consecutive write
	// For now, simulate results
	results := make([]interface{}, len(requests))
	for i := range requests {
		results[i] = true // Simulated success
	}
	return results
}

// findRequestIndex finds the index of a request in the original slice
func (db *DeviceBatch) findRequestIndex(requests []*BatchRequest, target *BatchRequest) int {
	for i, req := range requests {
		if req == target {
			return i
		}
	}
	return 0
}

// executeSingleRequest executes a single request
func (db *DeviceBatch) executeSingleRequest(req *BatchRequest) (interface{}, error) {
	// This would integrate with the actual protocol handler
	// For now, simulate execution
	time.Sleep(100 * time.Microsecond) // Simulate network latency
	return "simulated_result", nil
}

// updateMetrics updates batch performance metrics
func (db *DeviceBatch) updateMetrics(batchSize int, latency time.Duration, successCount, failCount int64) {
	// Update device statistics
	db.stats.AverageBatchSize = (db.stats.AverageBatchSize + float64(batchSize)) / 2
	db.stats.AverageLatency = (db.stats.AverageLatency + latency) / 2
	
	if successCount > 0 {
		atomic.AddInt64(&db.stats.SuccessfulBatches, 1)
	}
	if failCount > 0 {
		atomic.AddInt64(&db.stats.FailedBatches, 1)
	}
	
	// Update global metrics
	db.processor.metrics.AverageBatchSize = (db.processor.metrics.AverageBatchSize + float64(batchSize)) / 2
	db.processor.metrics.AverageLatency = (db.processor.metrics.AverageLatency + latency) / 2
	
	if successCount > 0 {
		atomic.AddInt64(&db.processor.metrics.BatchesProcessed, 1)
	}
	if failCount > 0 {
		atomic.AddInt64(&db.processor.metrics.BatchesFailed, 1)
	}
}

// adjustBatchSize adjusts the batch size based on performance metrics
func (db *DeviceBatch) adjustBatchSize(latency time.Duration, successRate float64) {
	if latency > db.processor.config.LatencyThreshold {
		// Decrease batch size if latency is too high
		if db.currentBatchSize > db.processor.config.MinBatchSize {
			db.currentBatchSize = int(float64(db.currentBatchSize) * 0.9)
			atomic.AddInt64(&db.processor.metrics.BatchSizeAdjustments, 1)
		}
	} else if successRate > 0.95 && latency < db.processor.config.LatencyThreshold/2 {
		// Increase batch size if performance is good
		if db.currentBatchSize < db.processor.config.MaxBatchSize {
			db.currentBatchSize = int(float64(db.currentBatchSize) * 1.1)
			if db.currentBatchSize > db.processor.config.MaxBatchSize {
				db.currentBatchSize = db.processor.config.MaxBatchSize
			}
			atomic.AddInt64(&db.processor.metrics.BatchSizeAdjustments, 1)
		}
	}
	
	// Update optimal batch size
	atomic.StoreInt32(&db.processor.metrics.OptimalBatchSize, int32(db.currentBatchSize))
	atomic.StoreInt32(&db.stats.OptimalBatchSize, int32(db.currentBatchSize))
}

// executeSingle executes a single request immediately
func (bp *BatchProcessor) executeSingle(request *BatchRequest) {
	go func() {
		result, err := bp.executeSingleRequest(request)
		if request.Callback != nil {
			request.Callback(result, err)
		}
	}()
}

// executeSingleRequest executes a single request
func (bp *BatchProcessor) executeSingleRequest(request *BatchRequest) (interface{}, error) {
	// This would integrate with the actual protocol handler
	// For now, simulate execution
	time.Sleep(50 * time.Microsecond) // Simulate network latency
	return "simulated_result", nil
}

// startFlushTimer starts the background flush timer
func (bp *BatchProcessor) startFlushTimer() {
	bp.timerMutex.Lock()
	defer bp.timerMutex.Unlock()
	
	if bp.flushTimer != nil {
		bp.flushTimer.Stop()
	}
	
	bp.flushTimer = time.AfterFunc(bp.config.FlushInterval, bp.flushAllBatches)
}

// flushAllBatches flushes all pending batches
func (bp *BatchProcessor) flushAllBatches() {
	bp.batches.Range(func(key, value interface{}) bool {
		batch := value.(*DeviceBatch)
		batch.mutex.Lock()
		if len(batch.requests) > 0 {
			batch.flush()
		}
		batch.mutex.Unlock()
		return true
	})
	
	// Restart the timer
	bp.startFlushTimer()
}

// GetMetrics returns batch processor metrics
func (bp *BatchProcessor) GetMetrics() *BatchMetrics {
	return bp.metrics
}

// GetDeviceStats returns statistics for a specific device
func (bp *BatchProcessor) GetDeviceStats(deviceID string) *DeviceBatchStats {
	if batch, exists := bp.batches.Load(deviceID); exists {
		return batch.(*DeviceBatch).stats
	}
	return nil
}

// Close gracefully shuts down the batch processor
func (bp *BatchProcessor) Close() error {
	bp.cancel()
	
	if bp.flushTimer != nil {
		bp.flushTimer.Stop()
	}
	
	// Flush all pending batches
	bp.flushAllBatches()
	
	return nil
}