package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bifrost-gateway/protocol-matrix/internal/matrix"
)

const (
	defaultMatrixPath = "protocol-matrix/protocol_matrix.yaml"
	defaultStatusPath = "protocol-matrix/status/matrix_status.yaml"
)

func main() {
	var (
		command    = flag.String("command", "status", "Command to execute: status, update, analyze, validate")
		matrixPath = flag.String("matrix", defaultMatrixPath, "Path to protocol matrix configuration file")
		statusPath = flag.String("status", defaultStatusPath, "Path to status file")
		protocol   = flag.String("protocol", "", "Protocol name for update operations")
		impl       = flag.String("implementation", "", "Implementation name for update operations")
		jsonOutput = flag.Bool("json", false, "Output results in JSON format")
		verbose    = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	// Create matrix manager
	manager := matrix.NewManager(*matrixPath, *statusPath)

	// Load the matrix configuration
	if err := manager.LoadMatrix(); err != nil {
		log.Fatalf("Failed to load matrix: %v", err)
	}

	switch *command {
	case "status":
		if err := handleStatus(manager, *jsonOutput, *verbose); err != nil {
			log.Fatalf("Status command failed: %v", err)
		}
	case "analyze":
		if err := handleAnalyze(manager, *jsonOutput); err != nil {
			log.Fatalf("Analyze command failed: %v", err)
		}
	case "validate":
		if err := handleValidate(manager); err != nil {
			log.Fatalf("Validate command failed: %v", err)
		}
	case "update":
		if err := handleUpdate(manager, *protocol, *impl); err != nil {
			log.Fatalf("Update command failed: %v", err)
		}
	case "report":
		if err := handleReport(manager, *jsonOutput); err != nil {
			log.Fatalf("Report command failed: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		flag.Usage()
		os.Exit(1)
	}
}

// handleStatus displays the current status of the protocol matrix
func handleStatus(manager *matrix.Manager, jsonOutput, verbose bool) error {
	status, err := manager.GenerateStatus()
	if err != nil {
		return fmt.Errorf("failed to generate status: %w", err)
	}

	if jsonOutput {
		data, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal status to JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Text output
	fmt.Printf("Protocol Testing Matrix Status\n")
	fmt.Printf("==============================\n\n")
	fmt.Printf("Generated: %s\n", status.GeneratedAt.Format(time.RFC3339))
	fmt.Printf("Overall Status: %s\n", strings.ToUpper(status.OverallStatus))
	fmt.Printf("Total Tests: %d (Passed: %d, Failed: %d)\n", 
		status.TotalTests, status.PassedTests, status.FailedTests)
	fmt.Printf("Coverage: %.1f%%\n\n", status.CoveragePercent)

	fmt.Printf("Protocol Status:\n")
	fmt.Printf("----------------\n")
	for _, protocolStatus := range status.ProtocolStatus {
		fmt.Printf("• %s\n", protocolStatus.String())
		if verbose && !protocolStatus.LastRun.IsZero() {
			fmt.Printf("  Last Run: %s\n", protocolStatus.LastRun.Format(time.RFC3339))
		}
	}

	if len(status.GapAnalysis) > 0 {
		fmt.Printf("\nGap Analysis:\n")
		fmt.Printf("-------------\n")
		for _, gap := range status.GapAnalysis {
			fmt.Printf("• %s\n", gap.String())
		}
	}

	return nil
}

// handleAnalyze performs detailed analysis and saves the results
func handleAnalyze(manager *matrix.Manager, jsonOutput bool) error {
	status, err := manager.GenerateStatus()
	if err != nil {
		return fmt.Errorf("failed to generate status: %w", err)
	}

	// Save status to file
	if err := manager.SaveStatus(status); err != nil {
		return fmt.Errorf("failed to save status: %w", err)
	}

	fmt.Printf("Analysis complete. Status saved.\n")

	// Display summary
	if jsonOutput {
		summary := map[string]interface{}{
			"protocols_analyzed": len(status.ProtocolStatus),
			"total_tests": status.TotalTests,
			"coverage_percent": status.CoveragePercent,
			"gaps_found": len(status.GapAnalysis),
			"overall_status": status.OverallStatus,
		}
		data, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Analyzed %d protocols\n", len(status.ProtocolStatus))
		fmt.Printf("Found %d gaps\n", len(status.GapAnalysis))
		fmt.Printf("Overall coverage: %.1f%%\n", status.CoveragePercent)
	}

	return nil
}

// handleValidate validates the matrix configuration
func handleValidate(manager *matrix.Manager) error {
	matrixData := manager.GetMatrix()
	if matrixData == nil {
		return fmt.Errorf("no matrix loaded")
	}

	errors := []string{}

	// Validate schema version
	if matrixData.SchemaVersion == "" {
		errors = append(errors, "schema_version is required")
	}

	// Validate protocols
	if len(matrixData.Protocols) == 0 {
		errors = append(errors, "at least one protocol must be defined")
	}

	for protocolName, protocol := range matrixData.Protocols {
		// Check if protocol has devices (either at protocol level or implementation level)
		hasDevices := len(protocol.VirtualDevices) > 0 || len(protocol.RealDevices) > 0
		
		// For protocols with implementations, check implementations too
		if protocol.Implementations != nil {
			for _, impl := range protocol.Implementations {
				if len(impl.VirtualDevices) > 0 || len(impl.RealDevices) > 0 {
					hasDevices = true
					break
				}
			}
		}
		
		if !hasDevices {
			errors = append(errors, fmt.Sprintf("protocol %s has no devices configured", protocolName))
		}

		// Validate implementations for modbus
		if protocolName == "modbus" {
			if protocol.Implementations == nil || len(protocol.Implementations) == 0 {
				errors = append(errors, "modbus protocol must have implementations (tcp, rtu)")
			}
		}
	}

	// Validate performance targets
	for protocolName := range matrixData.Protocols {
		if _, exists := matrixData.PerformanceTargets[protocolName]; !exists {
			errors = append(errors, fmt.Sprintf("no performance targets defined for protocol %s", protocolName))
		}
	}

	if len(errors) > 0 {
		fmt.Printf("Validation failed with %d errors:\n", len(errors))
		for i, err := range errors {
			fmt.Printf("%d. %s\n", i+1, err)
		}
		return fmt.Errorf("validation failed")
	}

	fmt.Println("✅ Matrix configuration is valid")
	return nil
}

// handleUpdate updates test results for a protocol
func handleUpdate(manager *matrix.Manager, protocolName, implementation string) error {
	if protocolName == "" {
		return fmt.Errorf("protocol name is required for update operations")
	}

	// For demonstration, create sample test results
	// In a real implementation, this would integrate with actual test runners
	results := matrix.TestResults{
		LastRun:            time.Now(),
		Status:             "passed",
		Passed:             10,
		Failed:             0,
		CoveragePercentage: 100.0,
		Details: []matrix.TestDetail{
			{
				TestName: "function_code_01_test",
				Status:   "passed",
				Duration: 50 * time.Millisecond,
				Category: "function_code",
			},
			{
				TestName: "performance_throughput_test",
				Status:   "passed",
				Duration: 2 * time.Second,
				Category: "performance",
			},
		},
	}

	if err := manager.UpdateTestResults(protocolName, implementation, results); err != nil {
		return fmt.Errorf("failed to update test results: %w", err)
	}

	if err := manager.SaveMatrix(); err != nil {
		return fmt.Errorf("failed to save matrix: %w", err)
	}

	fmt.Printf("✅ Updated test results for %s", protocolName)
	if implementation != "" {
		fmt.Printf("/%s", implementation)
	}
	fmt.Println()

	return nil
}

// handleReport generates various types of reports
func handleReport(manager *matrix.Manager, jsonOutput bool) error {
	status, err := manager.GenerateStatus()
	if err != nil {
		return fmt.Errorf("failed to generate status: %w", err)
	}

	// Generate HTML report
	if err := generateHTMLReport(status); err != nil {
		fmt.Printf("Warning: failed to generate HTML report: %v\n", err)
	}

	// Generate JSON report
	reportPath := "protocol-matrix/status/report.json"
	if err := generateJSONReport(status, reportPath); err != nil {
		return fmt.Errorf("failed to generate JSON report: %w", err)
	}

	fmt.Printf("Reports generated:\n")
	fmt.Printf("• JSON: %s\n", reportPath)
	fmt.Printf("• HTML: protocol-matrix/status/report.html\n")

	return nil
}

// generateJSONReport creates a detailed JSON report
func generateJSONReport(status *matrix.MatrixStatus, filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status to JSON: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON report: %w", err)
	}

	return nil
}

// generateHTMLReport creates a basic HTML report
func generateHTMLReport(status *matrix.MatrixStatus) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Protocol Testing Matrix Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f0f0f0; padding: 20px; border-radius: 5px; }
        .protocol { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .passed { color: green; }
        .failed { color: red; }
        .partial { color: orange; }
        .not_run { color: gray; }
        .gap { background: #fff3cd; padding: 10px; margin: 5px 0; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Protocol Testing Matrix Report</h1>
        <p>Generated: %s</p>
        <p>Overall Status: <span class="%s">%s</span></p>
        <p>Coverage: %.1f%% (%d/%d tests passed)</p>
    </div>
    
    <h2>Protocol Status</h2>
`

	html = fmt.Sprintf(html, 
		status.GeneratedAt.Format(time.RFC3339),
		status.OverallStatus,
		strings.ToUpper(status.OverallStatus),
		status.CoveragePercent,
		status.PassedTests,
		status.TotalTests)

	for _, ps := range status.ProtocolStatus {
		html += fmt.Sprintf(`
    <div class="protocol">
        <h3>%s</h3>
        <p>Status: <span class="%s">%s</span></p>
        <p>Tests: %d passed, %d failed (%.1f%% coverage)</p>
        <p>Last Run: %s</p>
    </div>`, 
			ps.Name, 
			ps.Status, 
			strings.ToUpper(ps.Status),
			ps.TestsPassed, 
			ps.TestsFailed, 
			ps.CoveragePercent,
			ps.LastRun.Format(time.RFC3339))
	}

	if len(status.GapAnalysis) > 0 {
		html += `<h2>Gap Analysis</h2>`
		for _, gap := range status.GapAnalysis {
			html += fmt.Sprintf(`
    <div class="gap">
        <strong>[%s]</strong> %s/%s<br>
        Missing: %s
    </div>`, 
				strings.ToUpper(gap.Severity),
				gap.Protocol,
				gap.Category,
				strings.Join(gap.Missing, ", "))
		}
	}

	html += `
</body>
</html>`

	filePath := "protocol-matrix/status/report.html"
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write HTML report: %w", err)
	}

	return nil
}