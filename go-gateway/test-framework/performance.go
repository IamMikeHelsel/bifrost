package testframework

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// PerformanceTest represents a single performance test
type PerformanceTest struct {
	Name              string
	Target            string
	Setup             func() error
	Operation         func(ctx context.Context) error
	Teardown          func() error
	ConcurrencyLevels []int
	DurationPerLevel  time.Duration
	WarmupDuration    time.Duration
}

// PerformanceMetrics contains performance measurements
type PerformanceMetrics struct {
	TestName          string
	ConcurrencyLevel  int
	TotalOperations   int64
	SuccessfulOps     int64
	FailedOps         int64
	Duration          time.Duration
	OperationsPerSec  float64
	AvgLatency        time.Duration
	P50Latency        time.Duration
	P95Latency        time.Duration
	P99Latency        time.Duration
	MinLatency        time.Duration
	MaxLatency        time.Duration
	ErrorRate         float64
	ThroughputMBps    float64
}

// BaselineData stores historical performance data for comparison
type BaselineData struct {
	TestName         string
	ConcurrencyLevel int
	Timestamp        time.Time
	OperationsPerSec float64
	P99Latency       time.Duration
	ErrorRate        float64
}

// PerformanceSuite manages performance tests
type PerformanceSuite struct {
	logger    *zap.Logger
	tests     []PerformanceTest
	baselines map[string]BaselineData
	mu        sync.RWMutex
}

// NewPerformanceSuite creates a new performance testing suite
func NewPerformanceSuite(logger *zap.Logger) *PerformanceSuite {
	return &PerformanceSuite{
		logger:    logger,
		tests:     make([]PerformanceTest, 0),
		baselines: make(map[string]BaselineData),
	}
}

// AddTest adds a performance test to the suite
func (ps *PerformanceSuite) AddTest(test PerformanceTest) {
	ps.tests = append(ps.tests, test)
}

// RunAll executes all performance tests
func (ps *PerformanceSuite) RunAll(ctx context.Context) ([]PerformanceMetrics, error) {
	var allResults []PerformanceMetrics

	for _, test := range ps.tests {
		ps.logger.Info("Starting performance test", zap.String("test", test.Name))

		results, err := ps.runSingleTest(ctx, test)
		if err != nil {
			ps.logger.Error("Performance test failed", zap.String("test", test.Name), zap.Error(err))
			continue
		}

		allResults = append(allResults, results...)

		// Check for regressions
		for _, result := range results {
			ps.checkRegression(result)
		}
	}

	return allResults, nil
}

// runSingleTest executes a single performance test at multiple concurrency levels
func (ps *PerformanceSuite) runSingleTest(ctx context.Context, test PerformanceTest) ([]PerformanceMetrics, error) {
	var results []PerformanceMetrics

	// Setup
	if test.Setup != nil {
		if err := test.Setup(); err != nil {
			return nil, fmt.Errorf("test setup failed: %w", err)
		}
	}

	// Cleanup
	defer func() {
		if test.Teardown != nil {
			test.Teardown()
		}
	}()

	// Run test at each concurrency level
	for _, concurrency := range test.ConcurrencyLevels {
		ps.logger.Info("Running test at concurrency level", 
			zap.String("test", test.Name), 
			zap.Int("concurrency", concurrency))

		metrics, err := ps.runAtConcurrencyLevel(ctx, test, concurrency)
		if err != nil {
			ps.logger.Error("Test failed at concurrency level", 
				zap.String("test", test.Name), 
				zap.Int("concurrency", concurrency), 
				zap.Error(err))
			continue
		}

		results = append(results, metrics)
	}

	return results, nil
}

// runAtConcurrencyLevel runs a test at a specific concurrency level
func (ps *PerformanceSuite) runAtConcurrencyLevel(ctx context.Context, test PerformanceTest, concurrency int) (PerformanceMetrics, error) {
	metrics := PerformanceMetrics{
		TestName:         test.Name,
		ConcurrencyLevel: concurrency,
	}

	// Warmup phase
	if test.WarmupDuration > 0 {
		ps.logger.Debug("Warmup phase", zap.Duration("duration", test.WarmupDuration))
		warmupCtx, cancel := context.WithTimeout(ctx, test.WarmupDuration)
		ps.runLoadTest(warmupCtx, test, concurrency/2, nil) // Half concurrency for warmup
		cancel()
	}

	// Actual test phase
	latencies := make([]time.Duration, 0, 10000)
	var latencyMutex sync.Mutex

	collector := func(latency time.Duration) {
		latencyMutex.Lock()
		latencies = append(latencies, latency)
		latencyMutex.Unlock()
	}

	testCtx, cancel := context.WithTimeout(ctx, test.DurationPerLevel)
	defer cancel()

	startTime := time.Now()
	successCount, errorCount := ps.runLoadTest(testCtx, test, concurrency, collector)
	duration := time.Since(startTime)

	// Calculate metrics
	metrics.Duration = duration
	metrics.TotalOperations = successCount + errorCount
	metrics.SuccessfulOps = successCount
	metrics.FailedOps = errorCount
	metrics.OperationsPerSec = float64(metrics.TotalOperations) / duration.Seconds()
	if metrics.TotalOperations > 0 {
		metrics.ErrorRate = float64(errorCount) / float64(metrics.TotalOperations) * 100
	}

	// Calculate latency statistics
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		metrics.MinLatency = latencies[0]
		metrics.MaxLatency = latencies[len(latencies)-1]
		metrics.P50Latency = latencies[len(latencies)*50/100]
		metrics.P95Latency = latencies[len(latencies)*95/100]
		metrics.P99Latency = latencies[len(latencies)*99/100]

		// Calculate average latency
		var totalLatency time.Duration
		for _, lat := range latencies {
			totalLatency += lat
		}
		metrics.AvgLatency = totalLatency / time.Duration(len(latencies))
	}

	ps.logger.Info("Performance test completed",
		zap.String("test", test.Name),
		zap.Int("concurrency", concurrency),
		zap.Int64("operations", metrics.TotalOperations),
		zap.Float64("ops_per_sec", metrics.OperationsPerSec),
		zap.Duration("p99_latency", metrics.P99Latency),
		zap.Float64("error_rate", metrics.ErrorRate))

	return metrics, nil
}

// runLoadTest executes the load test with specified concurrency
func (ps *PerformanceSuite) runLoadTest(ctx context.Context, test PerformanceTest, concurrency int, latencyCollector func(time.Duration)) (int64, int64) {
	var successCount, errorCount int64
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				startTime := time.Now()
				err := test.Operation(ctx)
				latency := time.Since(startTime)

				if latencyCollector != nil {
					latencyCollector(latency)
				}

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()
	return atomic.LoadInt64(&successCount), atomic.LoadInt64(&errorCount)
}

// SetBaseline stores performance baseline data
func (ps *PerformanceSuite) SetBaseline(testName string, concurrency int, opsPerSec float64, p99Latency time.Duration, errorRate float64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	key := fmt.Sprintf("%s-%d", testName, concurrency)
	ps.baselines[key] = BaselineData{
		TestName:         testName,
		ConcurrencyLevel: concurrency,
		Timestamp:        time.Now(),
		OperationsPerSec: opsPerSec,
		P99Latency:       p99Latency,
		ErrorRate:        errorRate,
	}
}

// checkRegression compares current results against baseline
func (ps *PerformanceSuite) checkRegression(metrics PerformanceMetrics) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	key := fmt.Sprintf("%s-%d", metrics.TestName, metrics.ConcurrencyLevel)
	baseline, exists := ps.baselines[key]
	if !exists {
		ps.logger.Info("No baseline found, storing current metrics as baseline",
			zap.String("test", metrics.TestName),
			zap.Int("concurrency", metrics.ConcurrencyLevel))
		
		// Store current metrics as baseline
		go func() {
			ps.SetBaseline(metrics.TestName, metrics.ConcurrencyLevel, 
				metrics.OperationsPerSec, metrics.P99Latency, metrics.ErrorRate)
		}()
		return
	}

	// Check for performance regressions
	var regressions []string

	// Throughput regression (>10% decrease)
	throughputChange := (metrics.OperationsPerSec - baseline.OperationsPerSec) / baseline.OperationsPerSec * 100
	if throughputChange < -10 {
		regressions = append(regressions, fmt.Sprintf("throughput decreased by %.1f%%", -throughputChange))
	}

	// Latency regression (>20% increase)
	latencyChange := float64(metrics.P99Latency-baseline.P99Latency) / float64(baseline.P99Latency) * 100
	if latencyChange > 20 {
		regressions = append(regressions, fmt.Sprintf("P99 latency increased by %.1f%%", latencyChange))
	}

	// Error rate regression (>5% increase)
	errorRateChange := metrics.ErrorRate - baseline.ErrorRate
	if errorRateChange > 5 {
		regressions = append(regressions, fmt.Sprintf("error rate increased by %.1f%%", errorRateChange))
	}

	if len(regressions) > 0 {
		ps.logger.Warn("Performance regression detected",
			zap.String("test", metrics.TestName),
			zap.Int("concurrency", metrics.ConcurrencyLevel),
			zap.Strings("regressions", regressions))
	} else {
		ps.logger.Info("Performance test passed regression checks",
			zap.String("test", metrics.TestName),
			zap.Int("concurrency", metrics.ConcurrencyLevel))
	}
}

// CreateStandardPerformanceTests creates a standard set of performance tests
func CreateStandardPerformanceTests() []PerformanceTest {
	return []PerformanceTest{
		{
			Name:              "Modbus Read Performance",
			Target:            "modbus-read",
			ConcurrencyLevels: []int{1, 10, 50, 100, 500},
			DurationPerLevel:  30 * time.Second,
			WarmupDuration:    5 * time.Second,
			Operation: func(ctx context.Context) error {
				// Simulate Modbus read operation
				time.Sleep(time.Millisecond * time.Duration(1+randomUint32()%5))
				
				// Simulate occasional errors (1% error rate)
				if randomUint32()%100 == 0 {
					return fmt.Errorf("simulated modbus read error")
				}
				return nil
			},
		},
		{
			Name:              "Modbus Write Performance",
			Target:            "modbus-write",
			ConcurrencyLevels: []int{1, 10, 50, 100},
			DurationPerLevel:  30 * time.Second,
			WarmupDuration:    5 * time.Second,
			Operation: func(ctx context.Context) error {
				// Simulate Modbus write operation (typically slower)
				time.Sleep(time.Millisecond * time.Duration(2+randomUint32()%8))
				
				// Simulate occasional errors (2% error rate for writes)
				if randomUint32()%50 == 0 {
					return fmt.Errorf("simulated modbus write error")
				}
				return nil
			},
		},
		{
			Name:              "EtherNet/IP Read Performance",
			Target:            "ethernetip-read",
			ConcurrencyLevels: []int{1, 10, 50, 100, 200},
			DurationPerLevel:  30 * time.Second,
			WarmupDuration:    5 * time.Second,
			Operation: func(ctx context.Context) error {
				// Simulate EtherNet/IP read operation
				time.Sleep(time.Microsecond * time.Duration(500+randomUint32()%1000))
				
				// Simulate occasional errors (0.5% error rate)
				if randomUint32()%200 == 0 {
					return fmt.Errorf("simulated ethernetip read error")
				}
				return nil
			},
		},
		{
			Name:              "Connection Pool Performance",
			Target:            "connection-pool",
			ConcurrencyLevels: []int{10, 50, 100, 500, 1000},
			DurationPerLevel:  20 * time.Second,
			WarmupDuration:    3 * time.Second,
			Operation: func(ctx context.Context) error {
				// Simulate connection acquisition and release
				time.Sleep(time.Microsecond * time.Duration(100+randomUint32()%200))
				return nil
			},
		},
		{
			Name:              "Memory Allocation Performance",
			Target:            "memory-allocation",
			ConcurrencyLevels: []int{1, 10, 50, 100},
			DurationPerLevel:  15 * time.Second,
			WarmupDuration:    2 * time.Second,
			Operation: func(ctx context.Context) error {
				// Simulate memory-intensive operations
				data := make([]byte, 1024+randomUint32()%4096)
				_ = data
				time.Sleep(time.Microsecond * time.Duration(50+randomUint32()%100))
				return nil
			},
		},
	}
}

// StressTest represents a stress testing scenario
type StressTest struct {
	Name            string
	Target          string
	InitialLoad     int
	MaxLoad         int
	LoadIncrement   int
	StepDuration    time.Duration
	Operation       func(ctx context.Context) error
	BreakCondition  func(metrics PerformanceMetrics) bool
}

// RunStressTest executes a stress test that gradually increases load
func (ps *PerformanceSuite) RunStressTest(ctx context.Context, test StressTest) ([]PerformanceMetrics, error) {
	var results []PerformanceMetrics

	ps.logger.Info("Starting stress test", zap.String("test", test.Name))

	for load := test.InitialLoad; load <= test.MaxLoad; load += test.LoadIncrement {
		ps.logger.Info("Stress test step", 
			zap.String("test", test.Name), 
			zap.Int("load", load))

		perfTest := PerformanceTest{
			Name:              fmt.Sprintf("%s-load-%d", test.Name, load),
			ConcurrencyLevels: []int{load},
			DurationPerLevel:  test.StepDuration,
			Operation:         test.Operation,
		}

		stepResults, err := ps.runAtConcurrencyLevel(ctx, perfTest, load)
		if err != nil {
			ps.logger.Error("Stress test step failed", 
				zap.String("test", test.Name), 
				zap.Int("load", load), 
				zap.Error(err))
			break
		}

		results = append(results, stepResults)

		// Check break condition
		if test.BreakCondition != nil && test.BreakCondition(stepResults) {
			ps.logger.Info("Stress test break condition met", 
				zap.String("test", test.Name), 
				zap.Int("breaking_load", load))
			break
		}

		// Brief pause between steps
		time.Sleep(1 * time.Second)
	}

	return results, nil
}

// CreateStandardStressTests creates standard stress tests
func CreateStandardStressTests() []StressTest {
	return []StressTest{
		{
			Name:          "Modbus Connection Stress",
			Target:        "modbus-connections",
			InitialLoad:   10,
			MaxLoad:       1000,
			LoadIncrement: 50,
			StepDuration:  15 * time.Second,
			Operation: func(ctx context.Context) error {
				// Simulate connection establishment and basic operation
				time.Sleep(time.Millisecond * time.Duration(5+randomUint32()%10))
				if randomUint32()%1000 == 0 {
					return fmt.Errorf("connection failed")
				}
				return nil
			},
			BreakCondition: func(metrics PerformanceMetrics) bool {
				// Break if error rate exceeds 5% or P99 latency exceeds 100ms
				return metrics.ErrorRate > 5.0 || metrics.P99Latency > 100*time.Millisecond
			},
		},
		{
			Name:          "Memory Usage Stress",
			Target:        "memory-usage",
			InitialLoad:   10,
			MaxLoad:       500,
			LoadIncrement: 25,
			StepDuration:  10 * time.Second,
			Operation: func(ctx context.Context) error {
				// Allocate and hold memory to stress test memory management
				data := make([]byte, 10240) // 10KB per operation
				time.Sleep(time.Millisecond * time.Duration(100+randomUint32()%200))
				_ = data
				return nil
			},
			BreakCondition: func(metrics PerformanceMetrics) bool {
				// Break if operations per second drops below 50% of initial
				return metrics.OperationsPerSec < 50
			},
		},
	}
}