# Release Card System

The release card system provides automated performance benchmarking integration for tracking and reporting performance metrics across releases. It generates comprehensive release cards that include performance benchmarks, trend analysis, and regression detection.

## Features

- **Performance Data Collection**: Automated collection of throughput, latency, and resource usage metrics
- **Benchmark Execution**: Integration with the existing performance benchmark suite
- **Trend Analysis**: Performance trend tracking across releases
- **Regression Detection**: Automated detection of performance regressions
- **Multiple Output Formats**: YAML, JSON, and HTML output formats
- **Quality Gates**: Configurable performance thresholds for CI/CD integration

## Usage

### Command Line Tool

Generate a release card:

```bash
# Basic usage
./bin/release-card -version v1.0.0 -commit abc123 -env production

# With custom output directory and formats
./bin/release-card -version v1.0.0 -commit abc123 -env production \
  -output ./release_cards -formats yaml,json,html

# Dry run (don't save files)
./bin/release-card -version v1.0.0 -commit abc123 -env test -dry-run

# Load historical data for trend analysis
./bin/release-card -version v1.1.0 -commit def456 -env production \
  -history ./previous_release_cards
```

### Makefile Targets

```bash
# Generate release card using environment variables
make release-card VERSION=v1.0.0 COMMIT=abc123 ENV=production

# Generate with custom parameters
make release-card-custom VERSION=v1.0.0 COMMIT=abc123 ENV=production \
  OUTPUT=./custom_output FORMATS=yaml,json,html
```

### Programmatic Usage

```go
package main

import (
    "context"
    "time"
    
    "go.uber.org/zap"
    "bifrost-gateway/internal/performance"
    "bifrost-gateway/internal/release"
)

func generateReleaseCard() {
    logger := zap.NewNop()
    
    // Set up benchmark configuration
    benchmarkConfig := &performance.BenchmarkConfig{
        EnableLatencyTests:     true,
        EnableThroughputTests:  true,
        EnableConcurrencyTests: true,
        WarmupDuration:         10 * time.Second,
        TestDuration:           1 * time.Minute,
        MaxConcurrentRequests:  1000,
        RequestsPerSecond:      500,
    }
    
    targets := &performance.PerformanceTargets{
        MaxLatencyMicroseconds:   1000,  // 1ms
        MinThroughputOpsPerSec:   1000,  // 1K ops/sec
        MaxConcurrentConnections: 1000,  // 1K connections
        MaxMemoryUsageMB:         100,   // 100MB
    }
    
    benchmarkSuite := performance.NewBenchmarkSuite(benchmarkConfig, targets, logger)
    
    // Set up release card integration
    integratorConfig := &release.IntegratorConfig{
        EnableTrendAnalysis:       true,
        EnableRegressionDetection: true,
        HistoryRetentionDays:      90,
        RegressionThreshold:       10.0,
        MinHistoryForTrends:       3,
    }
    
    integrator := release.NewBenchmarkIntegrator(logger, benchmarkSuite, nil, integratorConfig)
    
    // Create metadata
    metadata := release.CardMetadata{
        Version:     "v1.0.0",
        ReleaseDate: time.Now(),
        GitCommit:   "abc123",
        Environment: "production",
    }
    
    // Generate release card
    ctx := context.Background()
    releaseCard, err := integrator.GenerateReleaseCard(ctx, metadata)
    if err != nil {
        panic(err)
    }
    
    // Save to file
    yamlData, _ := releaseCard.ToYAML()
    // ... save yamlData to file
}
```

## Release Card Format

### YAML Example

```yaml
metadata:
  version: v1.0.0
  release_date: 2025-07-08T14:18:03.615Z
  git_commit: abc123
  environment: production
  generated_at: 2025-07-08T14:18:10.750Z

performance_benchmarks:
  modbus_tcp:
    throughput:
      single_connection:
        target: ">1000 regs/sec"
        measured: "1247 regs/sec"
        status: "pass"
        value: 1247
      concurrent_100:
        target: ">50000 regs/sec"
        measured: "67000 regs/sec"
        status: "pass"
        value: 67000
    latency:
      single_register:
        target: "<1ms"
        measured: "0.7ms"
        status: "pass"
        value: 700µs
    resources:
      memory_usage: "12MB"
      cpu_usage: "5%"
      connections: "100"

summary:
  overall_status: "pass"
  performance_score: 85.0
  protocols_supported: 1
  performance_regression: false
  trend_analysis:
    latency_trend: "improving"
    throughput_trend: "stable"
    memory_trend: "stable"
    comparison_to_previous:
      latency_change_percent: -5.2
      throughput_change_percent: 2.1
      memory_change_percent: 0.8
      score_change_percent: 3.4
```

## Configuration

### Integrator Configuration

```go
type IntegratorConfig struct {
    EnableTrendAnalysis       bool    `yaml:"enable_trend_analysis"`
    EnableRegressionDetection bool    `yaml:"enable_regression_detection"`
    HistoryRetentionDays      int     `yaml:"history_retention_days"`
    RegressionThreshold       float64 `yaml:"regression_threshold"` // Percentage
    MinHistoryForTrends       int     `yaml:"min_history_for_trends"`
}
```

### Automation Configuration

```go
type AutomationConfig struct {
    OutputDirectory      string   `yaml:"output_directory"`
    OutputFormats        []string `yaml:"output_formats"` // yaml, json, html
    AutoPublish          bool     `yaml:"auto_publish"`
    GitIntegration       bool     `yaml:"git_integration"`
    
    // Quality Gates
    MinPerformanceScore  float64 `yaml:"min_performance_score"`
    BlockOnRegression    bool    `yaml:"block_on_regression"`
    RequireTestCoverage  float64 `yaml:"require_test_coverage"`
}
```

## Quality Gates

The system supports configurable quality gates for CI/CD integration:

- **Performance Score Threshold**: Minimum performance score required
- **Regression Detection**: Block releases with performance regressions
- **Test Coverage**: Minimum test coverage requirement

Example usage in CI/CD:

```bash
# Generate release card and fail if quality gates don't pass
./bin/release-card -version $VERSION -commit $GIT_COMMIT -env production
if [ $? -ne 0 ]; then
    echo "Quality gates failed!"
    exit 1
fi
```

## Trend Analysis

The system tracks performance trends across releases:

- **Latency Trends**: Tracks latency improvements/degradations
- **Throughput Trends**: Monitors throughput changes
- **Memory Trends**: Tracks memory usage patterns
- **Comparison Metrics**: Percentage changes from previous releases

Trend categories:
- `improving`: Performance is getting better
- `stable`: Performance is consistent
- `degrading`: Performance is getting worse
- `volatile`: Mixed or inconsistent trends

## Integration with CI/CD

Example GitHub Actions workflow:

```yaml
name: Release Card Generation

on:
  push:
    tags:
      - 'v*'

jobs:
  release-card:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version: 1.22
      
      - name: Build release card tool
        run: |
          cd go-gateway
          go build -o bin/release-card ./cmd/release_card/main.go
      
      - name: Generate release card
        run: |
          cd go-gateway
          ./bin/release-card \
            -version ${{ github.ref_name }} \
            -commit ${{ github.sha }} \
            -env production \
            -output ./release_cards \
            -formats yaml,json,html
      
      - name: Upload release card
        uses: actions/upload-artifact@v3
        with:
          name: release-card-${{ github.ref_name }}
          path: go-gateway/release_cards/
```

## Storage Providers

The system supports different storage backends:

### File Storage
```go
storage := release.NewFileStorageProvider("./release_cards")
```

### In-Memory Storage (for testing)
```go
storage := release.NewInMemoryStorageProvider()
```

### Custom Storage
Implement the `StorageProvider` interface:

```go
type StorageProvider interface {
    Store(card *ReleaseCard, format string) error
    Load(version string) (*ReleaseCard, error)
    List() ([]*ReleaseCard, error)
    Delete(version string) error
}
```

## Testing

Run the release card system tests:

```bash
cd go-gateway
go test ./internal/release/
```

Example test output:
```
ok      bifrost-gateway/internal/release    0.005s
```

## Performance Targets

Configure performance targets for your protocols:

```go
targets := &performance.PerformanceTargets{
    MaxLatencyMicroseconds:   1000,  // 1ms
    MinThroughputOpsPerSec:   1000,  // 1K ops/sec
    MaxConcurrentConnections: 1000,  // 1K connections
    MaxMemoryUsageMB:         100,   // 100MB
    MinTagsPerSecond:         10000, // 10K tags/sec
    MaxErrorRate:             0.001, // 0.1%
    MinSuccessRate:           0.999, // 99.9%
    MaxP99LatencyMicroseconds: 5000, // 5ms
}
```

## Future Enhancements

- Protocol-specific performance targets
- Integration with external monitoring systems
- Automated performance alerting
- Historical trend visualization
- Performance comparison across branches
- Integration with artifact repositories