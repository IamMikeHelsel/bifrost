package release

import (
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
)

// TrendAnalyzer analyzes performance trends across releases
type TrendAnalyzer struct {
	logger *zap.Logger
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer(logger *zap.Logger) *TrendAnalyzer {
	return &TrendAnalyzer{
		logger: logger,
	}
}

// generateTrendAnalysis generates trend analysis for the current release
func (bi *BenchmarkIntegrator) generateTrendAnalysis(currentCard *ReleaseCard) (*TrendAnalysis, error) {
	if len(bi.previousResults) < bi.config.MinHistoryForTrends {
		return nil, fmt.Errorf("insufficient historical data for trend analysis (need at least %d, have %d)", 
			bi.config.MinHistoryForTrends, len(bi.previousResults))
	}
	
	// Get the most recent previous result for comparison
	previousCard := bi.previousResults[len(bi.previousResults)-1]
	
	// Calculate comparison metrics
	comparison, err := bi.calculateComparison(currentCard, previousCard)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate comparison metrics: %w", err)
	}
	
	// Analyze trends
	latencyTrend := bi.analyzeTrend("latency", bi.getLatencyHistory(), comparison.LatencyChange)
	throughputTrend := bi.analyzeTrend("throughput", bi.getThroughputHistory(), comparison.ThroughputChange)
	memoryTrend := bi.analyzeTrend("memory", bi.getMemoryHistory(), comparison.MemoryChange)
	
	return &TrendAnalysis{
		LatencyTrend:         latencyTrend,
		ThroughputTrend:      throughputTrend,
		MemoryTrend:          memoryTrend,
		ComparisonToPrevious: *comparison,
	}, nil
}

// calculateComparison calculates comparison metrics between two release cards
func (bi *BenchmarkIntegrator) calculateComparison(current, previous *ReleaseCard) (*ComparisonMetrics, error) {
	// Extract primary protocol metrics for comparison (using Modbus TCP as primary)
	currentProtocol := current.PerformanceBenchmarks.ModbusTCP
	previousProtocol := previous.PerformanceBenchmarks.ModbusTCP
	
	if currentProtocol == nil || previousProtocol == nil {
		return &ComparisonMetrics{}, nil // Return empty metrics if no comparable data
	}
	
	// Calculate latency change
	latencyChange := 0.0
	if currentProtocol.Latency.SingleRegister.Value > 0 && previousProtocol.Latency.SingleRegister.Value > 0 {
		latencyChange = ((float64(currentProtocol.Latency.SingleRegister.Value) - 
			float64(previousProtocol.Latency.SingleRegister.Value)) / 
			float64(previousProtocol.Latency.SingleRegister.Value)) * 100
	}
	
	// Calculate throughput change
	throughputChange := 0.0
	if currentProtocol.Throughput.SingleConnection.Value > 0 && previousProtocol.Throughput.SingleConnection.Value > 0 {
		throughputChange = ((currentProtocol.Throughput.SingleConnection.Value - 
			previousProtocol.Throughput.SingleConnection.Value) / 
			previousProtocol.Throughput.SingleConnection.Value) * 100
	}
	
	// Calculate performance score change
	scoreChange := ((current.Summary.PerformanceScore - previous.Summary.PerformanceScore) / 
		previous.Summary.PerformanceScore) * 100
	
	return &ComparisonMetrics{
		LatencyChange:    latencyChange,
		ThroughputChange: throughputChange,
		MemoryChange:     0.0, // Would need to extract from resource metrics
		ScoreChange:      scoreChange,
	}, nil
}

// analyzeTrend analyzes the trend for a specific metric
func (bi *BenchmarkIntegrator) analyzeTrend(metricName string, history []float64, recentChange float64) string {
	if len(history) < 3 {
		return "insufficient_data"
	}
	
	// Calculate trend using linear regression slope
	slope := bi.calculateTrendSlope(history)
	
	// Determine trend based on slope and recent change
	const stableThreshold = 2.0 // 2% threshold for stability
	
	if math.Abs(slope) < stableThreshold && math.Abs(recentChange) < stableThreshold {
		return "stable"
	}
	
	if slope > 0 && recentChange > 0 {
		if metricName == "latency" || metricName == "memory" {
			return "degrading" // Higher latency/memory is worse
		}
		return "improving" // Higher throughput is better
	}
	
	if slope < 0 && recentChange < 0 {
		if metricName == "latency" || metricName == "memory" {
			return "improving" // Lower latency/memory is better
		}
		return "degrading" // Lower throughput is worse
	}
	
	return "volatile" // Mixed signals
}

// calculateTrendSlope calculates the slope of a linear trend through the data points
func (bi *BenchmarkIntegrator) calculateTrendSlope(values []float64) float64 {
	n := len(values)
	if n < 2 {
		return 0
	}
	
	// Create x values (time points)
	xSum := 0.0
	ySum := 0.0
	xySum := 0.0
	x2Sum := 0.0
	
	for i, y := range values {
		x := float64(i)
		xSum += x
		ySum += y
		xySum += x * y
		x2Sum += x * x
	}
	
	// Calculate slope using least squares method
	denominator := float64(n)*x2Sum - xSum*xSum
	if denominator == 0 {
		return 0
	}
	
	slope := (float64(n)*xySum - xSum*ySum) / denominator
	return slope
}

// getLatencyHistory extracts latency values from historical results
func (bi *BenchmarkIntegrator) getLatencyHistory() []float64 {
	history := make([]float64, 0, len(bi.previousResults))
	
	for _, result := range bi.previousResults {
		if result.PerformanceBenchmarks.ModbusTCP != nil &&
			result.PerformanceBenchmarks.ModbusTCP.Latency.SingleRegister.Value > 0 {
			latencyMs := float64(result.PerformanceBenchmarks.ModbusTCP.Latency.SingleRegister.Value.Nanoseconds()) / 1e6
			history = append(history, latencyMs)
		}
	}
	
	return history
}

// getThroughputHistory extracts throughput values from historical results
func (bi *BenchmarkIntegrator) getThroughputHistory() []float64 {
	history := make([]float64, 0, len(bi.previousResults))
	
	for _, result := range bi.previousResults {
		if result.PerformanceBenchmarks.ModbusTCP != nil &&
			result.PerformanceBenchmarks.ModbusTCP.Throughput.SingleConnection.Value > 0 {
			history = append(history, result.PerformanceBenchmarks.ModbusTCP.Throughput.SingleConnection.Value)
		}
	}
	
	return history
}

// getMemoryHistory extracts memory usage values from historical results
func (bi *BenchmarkIntegrator) getMemoryHistory() []float64 {
	history := make([]float64, 0, len(bi.previousResults))
	
	for _, result := range bi.previousResults {
		if result.PerformanceBenchmarks.ModbusTCP != nil {
			// Parse memory usage from string (e.g., "12.5MB" -> 12.5)
			memoryStr := result.PerformanceBenchmarks.ModbusTCP.Resources.MemoryUsage
			if memoryStr != "" && memoryStr != "not measured" {
				var memoryValue float64
				if _, err := fmt.Sscanf(memoryStr, "%fMB", &memoryValue); err == nil {
					history = append(history, memoryValue)
				}
			}
		}
	}
	
	return history
}

// detectRegressions detects performance regressions in the current release
func (bi *BenchmarkIntegrator) detectRegressions(currentCard *ReleaseCard) bool {
	if len(bi.previousResults) == 0 {
		return false // No baseline to compare against
	}
	
	// Compare against the most recent previous result
	previousCard := bi.previousResults[len(bi.previousResults)-1]
	
	regressions := make([]string, 0)
	
	// Check for performance score regression
	scoreChange := ((currentCard.Summary.PerformanceScore - previousCard.Summary.PerformanceScore) /
		previousCard.Summary.PerformanceScore) * 100
	
	if scoreChange < -bi.config.RegressionThreshold {
		regressions = append(regressions, fmt.Sprintf("Overall performance score decreased by %.1f%%", -scoreChange))
	}
	
	// Check protocol-specific regressions
	if bi.detectProtocolRegression(currentCard.PerformanceBenchmarks.ModbusTCP, 
		previousCard.PerformanceBenchmarks.ModbusTCP, "Modbus TCP") {
		regressions = append(regressions, "Modbus TCP performance regression detected")
	}
	
	if bi.detectProtocolRegression(currentCard.PerformanceBenchmarks.EtherNetIP, 
		previousCard.PerformanceBenchmarks.EtherNetIP, "EtherNet/IP") {
		regressions = append(regressions, "EtherNet/IP performance regression detected")
	}
	
	// Log regressions
	if len(regressions) > 0 {
		bi.logger.Warn("Performance regressions detected",
			zap.Strings("regressions", regressions),
			zap.String("version", currentCard.Metadata.Version),
		)
		
		// Add to known issues
		if currentCard.Summary.KnownIssues == nil {
			currentCard.Summary.KnownIssues = make([]string, 0)
		}
		currentCard.Summary.KnownIssues = append(currentCard.Summary.KnownIssues, regressions...)
		
		return true
	}
	
	return false
}

// detectProtocolRegression checks for regressions in a specific protocol
func (bi *BenchmarkIntegrator) detectProtocolRegression(current, previous *ProtocolBenchmark, protocolName string) bool {
	if current == nil || previous == nil {
		return false
	}
	
	// Check latency regression (higher latency is worse)
	if current.Latency.SingleRegister.Value > 0 && previous.Latency.SingleRegister.Value > 0 {
		latencyChange := ((float64(current.Latency.SingleRegister.Value) - 
			float64(previous.Latency.SingleRegister.Value)) / 
			float64(previous.Latency.SingleRegister.Value)) * 100
		
		if latencyChange > bi.config.RegressionThreshold {
			bi.logger.Warn("Latency regression detected",
				zap.String("protocol", protocolName),
				zap.Float64("change_percent", latencyChange),
			)
			return true
		}
	}
	
	// Check throughput regression (lower throughput is worse)
	if current.Throughput.SingleConnection.Value > 0 && previous.Throughput.SingleConnection.Value > 0 {
		throughputChange := ((current.Throughput.SingleConnection.Value - 
			previous.Throughput.SingleConnection.Value) / 
			previous.Throughput.SingleConnection.Value) * 100
		
		if throughputChange < -bi.config.RegressionThreshold {
			bi.logger.Warn("Throughput regression detected",
				zap.String("protocol", protocolName),
				zap.Float64("change_percent", throughputChange),
			)
			return true
		}
	}
	
	return false
}

// LoadHistoricalData loads historical release card data for trend analysis
func (bi *BenchmarkIntegrator) LoadHistoricalData(cards []*ReleaseCard) {
	// Filter and sort historical data
	filtered := make([]*ReleaseCard, 0, len(cards))
	retentionCutoff := time.Now().AddDate(0, 0, -bi.config.HistoryRetentionDays)
	
	for _, card := range cards {
		if card.Metadata.ReleaseDate.After(retentionCutoff) {
			filtered = append(filtered, card)
		}
	}
	
	// Sort by release date
	for i := 0; i < len(filtered)-1; i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[i].Metadata.ReleaseDate.After(filtered[j].Metadata.ReleaseDate) {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}
	
	bi.previousResults = filtered
	bi.logger.Info("Loaded historical release card data",
		zap.Int("count", len(filtered)),
		zap.Int("retention_days", bi.config.HistoryRetentionDays),
	)
}