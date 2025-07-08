package matrix

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerLoadMatrix(t *testing.T) {
	// Create a temporary matrix file
	tempDir := t.TempDir()
	matrixPath := filepath.Join(tempDir, "test_matrix.yaml")
	statusPath := filepath.Join(tempDir, "test_status.yaml")

	matrixContent := `
schema_version: "1.0"
last_updated: "2024-07-08T14:00:00Z"
protocols:
  modbus:
    implementations:
      tcp:
        test_coverage:
          function_codes: [1, 2, 3, 4]
          performance_tests: true
        virtual_devices:
          - "test_simulator"
        real_devices: []
        test_results:
          status: "not_run"
          passed: 0
          failed: 0
          coverage_percentage: 0.0
test_execution:
  timeout: "30m"
  parallel_tests: true
performance_targets:
  modbus:
    throughput_ops_per_sec: 1000
`

	if err := os.WriteFile(matrixPath, []byte(matrixContent), 0644); err != nil {
		t.Fatalf("Failed to create test matrix file: %v", err)
	}

	manager := NewManager(matrixPath, statusPath)

	// Test loading
	if err := manager.LoadMatrix(); err != nil {
		t.Fatalf("Failed to load matrix: %v", err)
	}

	matrix := manager.GetMatrix()
	if matrix == nil {
		t.Fatal("Matrix is nil after loading")
	}

	if matrix.SchemaVersion != "1.0" {
		t.Errorf("Expected schema version 1.0, got %s", matrix.SchemaVersion)
	}

	if len(matrix.Protocols) != 1 {
		t.Errorf("Expected 1 protocol, got %d", len(matrix.Protocols))
	}

	modbus, exists := matrix.Protocols["modbus"]
	if !exists {
		t.Fatal("Modbus protocol not found")
	}

	if len(modbus.Implementations) != 1 {
		t.Errorf("Expected 1 modbus implementation, got %d", len(modbus.Implementations))
	}

	tcp, exists := modbus.Implementations["tcp"]
	if !exists {
		t.Fatal("Modbus TCP implementation not found")
	}

	if len(tcp.TestCoverage.FunctionCodes) != 4 {
		t.Errorf("Expected 4 function codes, got %d", len(tcp.TestCoverage.FunctionCodes))
	}
}

func TestManagerUpdateTestResults(t *testing.T) {
	tempDir := t.TempDir()
	matrixPath := filepath.Join(tempDir, "test_matrix.yaml")
	statusPath := filepath.Join(tempDir, "test_status.yaml")

	// Create a simple matrix
	matrixContent := `
schema_version: "1.0"
protocols:
  modbus:
    implementations:
      tcp:
        test_coverage:
          function_codes: [1, 2, 3]
        virtual_devices: ["test"]
        real_devices: []
        test_results:
          status: "not_run"
          passed: 0
          failed: 0
          coverage_percentage: 0.0
test_execution:
  timeout: "30m"
performance_targets:
  modbus:
    throughput_ops_per_sec: 1000
`

	if err := os.WriteFile(matrixPath, []byte(matrixContent), 0644); err != nil {
		t.Fatalf("Failed to create test matrix file: %v", err)
	}

	manager := NewManager(matrixPath, statusPath)
	if err := manager.LoadMatrix(); err != nil {
		t.Fatalf("Failed to load matrix: %v", err)
	}

	// Test updating results
	results := TestResults{
		LastRun:            time.Now(),
		Status:             "passed",
		Passed:             5,
		Failed:             1,
		CoveragePercentage: 83.3,
	}

	if err := manager.UpdateTestResults("modbus", "tcp", results); err != nil {
		t.Fatalf("Failed to update test results: %v", err)
	}

	// Verify the update
	matrix := manager.GetMatrix()
	tcp := matrix.Protocols["modbus"].Implementations["tcp"]
	
	if tcp.TestResults.Status != "passed" {
		t.Errorf("Expected status 'passed', got %s", tcp.TestResults.Status)
	}

	if tcp.TestResults.Passed != 5 {
		t.Errorf("Expected 5 passed tests, got %d", tcp.TestResults.Passed)
	}

	if tcp.TestResults.Failed != 1 {
		t.Errorf("Expected 1 failed test, got %d", tcp.TestResults.Failed)
	}
}

func TestManagerGenerateStatus(t *testing.T) {
	tempDir := t.TempDir()
	matrixPath := filepath.Join(tempDir, "test_matrix.yaml")
	statusPath := filepath.Join(tempDir, "test_status.yaml")

	// Create matrix with test results
	matrixContent := `
schema_version: "1.0"
protocols:
  modbus:
    implementations:
      tcp:
        test_coverage:
          function_codes: [1, 2, 3]
        virtual_devices: ["test"]
        real_devices: []
        test_results:
          last_run: "2024-07-08T14:00:00Z"
          status: "passed"
          passed: 5
          failed: 1
          coverage_percentage: 83.3
  opcua:
    test_coverage:
      operations: ["Read", "Write"]
    virtual_devices: []
    real_devices: []
    test_results:
      status: "not_run"
      passed: 0
      failed: 0
      coverage_percentage: 0.0
test_execution:
  timeout: "30m"
performance_targets:
  modbus:
    throughput_ops_per_sec: 1000
`

	if err := os.WriteFile(matrixPath, []byte(matrixContent), 0644); err != nil {
		t.Fatalf("Failed to create test matrix file: %v", err)
	}

	manager := NewManager(matrixPath, statusPath)
	if err := manager.LoadMatrix(); err != nil {
		t.Fatalf("Failed to load matrix: %v", err)
	}

	// Generate status
	status, err := manager.GenerateStatus()
	if err != nil {
		t.Fatalf("Failed to generate status: %v", err)
	}

	if status.TotalTests != 6 {
		t.Errorf("Expected 6 total tests, got %d", status.TotalTests)
	}

	if status.PassedTests != 5 {
		t.Errorf("Expected 5 passed tests, got %d", status.PassedTests)
	}

	if status.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", status.FailedTests)
	}

	if len(status.ProtocolStatus) != 2 {
		t.Errorf("Expected 2 protocol statuses, got %d", len(status.ProtocolStatus))
	}

	modbusStatus := status.ProtocolStatus["modbus"]
	if modbusStatus.Status != "partial" {
		t.Errorf("Expected modbus status 'partial', got %s", modbusStatus.Status)
	}

	opcuaStatus := status.ProtocolStatus["opcua"]
	if opcuaStatus.Status != "not_run" {
		t.Errorf("Expected opcua status 'not_run', got %s", opcuaStatus.Status)
	}

	// Check gaps analysis
	if len(status.GapAnalysis) == 0 {
		t.Error("Expected gaps to be found in analysis")
	}

	// Check for missing virtual devices gap for opcua
	foundGap := false
	for _, gap := range status.GapAnalysis {
		if gap.Protocol == "opcua" && gap.Category == "virtual_devices" {
			foundGap = true
			break
		}
	}
	if !foundGap {
		t.Error("Expected to find virtual devices gap for opcua")
	}
}

func TestManagerSaveAndLoadStatus(t *testing.T) {
	tempDir := t.TempDir()
	matrixPath := filepath.Join(tempDir, "test_matrix.yaml")
	statusPath := filepath.Join(tempDir, "test_status.yaml")

	// Create minimal matrix
	matrixContent := `
schema_version: "1.0"
protocols:
  modbus:
    implementations:
      tcp:
        test_coverage:
          function_codes: [1]
        virtual_devices: ["test"]
        real_devices: []
        test_results:
          status: "passed"
          passed: 1
          failed: 0
          coverage_percentage: 100.0
test_execution:
  timeout: "30m"
performance_targets:
  modbus:
    throughput_ops_per_sec: 1000
`

	if err := os.WriteFile(matrixPath, []byte(matrixContent), 0644); err != nil {
		t.Fatalf("Failed to create test matrix file: %v", err)
	}

	manager := NewManager(matrixPath, statusPath)
	if err := manager.LoadMatrix(); err != nil {
		t.Fatalf("Failed to load matrix: %v", err)
	}

	// Generate and save status
	status, err := manager.GenerateStatus()
	if err != nil {
		t.Fatalf("Failed to generate status: %v", err)
	}

	if err := manager.SaveStatus(status); err != nil {
		t.Fatalf("Failed to save status: %v", err)
	}

	// Load status back
	loadedStatus, err := manager.LoadStatus()
	if err != nil {
		t.Fatalf("Failed to load status: %v", err)
	}

	if loadedStatus.TotalTests != status.TotalTests {
		t.Errorf("Loaded status has different total tests: expected %d, got %d", 
			status.TotalTests, loadedStatus.TotalTests)
	}

	if loadedStatus.OverallStatus != status.OverallStatus {
		t.Errorf("Loaded status has different overall status: expected %s, got %s", 
			status.OverallStatus, loadedStatus.OverallStatus)
	}
}

func TestProtocolStatus_String(t *testing.T) {
	ps := ProtocolStatus{
		Name:            "modbus",
		Status:          "passed",
		TestsRun:        10,
		TestsPassed:     9,
		TestsFailed:     1,
		CoveragePercent: 90.0,
	}

	expected := "modbus: passed (90.0% coverage, 9/10 tests passed)"
	if ps.String() != expected {
		t.Errorf("Expected %s, got %s", expected, ps.String())
	}
}

func TestGap_String(t *testing.T) {
	gap := Gap{
		Protocol: "modbus",
		Category: "function_codes",
		Missing:  []string{"code_5", "code_6"},
		Severity: "high",
	}

	expected := "[HIGH] modbus/function_codes - Missing: [code_5 code_6]"
	if gap.String() != expected {
		t.Errorf("Expected %s, got %s", expected, gap.String())
	}
}