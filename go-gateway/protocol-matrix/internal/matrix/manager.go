package matrix

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Manager handles protocol testing matrix operations
type Manager struct {
	matrixPath string
	statusPath string
	matrix     *ProtocolMatrix
}

// NewManager creates a new protocol matrix manager
func NewManager(matrixPath, statusPath string) *Manager {
	return &Manager{
		matrixPath: matrixPath,
		statusPath: statusPath,
	}
}

// LoadMatrix loads the protocol matrix from the configuration file
func (m *Manager) LoadMatrix() error {
	data, err := ioutil.ReadFile(m.matrixPath)
	if err != nil {
		return fmt.Errorf("failed to read matrix file %s: %w", m.matrixPath, err)
	}

	m.matrix = &ProtocolMatrix{}
	if err := yaml.Unmarshal(data, m.matrix); err != nil {
		return fmt.Errorf("failed to parse matrix YAML: %w", err)
	}

	return nil
}

// SaveMatrix saves the current matrix to the configuration file
func (m *Manager) SaveMatrix() error {
	if m.matrix == nil {
		return fmt.Errorf("no matrix loaded")
	}

	m.matrix.LastUpdated = time.Now()

	data, err := yaml.Marshal(m.matrix)
	if err != nil {
		return fmt.Errorf("failed to marshal matrix to YAML: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(m.matrixPath), 0755); err != nil {
		return fmt.Errorf("failed to create matrix directory: %w", err)
	}

	if err := ioutil.WriteFile(m.matrixPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write matrix file: %w", err)
	}

	return nil
}

// GetMatrix returns the current protocol matrix
func (m *Manager) GetMatrix() *ProtocolMatrix {
	return m.matrix
}

// UpdateTestResults updates test results for a specific protocol and implementation
func (m *Manager) UpdateTestResults(protocolName, implementation string, results TestResults) error {
	if m.matrix == nil {
		return fmt.Errorf("no matrix loaded")
	}

	protocol, exists := m.matrix.Protocols[protocolName]
	if !exists {
		return fmt.Errorf("protocol %s not found in matrix", protocolName)
	}

	if implementation != "" {
		// Update specific implementation
		if protocol.Implementations == nil {
			return fmt.Errorf("protocol %s has no implementations", protocolName)
		}
		impl, exists := protocol.Implementations[implementation]
		if !exists {
			return fmt.Errorf("implementation %s not found for protocol %s", implementation, protocolName)
		}
		impl.TestResults = results
		protocol.Implementations[implementation] = impl
	} else {
		// Update protocol-level results
		protocol.TestResults = results
	}

	m.matrix.Protocols[protocolName] = protocol
	return nil
}

// GenerateStatus generates a comprehensive status report
func (m *Manager) GenerateStatus() (*MatrixStatus, error) {
	if m.matrix == nil {
		return nil, fmt.Errorf("no matrix loaded")
	}

	status := &MatrixStatus{
		GeneratedAt:    time.Now(),
		ProtocolStatus: make(map[string]ProtocolStatus),
		GapAnalysis:    []Gap{},
	}

	totalTests := 0
	passedTests := 0
	failedTests := 0

	// Analyze each protocol
	for protocolName, protocol := range m.matrix.Protocols {
		protocolStatus := m.analyzeProtocol(protocolName, protocol)
		status.ProtocolStatus[protocolName] = protocolStatus

		totalTests += protocolStatus.TestsRun
		passedTests += protocolStatus.TestsPassed
		failedTests += protocolStatus.TestsFailed

		// Perform gap analysis for this protocol
		gaps := m.analyzeGaps(protocolName, protocol)
		status.GapAnalysis = append(status.GapAnalysis, gaps...)
	}

	status.TotalTests = totalTests
	status.PassedTests = passedTests
	status.FailedTests = failedTests

	if totalTests > 0 {
		status.CoveragePercent = float64(passedTests) / float64(totalTests) * 100
	}

	// Determine overall status
	status.OverallStatus = m.determineOverallStatus(status)

	return status, nil
}

// SaveStatus saves the current status to a file
func (m *Manager) SaveStatus(status *MatrixStatus) error {
	data, err := yaml.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status to YAML: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(m.statusPath), 0755); err != nil {
		return fmt.Errorf("failed to create status directory: %w", err)
	}

	if err := ioutil.WriteFile(m.statusPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write status file: %w", err)
	}

	return nil
}

// LoadStatus loads status from a file
func (m *Manager) LoadStatus() (*MatrixStatus, error) {
	data, err := ioutil.ReadFile(m.statusPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty status if file doesn't exist
			return &MatrixStatus{
				GeneratedAt:    time.Now(),
				OverallStatus:  "not_run",
				ProtocolStatus: make(map[string]ProtocolStatus),
				GapAnalysis:    []Gap{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read status file: %w", err)
	}

	status := &MatrixStatus{}
	if err := yaml.Unmarshal(data, status); err != nil {
		return nil, fmt.Errorf("failed to parse status YAML: %w", err)
	}

	return status, nil
}

// analyzeProtocol analyzes a single protocol's test results
func (m *Manager) analyzeProtocol(name string, protocol Protocol) ProtocolStatus {
	status := ProtocolStatus{
		Name: name,
		Status: "not_run",
	}

	// Check implementations first
	if protocol.Implementations != nil {
		for _, impl := range protocol.Implementations {
			status.TestsRun += impl.TestResults.Passed + impl.TestResults.Failed
			status.TestsPassed += impl.TestResults.Passed
			status.TestsFailed += impl.TestResults.Failed
			
			if !impl.TestResults.LastRun.IsZero() && impl.TestResults.LastRun.After(status.LastRun) {
				status.LastRun = impl.TestResults.LastRun
			}
		}
	}

	// Check protocol-level results
	status.TestsRun += protocol.TestResults.Passed + protocol.TestResults.Failed
	status.TestsPassed += protocol.TestResults.Passed
	status.TestsFailed += protocol.TestResults.Failed
	
	if !protocol.TestResults.LastRun.IsZero() && protocol.TestResults.LastRun.After(status.LastRun) {
		status.LastRun = protocol.TestResults.LastRun
	}

	// Calculate coverage percentage
	if status.TestsRun > 0 {
		status.CoveragePercent = float64(status.TestsPassed) / float64(status.TestsRun) * 100
	}

	// Determine status
	if status.TestsRun == 0 {
		status.Status = "not_run"
	} else if status.TestsFailed == 0 {
		status.Status = "passed"
	} else if status.TestsPassed == 0 {
		status.Status = "failed"
	} else {
		status.Status = "partial"
	}

	return status
}

// analyzeGaps performs gap analysis for a protocol
func (m *Manager) analyzeGaps(protocolName string, protocol Protocol) []Gap {
	gaps := []Gap{}

	// Check for missing virtual devices
	if len(protocol.VirtualDevices) == 0 {
		gaps = append(gaps, Gap{
			Protocol: protocolName,
			Category: "virtual_devices",
			Missing:  []string{"No virtual devices configured"},
			Severity: "high",
		})
	}

	// Check implementations for Modbus
	if protocolName == "modbus" && protocol.Implementations != nil {
		if tcp, exists := protocol.Implementations["tcp"]; exists {
			if len(tcp.TestCoverage.FunctionCodes) == 0 {
				gaps = append(gaps, Gap{
					Protocol:     protocolName,
					Implementation: "tcp",
					Category:     "function_codes",
					Missing:      []string{"No function codes defined"},
					Severity:     "high",
				})
			}
		}
	}

	// Check for missing performance tests
	if protocol.Implementations != nil {
		for implName, impl := range protocol.Implementations {
			if !impl.TestCoverage.PerformanceTests {
				gaps = append(gaps, Gap{
					Protocol:     protocolName,
					Implementation: implName,
					Category:     "performance_tests",
					Missing:      []string{"Performance tests not enabled"},
					Severity:     "medium",
				})
			}
		}
	}

	return gaps
}

// determineOverallStatus determines the overall status based on protocol statuses
func (m *Manager) determineOverallStatus(status *MatrixStatus) string {
	if status.TotalTests == 0 {
		return "not_run"
	}
	
	if status.FailedTests == 0 {
		return "passed"
	}
	
	if status.PassedTests == 0 {
		return "failed"
	}
	
	return "partial"
}

// GetProtocolNames returns the list of supported protocol names
func (m *Manager) GetProtocolNames() []string {
	if m.matrix == nil {
		return []string{}
	}

	names := make([]string, 0, len(m.matrix.Protocols))
	for name := range m.matrix.Protocols {
		names = append(names, name)
	}
	return names
}