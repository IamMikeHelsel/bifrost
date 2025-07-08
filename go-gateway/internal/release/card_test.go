package release

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"bifrost-gateway/internal/performance"
)

func TestReleaseCard_Basic(t *testing.T) {
	card := &ReleaseCard{
		Metadata: CardMetadata{
			Version:     "v1.0.0",
			ReleaseDate: time.Now(),
			GitCommit:   "abc123",
			Environment: "test",
			GeneratedAt: time.Now(),
		},
		PerformanceBenchmarks: PerformanceBenchmarks{
			ModbusTCP: &ProtocolBenchmark{
				Throughput: Throughput{
					SingleConnection: ThroughputMetric{
						Target:   ">1000 regs/sec",
						Measured: "1247 regs/sec",
						Status:   "pass",
						Value:    1247,
					},
				},
				Latency: Latency{
					SingleRegister: LatencyMetric{
						Target:   "<1ms",
						Measured: "0.7ms",
						Status:   "pass",
						Value:    700 * time.Microsecond,
					},
				},
				Resources: Resources{
					MemoryUsage: "12MB",
					CPUUsage:    "5%",
					Connections: "100",
				},
			},
		},
		Summary: CardSummary{
			OverallStatus:    "pass",
			PerformanceScore: 85.0,
			ProtocolsSupported: 1,
		},
	}

	// Test validation
	err := card.Validate()
	assert.NoError(t, err)

	// Test YAML serialization
	yamlData, err := card.ToYAML()
	require.NoError(t, err)
	assert.Contains(t, string(yamlData), "v1.0.0")
	assert.Contains(t, string(yamlData), "modbus_tcp")

	// Test JSON serialization
	jsonData, err := card.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "v1.0.0")
	assert.Contains(t, string(jsonData), "modbus_tcp")

	// Test deserialization
	card2 := &ReleaseCard{}
	err = card2.FromYAML(yamlData)
	require.NoError(t, err)
	assert.Equal(t, card.Metadata.Version, card2.Metadata.Version)

	card3 := &ReleaseCard{}
	err = card3.FromJSON(jsonData)
	require.NoError(t, err)
	assert.Equal(t, card.Metadata.Version, card3.Metadata.Version)
}

func TestReleaseCard_Validation(t *testing.T) {
	tests := []struct {
		name        string
		card        *ReleaseCard
		expectError bool
	}{
		{
			name: "valid card",
			card: &ReleaseCard{
				Metadata: CardMetadata{
					Version:     "v1.0.0",
					ReleaseDate: time.Now(),
				},
				PerformanceBenchmarks: PerformanceBenchmarks{
					ModbusTCP: &ProtocolBenchmark{},
				},
			},
			expectError: false,
		},
		{
			name: "missing version",
			card: &ReleaseCard{
				Metadata: CardMetadata{
					ReleaseDate: time.Now(),
				},
				PerformanceBenchmarks: PerformanceBenchmarks{
					ModbusTCP: &ProtocolBenchmark{},
				},
			},
			expectError: true,
		},
		{
			name: "missing release date",
			card: &ReleaseCard{
				Metadata: CardMetadata{
					Version: "v1.0.0",
				},
				PerformanceBenchmarks: PerformanceBenchmarks{
					ModbusTCP: &ProtocolBenchmark{},
				},
			},
			expectError: true,
		},
		{
			name: "missing performance data",
			card: &ReleaseCard{
				Metadata: CardMetadata{
					Version:     "v1.0.0",
					ReleaseDate: time.Now(),
				},
				PerformanceBenchmarks: PerformanceBenchmarks{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.card.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBenchmarkIntegrator_Basic(t *testing.T) {
	logger := zap.NewNop()
	
	// Create mock benchmark suite
	benchmarkConfig := &performance.BenchmarkConfig{
		EnableLatencyTests:    true,
		EnableThroughputTests: true,
		WarmupDuration:        time.Second,
		TestDuration:          time.Second,
	}
	
	targets := &performance.PerformanceTargets{
		MaxLatencyMicroseconds: 1000,
		MinThroughputOpsPerSec: 1000,
	}
	
	benchmarkSuite := performance.NewBenchmarkSuite(benchmarkConfig, targets, logger)
	
	// Create integrator
	integrator := NewBenchmarkIntegrator(logger, benchmarkSuite, nil, nil)
	
	// Test basic functionality (this would normally run actual benchmarks)
	assert.NotNil(t, integrator)
	assert.NotNil(t, integrator.config)
	assert.Equal(t, DefaultIntegratorConfig().EnableTrendAnalysis, integrator.config.EnableTrendAnalysis)
}

func TestStorageProvider_InMemory(t *testing.T) {
	storage := NewInMemoryStorageProvider()
	
	// Create test card
	card := &ReleaseCard{
		Metadata: CardMetadata{
			Version:     "v1.0.0",
			ReleaseDate: time.Now(),
			GitCommit:   "abc123",
			Environment: "test",
			GeneratedAt: time.Now(),
		},
		PerformanceBenchmarks: PerformanceBenchmarks{
			ModbusTCP: &ProtocolBenchmark{
				Throughput: Throughput{
					SingleConnection: ThroughputMetric{
						Target:   ">1000 regs/sec",
						Measured: "1247 regs/sec",
						Status:   "pass",
						Value:    1247,
					},
				},
			},
		},
		Summary: CardSummary{
			OverallStatus:    "pass",
			PerformanceScore: 85.0,
		},
	}
	
	// Test store
	err := storage.Store(card, "json")
	assert.NoError(t, err)
	
	// Test load
	loadedCard, err := storage.Load("v1.0.0")
	require.NoError(t, err)
	assert.Equal(t, card.Metadata.Version, loadedCard.Metadata.Version)
	assert.Equal(t, card.Summary.PerformanceScore, loadedCard.Summary.PerformanceScore)
	
	// Test list
	cards, err := storage.List()
	require.NoError(t, err)
	assert.Len(t, cards, 1)
	assert.Equal(t, "v1.0.0", cards[0].Metadata.Version)
	
	// Test delete
	err = storage.Delete("v1.0.0")
	assert.NoError(t, err)
	
	_, err = storage.Load("v1.0.0")
	assert.Error(t, err)
	
	cards, err = storage.List()
	require.NoError(t, err)
	assert.Len(t, cards, 0)
}

func TestTrendAnalyzer_Basic(t *testing.T) {
	logger := zap.NewNop()
	analyzer := NewTrendAnalyzer(logger)
	
	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.logger)
}

func TestAutomationEngine_QualityGates(t *testing.T) {
	logger := zap.NewNop()
	storage := NewInMemoryStorageProvider()
	
	config := &AutomationConfig{
		MinPerformanceScore: 80.0,
		BlockOnRegression:   true,
		RequireTestCoverage: 85.0,
	}
	
	// Create mock integrator
	benchmarkConfig := &performance.BenchmarkConfig{
		EnableLatencyTests:    true,
		EnableThroughputTests: true,
		WarmupDuration:        time.Second,
		TestDuration:          time.Second,
	}
	
	targets := &performance.PerformanceTargets{
		MaxLatencyMicroseconds: 1000,
		MinThroughputOpsPerSec: 1000,
	}
	
	benchmarkSuite := performance.NewBenchmarkSuite(benchmarkConfig, targets, logger)
	integrator := NewBenchmarkIntegrator(logger, benchmarkSuite, nil, nil)
	
	engine := NewAutomationEngine(logger, integrator, storage, config)
	
	tests := []struct {
		name        string
		card        *ReleaseCard
		expectError bool
	}{
		{
			name: "passes quality gates",
			card: &ReleaseCard{
				Summary: CardSummary{
					PerformanceScore:      85.0,
					PerformanceRegression: false,
					TestCoverage:          90.0,
				},
			},
			expectError: false,
		},
		{
			name: "fails performance score",
			card: &ReleaseCard{
				Summary: CardSummary{
					PerformanceScore:      70.0,
					PerformanceRegression: false,
					TestCoverage:          90.0,
				},
			},
			expectError: true,
		},
		{
			name: "fails regression check",
			card: &ReleaseCard{
				Summary: CardSummary{
					PerformanceScore:      85.0,
					PerformanceRegression: true,
					TestCoverage:          90.0,
				},
			},
			expectError: true,
		},
		{
			name: "fails test coverage",
			card: &ReleaseCard{
				Summary: CardSummary{
					PerformanceScore:      85.0,
					PerformanceRegression: false,
					TestCoverage:          80.0,
				},
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.applyQualityGates(tt.card)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractNumericValue(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{">1000 regs/sec", 1000},
		{"<5ms", 5},
		{">50000 regs/sec", 50000},
		{"100", 100},
		{"invalid", 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractNumericValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractDurationValue(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"<1ms", time.Millisecond},
		{">5ms", 5 * time.Millisecond},
		{"100us", 100 * time.Microsecond},
		{"1s", time.Second},
		{"invalid", 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractDurationValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}