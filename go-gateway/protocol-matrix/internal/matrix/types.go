// Package matrix provides types and functionality for protocol testing matrix tracking
package matrix

import (
	"fmt"
	"strings"
	"time"
)

// ProtocolMatrix represents the complete testing matrix configuration
type ProtocolMatrix struct {
	SchemaVersion string                `yaml:"schema_version"`
	LastUpdated   time.Time             `yaml:"last_updated"`
	Protocols     map[string]Protocol   `yaml:"protocols"`
	TestExecution TestExecutionConfig   `yaml:"test_execution"`
	PerformanceTargets map[string]PerformanceTarget `yaml:"performance_targets"`
}

// Protocol represents a single protocol's testing configuration
type Protocol struct {
	Implementations    map[string]Implementation `yaml:"implementations,omitempty"`
	TestCoverage       TestCoverage             `yaml:"test_coverage,omitempty"`
	VirtualDevices     []string                 `yaml:"virtual_devices"`
	RealDevices        []string                 `yaml:"real_devices"`
	VendorCompatibility []string                `yaml:"vendor_compatibility,omitempty"`
	TestResults        TestResults              `yaml:"test_results"`
}

// Implementation represents a specific protocol implementation (e.g., Modbus TCP vs RTU)
type Implementation struct {
	TestCoverage   TestCoverage `yaml:"test_coverage"`
	VirtualDevices []string     `yaml:"virtual_devices"`
	RealDevices    []string     `yaml:"real_devices"`
	TestResults    TestResults  `yaml:"test_results"`
}

// TestCoverage defines what tests should be covered for a protocol
type TestCoverage struct {
	FunctionCodes       []int    `yaml:"function_codes,omitempty"`
	ExceptionCodes      []int    `yaml:"exception_codes,omitempty"`
	PerformanceTests    bool     `yaml:"performance_tests,omitempty"`
	ConcurrentConnections int    `yaml:"concurrent_connections,omitempty"`
	DataTypes           []string `yaml:"data_types,omitempty"`
	SerialConfigs       []string `yaml:"serial_configs,omitempty"`
	SecurityPolicies    []string `yaml:"security_policies,omitempty"`
	AuthenticationModes []string `yaml:"authentication_modes,omitempty"`
	Operations          []string `yaml:"operations,omitempty"`
	MessagingTypes      []string `yaml:"messaging_types,omitempty"`
	ConnectionTypes     []string `yaml:"connection_types,omitempty"`
	Services            []string `yaml:"services,omitempty"`
	DeviceProfiles      []string `yaml:"device_profiles,omitempty"`
	CPUFamilies         []string `yaml:"cpu_families,omitempty"`
	MemoryAreas         []string `yaml:"memory_areas,omitempty"`
}

// TestResults tracks the results of test execution
type TestResults struct {
	LastRun             time.Time `yaml:"last_run"`
	Status              string    `yaml:"status"` // "not_run", "running", "passed", "failed", "partial"
	Passed              int       `yaml:"passed"`
	Failed              int       `yaml:"failed"`
	CoveragePercentage  float64   `yaml:"coverage_percentage"`
	Details             []TestDetail `yaml:"details,omitempty"`
}

// TestDetail provides detailed information about individual test results
type TestDetail struct {
	TestName     string    `yaml:"test_name"`
	Status       string    `yaml:"status"` // "passed", "failed", "skipped"
	Duration     time.Duration `yaml:"duration"`
	ErrorMessage string    `yaml:"error_message,omitempty"`
	Category     string    `yaml:"category"` // "function_code", "performance", "exception", etc.
}

// TestExecutionConfig defines how tests should be executed
type TestExecutionConfig struct {
	Timeout            string `yaml:"timeout"`
	ParallelTests      bool   `yaml:"parallel_tests"`
	MaxWorkers         int    `yaml:"max_workers"`
	RetryFailedTests   bool   `yaml:"retry_failed_tests"`
	MaxRetries         int    `yaml:"max_retries"`
	GenerateHTMLReport bool   `yaml:"generate_html_report"`
	GenerateJSONReport bool   `yaml:"generate_json_report"`
}

// PerformanceTarget defines expected performance metrics for a protocol
type PerformanceTarget struct {
	ThroughputOpsPerSec   int `yaml:"throughput_ops_per_sec"`
	LatencyP95Ms          int `yaml:"latency_p95_ms"`
	ConcurrentConnections int `yaml:"concurrent_connections"`
}

// MatrixStatus represents the overall status of the testing matrix
type MatrixStatus struct {
	GeneratedAt     time.Time                  `yaml:"generated_at"`
	OverallStatus   string                     `yaml:"overall_status"`
	TotalTests      int                        `yaml:"total_tests"`
	PassedTests     int                        `yaml:"passed_tests"`
	FailedTests     int                        `yaml:"failed_tests"`
	CoveragePercent float64                    `yaml:"coverage_percent"`
	ProtocolStatus  map[string]ProtocolStatus  `yaml:"protocol_status"`
	GapAnalysis     []Gap                      `yaml:"gap_analysis"`
}

// ProtocolStatus represents the status of testing for a single protocol
type ProtocolStatus struct {
	Name            string    `yaml:"name"`
	Status          string    `yaml:"status"`
	TestsRun        int       `yaml:"tests_run"`
	TestsPassed     int       `yaml:"tests_passed"`
	TestsFailed     int       `yaml:"tests_failed"`
	CoveragePercent float64   `yaml:"coverage_percent"`
	LastRun         time.Time `yaml:"last_run"`
}

// Gap represents a gap in test coverage
type Gap struct {
	Protocol     string `yaml:"protocol"`
	Implementation string `yaml:"implementation,omitempty"`
	Category     string `yaml:"category"`
	Missing      []string `yaml:"missing"`
	Severity     string `yaml:"severity"` // "high", "medium", "low"
}

// String methods for better display
func (tr *TestResults) String() string {
	return fmt.Sprintf("Status: %s, Passed: %d, Failed: %d, Coverage: %.1f%%",
		tr.Status, tr.Passed, tr.Failed, tr.CoveragePercentage)
}

func (ps *ProtocolStatus) String() string {
	return fmt.Sprintf("%s: %s (%.1f%% coverage, %d/%d tests passed)",
		ps.Name, ps.Status, ps.CoveragePercent, ps.TestsPassed, ps.TestsRun)
}

func (g *Gap) String() string {
	return fmt.Sprintf("[%s] %s/%s - Missing: %v",
		strings.ToUpper(g.Severity), g.Protocol, g.Category, g.Missing)
}