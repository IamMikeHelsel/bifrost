package release

import (
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// ReleaseCard represents a complete release card with all testing results
type ReleaseCard struct {
	Metadata        CardMetadata         `yaml:"metadata" json:"metadata"`
	PerformanceBenchmarks PerformanceBenchmarks `yaml:"performance_benchmarks" json:"performance_benchmarks"`
	TestResults     TestResults          `yaml:"test_results,omitempty" json:"test_results,omitempty"`
	DeviceRegistry  DeviceRegistry       `yaml:"device_registry,omitempty" json:"device_registry,omitempty"`
	Summary         CardSummary          `yaml:"summary" json:"summary"`
}

// CardMetadata holds release card metadata
type CardMetadata struct {
	Version     string    `yaml:"version" json:"version"`
	ReleaseDate time.Time `yaml:"release_date" json:"release_date"`
	GitCommit   string    `yaml:"git_commit" json:"git_commit"`
	BuildNumber string    `yaml:"build_number,omitempty" json:"build_number,omitempty"`
	Environment string    `yaml:"environment" json:"environment"`
	GeneratedAt time.Time `yaml:"generated_at" json:"generated_at"`
}

// PerformanceBenchmarks holds protocol-specific performance data
type PerformanceBenchmarks struct {
	ModbusTCP   *ProtocolBenchmark `yaml:"modbus_tcp,omitempty" json:"modbus_tcp,omitempty"`
	ModbusRTU   *ProtocolBenchmark `yaml:"modbus_rtu,omitempty" json:"modbus_rtu,omitempty"`
	EtherNetIP  *ProtocolBenchmark `yaml:"ethernet_ip,omitempty" json:"ethernet_ip,omitempty"`
	OPCUA       *ProtocolBenchmark `yaml:"opc_ua,omitempty" json:"opc_ua,omitempty"`
}

// ProtocolBenchmark holds performance metrics for a specific protocol
type ProtocolBenchmark struct {
	Throughput Throughput `yaml:"throughput" json:"throughput"`
	Latency    Latency    `yaml:"latency" json:"latency"`
	Resources  Resources  `yaml:"resources" json:"resources"`
	Scalability Scalability `yaml:"scalability,omitempty" json:"scalability,omitempty"`
}

// Throughput metrics
type Throughput struct {
	SingleConnection ThroughputMetric `yaml:"single_connection" json:"single_connection"`
	Concurrent100    ThroughputMetric `yaml:"concurrent_100" json:"concurrent_100"`
	Concurrent1000   ThroughputMetric `yaml:"concurrent_1000,omitempty" json:"concurrent_1000,omitempty"`
	BulkOperations   ThroughputMetric `yaml:"bulk_operations,omitempty" json:"bulk_operations,omitempty"`
}

// ThroughputMetric represents a single throughput measurement
type ThroughputMetric struct {
	Target   string  `yaml:"target" json:"target"`
	Measured string  `yaml:"measured" json:"measured"`
	Status   string  `yaml:"status" json:"status"`
	Value    float64 `yaml:"value,omitempty" json:"value,omitempty"` // Numeric value for calculations
}

// Latency metrics
type Latency struct {
	SingleRegister   LatencyMetric `yaml:"single_register" json:"single_register"`
	RoundTrip        LatencyMetric `yaml:"round_trip,omitempty" json:"round_trip,omitempty"`
	NetworkImpact    LatencyMetric `yaml:"network_impact,omitempty" json:"network_impact,omitempty"`
	P95Latency       LatencyMetric `yaml:"p95_latency,omitempty" json:"p95_latency,omitempty"`
	P99Latency       LatencyMetric `yaml:"p99_latency,omitempty" json:"p99_latency,omitempty"`
}

// LatencyMetric represents a single latency measurement
type LatencyMetric struct {
	Target   string        `yaml:"target" json:"target"`
	Measured string        `yaml:"measured" json:"measured"`
	Status   string        `yaml:"status" json:"status"`
	Value    time.Duration `yaml:"value,omitempty" json:"value,omitempty"` // Numeric value for calculations
}

// Resources metrics
type Resources struct {
	MemoryUsage    string `yaml:"memory_usage" json:"memory_usage"`
	CPUUsage       string `yaml:"cpu_usage" json:"cpu_usage"`
	Connections    string `yaml:"connections" json:"connections"`
	NetworkBandwidth string `yaml:"network_bandwidth,omitempty" json:"network_bandwidth,omitempty"`
	FileDescriptors  string `yaml:"file_descriptors,omitempty" json:"file_descriptors,omitempty"`
}

// Scalability metrics
type Scalability struct {
	ConnectionLimit      int    `yaml:"connection_limit,omitempty" json:"connection_limit,omitempty"`
	PerformanceUnderLoad string `yaml:"performance_under_load,omitempty" json:"performance_under_load,omitempty"`
	DegradationPattern   string `yaml:"degradation_pattern,omitempty" json:"degradation_pattern,omitempty"`
}

// TestResults holds test execution results
type TestResults struct {
	UnitTests        TestSuite `yaml:"unit_tests,omitempty" json:"unit_tests,omitempty"`
	IntegrationTests TestSuite `yaml:"integration_tests,omitempty" json:"integration_tests,omitempty"`
	EndToEndTests    TestSuite `yaml:"end_to_end_tests,omitempty" json:"end_to_end_tests,omitempty"`
}

// TestSuite represents test execution results for a test suite
type TestSuite struct {
	Total   int `yaml:"total" json:"total"`
	Passed  int `yaml:"passed" json:"passed"`
	Failed  int `yaml:"failed" json:"failed"`
	Skipped int `yaml:"skipped" json:"skipped"`
}

// DeviceRegistry holds device compatibility information
type DeviceRegistry struct {
	VirtualDevices []DeviceInfo `yaml:"virtual_devices,omitempty" json:"virtual_devices,omitempty"`
	RealHardware   []DeviceInfo `yaml:"real_hardware,omitempty" json:"real_hardware,omitempty"`
}

// DeviceInfo represents device compatibility information
type DeviceInfo struct {
	Vendor      string   `yaml:"vendor" json:"vendor"`
	Model       string   `yaml:"model" json:"model"`
	Protocol    string   `yaml:"protocol" json:"protocol"`
	Version     string   `yaml:"version,omitempty" json:"version,omitempty"`
	Status      string   `yaml:"status" json:"status"`
	TestedFeatures []string `yaml:"tested_features,omitempty" json:"tested_features,omitempty"`
}

// CardSummary provides a high-level summary of the release
type CardSummary struct {
	OverallStatus         string             `yaml:"overall_status" json:"overall_status"`
	PerformanceScore      float64            `yaml:"performance_score" json:"performance_score"`
	TestCoverage          float64            `yaml:"test_coverage,omitempty" json:"test_coverage,omitempty"`
	ProtocolsSupported    int                `yaml:"protocols_supported" json:"protocols_supported"`
	DevicesTested         int                `yaml:"devices_tested,omitempty" json:"devices_tested,omitempty"`
	KnownIssues           []string           `yaml:"known_issues,omitempty" json:"known_issues,omitempty"`
	PerformanceRegression bool               `yaml:"performance_regression" json:"performance_regression"`
	TrendAnalysis         *TrendAnalysis     `yaml:"trend_analysis,omitempty" json:"trend_analysis,omitempty"`
}

// TrendAnalysis provides performance trend information
type TrendAnalysis struct {
	LatencyTrend       string  `yaml:"latency_trend" json:"latency_trend"`       // "improving", "stable", "degrading"
	ThroughputTrend    string  `yaml:"throughput_trend" json:"throughput_trend"` // "improving", "stable", "degrading"
	MemoryTrend        string  `yaml:"memory_trend" json:"memory_trend"`         // "improving", "stable", "degrading"
	ComparisonToPrevious ComparisonMetrics `yaml:"comparison_to_previous" json:"comparison_to_previous"`
}

// ComparisonMetrics provides comparison to previous release
type ComparisonMetrics struct {
	LatencyChange     float64 `yaml:"latency_change_percent" json:"latency_change_percent"`
	ThroughputChange  float64 `yaml:"throughput_change_percent" json:"throughput_change_percent"`
	MemoryChange      float64 `yaml:"memory_change_percent" json:"memory_change_percent"`
	ScoreChange       float64 `yaml:"score_change_percent" json:"score_change_percent"`
}

// ToYAML converts the release card to YAML format
func (rc *ReleaseCard) ToYAML() ([]byte, error) {
	return yaml.Marshal(rc)
}

// ToJSON converts the release card to JSON format
func (rc *ReleaseCard) ToJSON() ([]byte, error) {
	return json.MarshalIndent(rc, "", "  ")
}

// FromYAML loads a release card from YAML data
func (rc *ReleaseCard) FromYAML(data []byte) error {
	return yaml.Unmarshal(data, rc)
}

// FromJSON loads a release card from JSON data
func (rc *ReleaseCard) FromJSON(data []byte) error {
	return json.Unmarshal(data, rc)
}

// Validate checks if the release card has required fields and valid data
func (rc *ReleaseCard) Validate() error {
	if rc.Metadata.Version == "" {
		return fmt.Errorf("release card missing version")
	}
	
	if rc.Metadata.ReleaseDate.IsZero() {
		return fmt.Errorf("release card missing release date")
	}
	
	// Validate performance benchmark data
	if rc.PerformanceBenchmarks.ModbusTCP == nil && 
	   rc.PerformanceBenchmarks.ModbusRTU == nil && 
	   rc.PerformanceBenchmarks.EtherNetIP == nil && 
	   rc.PerformanceBenchmarks.OPCUA == nil {
		return fmt.Errorf("release card missing performance benchmark data")
	}
	
	return nil
}

// GetPerformanceScore calculates an overall performance score based on benchmarks
func (rc *ReleaseCard) GetPerformanceScore() float64 {
	if rc.Summary.PerformanceScore > 0 {
		return rc.Summary.PerformanceScore
	}
	
	// Calculate score based on available benchmarks
	totalScore := 0.0
	protocolCount := 0
	
	protocols := []*ProtocolBenchmark{
		rc.PerformanceBenchmarks.ModbusTCP,
		rc.PerformanceBenchmarks.ModbusRTU,
		rc.PerformanceBenchmarks.EtherNetIP,
		rc.PerformanceBenchmarks.OPCUA,
	}
	
	for _, protocol := range protocols {
		if protocol != nil {
			protocolScore := calculateProtocolScore(protocol)
			totalScore += protocolScore
			protocolCount++
		}
	}
	
	if protocolCount == 0 {
		return 0.0
	}
	
	return totalScore / float64(protocolCount)
}

// calculateProtocolScore calculates a score for a single protocol benchmark
func calculateProtocolScore(benchmark *ProtocolBenchmark) float64 {
	score := 0.0
	metrics := 0
	
	// Score throughput metrics
	if benchmark.Throughput.SingleConnection.Status == "pass" {
		score += 25
	}
	metrics++
	
	if benchmark.Throughput.Concurrent100.Status == "pass" {
		score += 25
	}
	metrics++
	
	// Score latency metrics
	if benchmark.Latency.SingleRegister.Status == "pass" {
		score += 25
	}
	metrics++
	
	// Resource efficiency (basic score)
	score += 25
	metrics++
	
	return score
}