# Protocol Testing Matrix Tracking

This system provides comprehensive tracking and management of testing matrices for all supported industrial protocols in the Bifrost gateway.

## Overview

The Protocol Testing Matrix system tracks:

- **Test Coverage**: What tests are defined and executed for each protocol
- **Test Results**: Current status, pass/fail counts, and coverage percentages  
- **Gap Analysis**: Missing tests or coverage gaps
- **Performance Targets**: Expected performance metrics for each protocol
- **Device Compatibility**: Virtual and real device testing status

## Supported Protocols

### Modbus
- **TCP Implementation**: Function codes 1-23, exception handling, performance tests
- **RTU Implementation**: Serial configurations, function codes, performance tests
- **Coverage**: Function codes, exception codes, data types, concurrent connections

### OPC UA
- **Security Policies**: None, Basic128Rsa15, Basic256, Basic256Sha256
- **Operations**: Browse, Read, Write, Subscribe, HistoryRead
- **Authentication**: Anonymous, Username, Certificate
- **Vendor Compatibility**: Schneider Electric, Siemens, Rockwell, ABB

### Ethernet/IP
- **Messaging**: Explicit and Implicit messaging
- **Connections**: Class1 and Class3 connections
- **Services**: Get/Set Attribute operations, Forward Open/Close
- **Device Profiles**: Generic Device, I/O Device, Drive Device
- **Vendor Compatibility**: Allen-Bradley, Schneider Electric, Omron

### S7 Protocol
- **CPU Families**: S7-300, S7-400, S7-1200, S7-1500
- **Operations**: DB Read/Write, Memory access, CPU info
- **Memory Areas**: DB, M, I, Q, T, C
- **Data Types**: BOOL, BYTE, WORD, DWORD, INT, DINT, REAL, STRING

## File Structure

```
protocol-matrix/
├── README.md                          # This file
├── go.mod                             # Go module definition
├── protocol_matrix.yaml               # Main configuration file
├── schema/
│   └── protocol_matrix_schema.yaml    # Schema definition
├── status/
│   ├── matrix_status.yaml            # Current status
│   ├── report.json                   # JSON report
│   └── report.html                   # HTML dashboard
├── cmd/
│   └── matrix/
│       └── main.go                   # CLI tool
└── internal/
    └── matrix/
        ├── types.go                  # Data structures
        ├── manager.go                # Core functionality
        └── manager_test.go           # Tests
```

## Usage

### CLI Commands

Build the matrix CLI tool:
```bash
make matrix-build
```

Check current status:
```bash
make matrix-status
# or
./bin/matrix -command=status -verbose
```

Analyze the matrix and generate reports:
```bash
make matrix-analyze
# or  
./bin/matrix -command=analyze
```

Validate configuration:
```bash
make matrix-validate
# or
./bin/matrix -command=validate
```

Generate reports:
```bash
make matrix-report
# or
./bin/matrix -command=report
```

Run all matrix operations:
```bash
make matrix
```

### JSON Output

Get status in JSON format:
```bash
./bin/matrix -command=status -json
```

### Update Test Results

Update results for a protocol (example):
```bash
./bin/matrix -command=update -protocol=modbus -implementation=tcp
```

## Configuration

### Protocol Matrix Configuration

The main configuration is in `protocol_matrix.yaml`. Example structure:

```yaml
schema_version: "1.0"
last_updated: "2024-07-08T14:00:00Z"

protocols:
  modbus:
    implementations:
      tcp:
        test_coverage:
          function_codes: [1, 2, 3, 4, 5, 6, 15, 16, 23]
          exception_codes: [1, 2, 3, 4]
          performance_tests: true
          concurrent_connections: 100
        virtual_devices:
          - "modbus_tcp_simulator"
        real_devices: []
        test_results:
          status: "not_run"
          passed: 0
          failed: 0
          coverage_percentage: 0.0

performance_targets:
  modbus_tcp:
    throughput_ops_per_sec: 10000
    latency_p95_ms: 10
    concurrent_connections: 100
```

### Test Execution Configuration

```yaml
test_execution:
  timeout: "30m"
  parallel_tests: true
  max_workers: 4
  retry_failed_tests: true
  max_retries: 3
  generate_html_report: true
  generate_json_report: true
```

## Integration

### CI/CD Integration

Add to GitHub Actions workflow:

```yaml
- name: Validate Protocol Matrix
  run: |
    cd go-gateway
    make matrix-validate

- name: Analyze Protocol Matrix  
  run: |
    cd go-gateway
    make matrix-analyze
    
- name: Upload Matrix Reports
  uses: actions/upload-artifact@v3
  with:
    name: protocol-matrix-reports
    path: go-gateway/protocol-matrix/status/
```

### Makefile Targets

The following targets are available in the main `go-gateway/Makefile`:

- `matrix-build`: Build the CLI tool
- `matrix-status`: Show current status
- `matrix-analyze`: Analyze and save status
- `matrix-validate`: Validate configuration
- `matrix-report`: Generate reports
- `matrix-test`: Run matrix tests
- `matrix`: Run all matrix operations

### Test Integration

Run matrix tests:
```bash
cd protocol-matrix
go test -v ./...
```

## Status Dashboard

The system generates both JSON and HTML reports:

### JSON Report (`status/report.json`)
- Machine-readable status
- Complete test results
- Gap analysis details
- Performance metrics

### HTML Dashboard (`status/report.html`)
- Human-readable overview
- Protocol status summary
- Visual coverage indicators
- Gap analysis with severity levels

## Gap Analysis

The system automatically identifies coverage gaps:

- **Missing virtual devices**: Protocols with no simulators configured
- **Missing performance tests**: Implementations without performance testing
- **Incomplete function coverage**: Missing function codes for Modbus
- **Missing security policies**: OPC UA without all security configurations

Gap severity levels:
- **High**: Critical functionality missing (no devices, no function codes)
- **Medium**: Performance or extended features missing
- **Low**: Nice-to-have features missing

## Performance Targets

Each protocol has defined performance targets:

| Protocol | Throughput (ops/sec) | Latency P95 (ms) | Connections |
|----------|---------------------|------------------|-------------|
| Modbus TCP | 10,000 | 10 | 100 |
| OPC UA | 5,000 | 20 | 50 |
| Ethernet/IP | 3,000 | 15 | 25 |
| S7 | 2,000 | 25 | 20 |

## Development

### Adding New Protocols

1. Update `protocol_matrix.yaml` with the new protocol configuration
2. Add performance targets in the `performance_targets` section
3. Update the schema if needed
4. Run validation: `make matrix-validate`

### Adding New Test Categories

1. Update the `TestCoverage` struct in `internal/matrix/types.go`
2. Update YAML tags and parsing
3. Add gap analysis logic in `manager.go`
4. Update tests

### Custom Reporting

The `Manager` type can be used programmatically:

```go
import "bifrost-gateway/protocol-matrix/internal/matrix"

manager := matrix.NewManager("protocol_matrix.yaml", "status.yaml")
if err := manager.LoadMatrix(); err != nil {
    log.Fatal(err)
}

status, err := manager.GenerateStatus()
if err != nil {
    log.Fatal(err)
}

// Use status for custom reporting
fmt.Printf("Overall coverage: %.1f%%\n", status.CoveragePercent)
```

## Contributing

1. Update protocol configurations in `protocol_matrix.yaml`
2. Add tests for new functionality
3. Update documentation
4. Run `make matrix` to validate changes
5. Include matrix reports in pull requests