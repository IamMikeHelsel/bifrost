# Protocol Testing Matrix Integration Guide

This document explains how to integrate the Protocol Testing Matrix system with your existing testing infrastructure.

## Quick Start

1. **Validate your matrix configuration:**
   ```bash
   cd go-gateway
   make matrix-validate
   ```

2. **Check current status:**
   ```bash
   make matrix-status
   ```

3. **Run analysis and generate reports:**
   ```bash
   make matrix
   ```

## Integration with Existing Tests

### Go Tests Integration

Add matrix updates to your existing Go test files:

```go
// internal/protocols/modbus_test.go
func TestModbusTCP(t *testing.T) {
    // ... existing test code ...
    
    // Update matrix with results
    manager := matrix.NewManager("../../protocol-matrix/protocol_matrix.yaml", "../../protocol-matrix/status/matrix_status.yaml")
    if err := manager.LoadMatrix(); err != nil {
        t.Logf("Warning: Could not load matrix: %v", err)
        return
    }
    
    results := matrix.TestResults{
        LastRun:            time.Now(),
        Status:             "passed", // or "failed"
        Passed:             testsPassed,
        Failed:             testsFailed,
        CoveragePercentage: float64(testsPassed) / float64(total) * 100,
    }
    
    manager.UpdateTestResults("modbus", "tcp", results)
    manager.SaveMatrix()
}
```

### Python Tests Integration

For Python-based virtual device tests:

```python
# virtual-devices/test_modbus_integration.py
import yaml
import json
from datetime import datetime

def update_protocol_matrix(protocol, implementation, passed, failed, details=None):
    """Update the protocol matrix with test results."""
    matrix_path = "../go-gateway/protocol-matrix/protocol_matrix.yaml"
    
    try:
        with open(matrix_path, 'r') as f:
            matrix = yaml.safe_load(f)
        
        # Update test results
        if implementation:
            target = matrix['protocols'][protocol]['implementations'][implementation]
        else:
            target = matrix['protocols'][protocol]
        
        target['test_results'] = {
            'last_run': datetime.now().isoformat() + 'Z',
            'status': 'passed' if failed == 0 else 'failed',
            'passed': passed,
            'failed': failed,
            'coverage_percentage': (passed / (passed + failed)) * 100 if (passed + failed) > 0 else 0,
            'details': details or []
        }
        
        with open(matrix_path, 'w') as f:
            yaml.dump(matrix, f, default_flow_style=False)
            
    except Exception as e:
        print(f"Warning: Could not update matrix: {e}")

def test_modbus_tcp_connection():
    """Test Modbus TCP connection and update matrix."""
    passed = 0
    failed = 0
    
    try:
        # Your existing test code here
        # ...
        passed += 1
    except Exception as e:
        failed += 1
        print(f"Test failed: {e}")
    
    # Update matrix
    update_protocol_matrix("modbus", "tcp", passed, failed)
```

### Performance Test Integration

For performance tests, include performance metrics:

```go
// go-gateway/cmd/performance_test/main.go
func updateMatrixWithPerfResults(protocol, implementation string, results *performance.BenchmarkResults) {
    manager := matrix.NewManager("protocol-matrix/protocol_matrix.yaml", "protocol-matrix/status/matrix_status.yaml")
    if err := manager.LoadMatrix(); err != nil {
        log.Printf("Warning: Could not load matrix: %v", err)
        return
    }
    
    testResults := matrix.TestResults{
        LastRun: time.Now(),
        Status:  "passed", // Determine based on performance targets
        Passed:  1,
        Failed:  0,
        CoveragePercentage: 100.0,
        Details: []matrix.TestDetail{
            {
                TestName: "performance_throughput",
                Status:   "passed",
                Duration: results.Duration,
                Category: "performance",
            },
        },
    }
    
    // Check against performance targets
    targets := manager.GetMatrix().PerformanceTargets[protocol]
    if results.ThroughputOPS < targets.ThroughputOpsPerSec {
        testResults.Status = "failed"
        testResults.Failed = 1
        testResults.Passed = 0
    }
    
    manager.UpdateTestResults(protocol, implementation, testResults)
    manager.SaveMatrix()
}
```

## CI/CD Integration

### GitHub Actions

Use the provided workflow example (`.github-workflows-example.yml`) as a template:

1. **Copy the workflow file:**
   ```bash
   cp go-gateway/protocol-matrix/.github-workflows-example.yml .github/workflows/protocol-matrix.yml
   ```

2. **Customize for your needs:**
   - Adjust the test commands
   - Modify the protocol/implementation matrix
   - Configure artifact retention

3. **Add to existing workflows:**
   ```yaml
   # Add to your existing CI workflow
   - name: Update Protocol Matrix
     run: |
       cd go-gateway
       make matrix-analyze
       make matrix-report
   
   - name: Upload Matrix Reports
     uses: actions/upload-artifact@v3
     with:
       name: protocol-matrix-reports
       path: go-gateway/protocol-matrix/status/
   ```

### Jenkins Integration

```groovy
// Jenkinsfile
pipeline {
    agent any
    
    stages {
        stage('Protocol Matrix Validation') {
            steps {
                dir('go-gateway') {
                    sh 'make matrix-validate'
                }
            }
        }
        
        stage('Run Protocol Tests') {
            parallel {
                stage('Modbus Tests') {
                    steps {
                        dir('go-gateway') {
                            sh 'go test -v ./internal/protocols/modbus_test.go'
                            sh './bin/matrix -command=update -protocol=modbus -implementation=tcp'
                        }
                    }
                }
                stage('OPC UA Tests') {
                    steps {
                        dir('go-gateway') {
                            sh 'go test -v ./internal/protocols/opcua_test.go'
                            sh './bin/matrix -command=update -protocol=opcua'
                        }
                    }
                }
            }
        }
        
        stage('Generate Matrix Reports') {
            steps {
                dir('go-gateway') {
                    sh 'make matrix-analyze'
                    sh 'make matrix-report'
                }
                
                publishHTML([
                    allowMissing: false,
                    alwaysLinkToLastBuild: true,
                    keepAll: true,
                    reportDir: 'go-gateway/protocol-matrix/status',
                    reportFiles: 'report.html',
                    reportName: 'Protocol Matrix Report'
                ])
                
                archiveArtifacts artifacts: 'go-gateway/protocol-matrix/status/*', allowEmptyArchive: false
            }
        }
    }
    
    post {
        always {
            dir('go-gateway') {
                sh 'make matrix-status'
            }
        }
    }
}
```

## Custom Test Runner Integration

Create a custom test runner that integrates with the matrix:

```bash
#!/bin/bash
# scripts/run-protocol-tests.sh

set -e

MATRIX_DIR="go-gateway/protocol-matrix"
MATRIX_CLI="go-gateway/bin/matrix"

# Build the matrix CLI
cd go-gateway
make matrix-build
cd ..

echo "Running comprehensive protocol test suite..."

# Function to run tests and update matrix
run_protocol_tests() {
    local protocol=$1
    local implementation=$2
    local test_command=$3
    
    echo "Running tests for $protocol/$implementation..."
    
    # Run the actual tests
    if eval "$test_command"; then
        echo "✅ Tests passed for $protocol/$implementation"
        if [ -n "$implementation" ]; then
            $MATRIX_CLI -command=update -protocol="$protocol" -implementation="$implementation"
        else
            $MATRIX_CLI -command=update -protocol="$protocol"
        fi
    else
        echo "❌ Tests failed for $protocol/$implementation"
        # You could still update the matrix with failure results
    fi
}

# Run tests for each protocol
run_protocol_tests "modbus" "tcp" "cd go-gateway && go test -v ./internal/protocols/modbus_test.go"
run_protocol_tests "modbus" "rtu" "cd virtual-devices && python3 test_modbus_rtu.py"
run_protocol_tests "opcua" "" "cd go-gateway && go test -v ./internal/protocols/opcua_test.go"
run_protocol_tests "ethernetip" "" "cd go-gateway && go test -v ./internal/protocols/ethernetip_test.go"
run_protocol_tests "s7" "" "cd virtual-devices && python3 test_s7.py"

# Generate final reports
echo "Generating final matrix analysis..."
cd go-gateway
make matrix-analyze
make matrix-report

echo "Protocol testing complete. Reports available in:"
echo "- JSON: $MATRIX_DIR/status/report.json"
echo "- HTML: $MATRIX_DIR/status/report.html"
echo "- YAML: $MATRIX_DIR/status/matrix_status.yaml"
```

## Dashboard Integration

### Grafana Integration

Create a Grafana dashboard that reads from the JSON reports:

```json
{
  "dashboard": {
    "title": "Protocol Testing Matrix",
    "panels": [
      {
        "title": "Overall Coverage",
        "type": "stat",
        "targets": [
          {
            "expr": "protocol_matrix_coverage_percent",
            "legendFormat": "Coverage %"
          }
        ]
      },
      {
        "title": "Protocol Status",
        "type": "table",
        "targets": [
          {
            "expr": "protocol_matrix_status",
            "legendFormat": "{{protocol}}"
          }
        ]
      }
    ]
  }
}
```

### Prometheus Metrics

Export matrix metrics to Prometheus:

```go
// Add to your metrics collection
func (m *Manager) ExportPrometheusMetrics() {
    status, _ := m.GenerateStatus()
    
    prometheus.GaugeSet("protocol_matrix_coverage_percent", status.CoveragePercent)
    prometheus.GaugeSet("protocol_matrix_total_tests", float64(status.TotalTests))
    prometheus.GaugeSet("protocol_matrix_passed_tests", float64(status.PassedTests))
    prometheus.GaugeSet("protocol_matrix_failed_tests", float64(status.FailedTests))
    
    for name, ps := range status.ProtocolStatus {
        prometheus.GaugeSet(fmt.Sprintf("protocol_matrix_status{protocol=\"%s\"}", name), 
                           statusToFloat(ps.Status))
        prometheus.GaugeSet(fmt.Sprintf("protocol_matrix_protocol_coverage{protocol=\"%s\"}", name), 
                           ps.CoveragePercent)
    }
}
```

## Troubleshooting

### Common Issues

1. **"Matrix file not found"**
   - Ensure you're running commands from the correct directory
   - Check that `protocol_matrix.yaml` exists

2. **"Validation failed"**
   - Run `make matrix-validate` to see specific errors
   - Check YAML syntax and required fields

3. **"No test results"**
   - Verify that test integration is updating the matrix
   - Check file permissions on matrix files

4. **"Time parsing errors"**
   - Ensure time fields are in RFC3339 format: `2024-07-08T14:00:00Z`
   - Don't leave time fields empty

### Debugging

Enable verbose output:
```bash
./bin/matrix -command=status -verbose
```

Check matrix configuration:
```bash
./bin/matrix -command=validate
```

Generate debug information:
```bash
./bin/matrix -command=analyze -json | jq '.'
```

## Best Practices

1. **Regular Updates**: Run matrix analysis after every test run
2. **Version Control**: Commit matrix status files to track progress over time  
3. **Automated Reports**: Generate reports in CI/CD pipelines
4. **Gap Monitoring**: Address high-severity gaps first
5. **Performance Tracking**: Monitor performance trends over time
6. **Integration Testing**: Test matrix updates in development environments first