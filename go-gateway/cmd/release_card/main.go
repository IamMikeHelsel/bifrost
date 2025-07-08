package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"bifrost-gateway/internal/performance"
	"bifrost-gateway/internal/release"
)

func main() {
	// Parse command-line flags
	var (
		version     = flag.String("version", "", "Release version (required)")
		gitCommit   = flag.String("commit", "", "Git commit hash")
		environment = flag.String("env", "production", "Environment (development, staging, production)")
		outputDir   = flag.String("output", "./release_cards", "Output directory for release cards")
		formats     = flag.String("formats", "yaml,json", "Output formats (yaml,json,html)")
		logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		dryRun      = flag.Bool("dry-run", false, "Dry run mode (don't save files)")
		loadHistory = flag.String("history", "", "Directory to load historical data from")
	)
	flag.Parse()

	if *version == "" {
		log.Fatal("Version is required. Use -version flag.")
	}

	// Set up logging
	logger := setupLogger(*logLevel)
	defer logger.Sync()

	logger.Info("Starting release card generation",
		zap.String("version", *version),
		zap.String("commit", *gitCommit),
		zap.String("environment", *environment),
		zap.String("output_dir", *outputDir),
		zap.String("formats", *formats),
		zap.Bool("dry_run", *dryRun),
	)

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Set up performance monitoring and benchmarking
	benchmarkConfig := &performance.BenchmarkConfig{
		EnableLatencyTests:     true,
		EnableThroughputTests:  true,
		EnableConcurrencyTests: true,
		EnableStressTests:      false, // Skip for faster generation
		EnableMemoryTests:      true,
		EnableEdgeTests:        false, // Skip for faster generation
		WarmupDuration:         10 * time.Second,
		TestDuration:           1 * time.Minute,
		CooldownDuration:       5 * time.Second,
		MaxConcurrentRequests:  1000,
		RequestsPerSecond:      500,
	}

	targets := &performance.PerformanceTargets{
		MaxLatencyMicroseconds:   1000,  // 1ms
		MinThroughputOpsPerSec:   1000,  // 1K ops/sec
		MaxConcurrentConnections: 1000,  // 1K connections
		MaxMemoryUsageMB:         100,   // 100MB
		MinTagsPerSecond:         10000, // 10K tags/sec
		MaxErrorRate:             0.001, // 0.1%
		MinSuccessRate:           0.999, // 99.9%
		MaxP99LatencyMicroseconds: 5000, // 5ms
		MaxCPUUsagePercent:       80.0,  // 80%
		MaxMemoryGrowthRate:      1.0,   // 1% per hour
		MinMemoryEfficiency:      0.9,   // 90%
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

	// Load historical data if specified
	if *loadHistory != "" {
		storage := release.NewFileStorageProvider(*loadHistory)
		cards, err := storage.List()
		if err != nil {
			logger.Warn("Failed to load historical data", zap.Error(err))
		} else {
			integrator.LoadHistoricalData(cards)
			logger.Info("Loaded historical data", zap.Int("count", len(cards)))
		}
	}

	// Create release card metadata
	metadata := release.CardMetadata{
		Version:     *version,
		ReleaseDate: time.Now(),
		GitCommit:   *gitCommit,
		Environment: *environment,
	}

	// Generate release card
	logger.Info("Generating release card with performance benchmarks...")
	releaseCard, err := integrator.GenerateReleaseCard(ctx, metadata)
	if err != nil {
		log.Fatalf("Failed to generate release card: %v", err)
	}

	logger.Info("Release card generated successfully",
		zap.String("version", releaseCard.Metadata.Version),
		zap.Float64("performance_score", releaseCard.Summary.PerformanceScore),
		zap.String("overall_status", releaseCard.Summary.OverallStatus),
		zap.Int("protocols_supported", releaseCard.Summary.ProtocolsSupported),
		zap.Bool("performance_regression", releaseCard.Summary.PerformanceRegression),
	)

	// Display results
	displayReleaseCard(releaseCard, logger)

	if !*dryRun {
		// Save release card files directly since we can't access private methods
		if err := saveReleaseCard(releaseCard, *outputDir, parseFormats(*formats)); err != nil {
			log.Fatalf("Failed to save release card: %v", err)
		}

		logger.Info("Release card saved successfully",
			zap.String("output_dir", *outputDir),
			zap.Strings("formats", parseFormats(*formats)),
		)
	} else {
		logger.Info("Dry run mode - release card not saved")
	}
}

func setupLogger(logLevel string) *zap.Logger {
	level := zapcore.InfoLevel
	switch logLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(level),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, _ := config.Build()
	return logger
}

func parseFormats(formatsStr string) []string {
	if formatsStr == "" {
		return []string{"yaml", "json"}
	}

	parts := strings.Split(formatsStr, ",")
	formats := make([]string, 0, len(parts))
	
	for _, part := range parts {
		format := strings.TrimSpace(part)
		if format == "yaml" || format == "json" || format == "html" {
			formats = append(formats, format)
		}
	}

	if len(formats) == 0 {
		return []string{"yaml", "json"}
	}

	return formats
}

func saveReleaseCard(card *release.ReleaseCard, outputDir string, formats []string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save in each format
	for _, format := range formats {
		filename := fmt.Sprintf("release-card-%s.%s", card.Metadata.Version, format)
		filepath := fmt.Sprintf("%s/%s", outputDir, filename)

		var data []byte
		var err error

		switch format {
		case "yaml", "yml":
			data, err = card.ToYAML()
		case "json":
			data, err = card.ToJSON()
		case "html":
			data, err = generateSimpleHTML(card)
		default:
			return fmt.Errorf("unsupported format: %s", format)
		}

		if err != nil {
			return fmt.Errorf("failed to convert to %s: %w", format, err)
		}

		if err := os.WriteFile(filepath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filepath, err)
		}
	}

	return nil
}

func generateSimpleHTML(card *release.ReleaseCard) ([]byte, error) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Release Card - %s</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .section { margin: 20px 0; padding: 15px; border-left: 3px solid #007cba; }
        .metric { display: inline-block; margin: 10px; padding: 10px; background: #f9f9f9; border-radius: 3px; }
        .pass { color: green; font-weight: bold; }
        .fail { color: red; font-weight: bold; }
        .warning { color: orange; font-weight: bold; }
        table { border-collapse: collapse; width: 100%%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Release Card - Version %s</h1>
        <p><strong>Generated:</strong> %s</p>
        <p><strong>Environment:</strong> %s</p>
        <p><strong>Git Commit:</strong> %s</p>
        <p><strong>Performance Score:</strong> <span class="%s">%.1f</span></p>
    </div>
    
    <div class="section">
        <h2>Summary</h2>
        <p><strong>Overall Status:</strong> <span class="%s">%s</span></p>
        <p><strong>Protocols Supported:</strong> %d</p>
        <p><strong>Performance Regression:</strong> %t</p>
    </div>
    
    <div class="section">
        <h2>Performance Benchmarks</h2>
        %s
    </div>
</body>
</html>`,
		card.Metadata.Version,
		card.Metadata.Version,
		card.Metadata.GeneratedAt.Format("2006-01-02 15:04:05"),
		card.Metadata.Environment,
		card.Metadata.GitCommit,
		getScoreClass(card.Summary.PerformanceScore),
		card.Summary.PerformanceScore,
		card.Summary.OverallStatus,
		card.Summary.OverallStatus,
		card.Summary.ProtocolsSupported,
		card.Summary.PerformanceRegression,
		generatePerformanceHTML(card),
	)

	return []byte(html), nil
}

func generatePerformanceHTML(card *release.ReleaseCard) string {
	html := ""

	if card.PerformanceBenchmarks.ModbusTCP != nil {
		html += generateProtocolHTML("Modbus TCP", card.PerformanceBenchmarks.ModbusTCP)
	}
	if card.PerformanceBenchmarks.EtherNetIP != nil {
		html += generateProtocolHTML("EtherNet/IP", card.PerformanceBenchmarks.EtherNetIP)
	}
	if card.PerformanceBenchmarks.OPCUA != nil {
		html += generateProtocolHTML("OPC UA", card.PerformanceBenchmarks.OPCUA)
	}

	return html
}

func generateProtocolHTML(name string, protocol *release.ProtocolBenchmark) string {
	return fmt.Sprintf(`
        <h3>%s</h3>
        <table>
            <tr><th>Metric</th><th>Target</th><th>Measured</th><th>Status</th></tr>
            <tr>
                <td>Single Connection Throughput</td>
                <td>%s</td>
                <td>%s</td>
                <td><span class="%s">%s</span></td>
            </tr>
            <tr>
                <td>Concurrent 100 Throughput</td>
                <td>%s</td>
                <td>%s</td>
                <td><span class="%s">%s</span></td>
            </tr>
            <tr>
                <td>Single Register Latency</td>
                <td>%s</td>
                <td>%s</td>
                <td><span class="%s">%s</span></td>
            </tr>
            <tr>
                <td>Memory Usage</td>
                <td>-</td>
                <td>%s</td>
                <td>-</td>
            </tr>
            <tr>
                <td>CPU Usage</td>
                <td>-</td>
                <td>%s</td>
                <td>-</td>
            </tr>
        </table>`,
		name,
		protocol.Throughput.SingleConnection.Target,
		protocol.Throughput.SingleConnection.Measured,
		protocol.Throughput.SingleConnection.Status,
		protocol.Throughput.SingleConnection.Status,
		protocol.Throughput.Concurrent100.Target,
		protocol.Throughput.Concurrent100.Measured,
		protocol.Throughput.Concurrent100.Status,
		protocol.Throughput.Concurrent100.Status,
		protocol.Latency.SingleRegister.Target,
		protocol.Latency.SingleRegister.Measured,
		protocol.Latency.SingleRegister.Status,
		protocol.Latency.SingleRegister.Status,
		protocol.Resources.MemoryUsage,
		protocol.Resources.CPUUsage,
	)
}

func getScoreClass(score float64) string {
	if score >= 85 {
		return "pass"
	} else if score >= 70 {
		return "warning"
	}
	return "fail"
}

func displayReleaseCard(card *release.ReleaseCard, logger *zap.Logger) {
	logger.Info("📋 RELEASE CARD SUMMARY 📋")
	logger.Info(strings.Repeat("=", 80))

	// Metadata
	logger.Info("📦 RELEASE INFORMATION",
		zap.String("version", card.Metadata.Version),
		zap.Time("release_date", card.Metadata.ReleaseDate),
		zap.String("git_commit", card.Metadata.GitCommit),
		zap.String("environment", card.Metadata.Environment),
	)

	// Overall summary
	logger.Info("🎯 OVERALL SUMMARY",
		zap.String("status", card.Summary.OverallStatus),
		zap.Float64("performance_score", card.Summary.PerformanceScore),
		zap.Int("protocols_supported", card.Summary.ProtocolsSupported),
		zap.Bool("performance_regression", card.Summary.PerformanceRegression),
	)

	// Performance benchmarks
	if card.PerformanceBenchmarks.ModbusTCP != nil {
		displayProtocolBenchmark("Modbus TCP", card.PerformanceBenchmarks.ModbusTCP, logger)
	}
	if card.PerformanceBenchmarks.EtherNetIP != nil {
		displayProtocolBenchmark("EtherNet/IP", card.PerformanceBenchmarks.EtherNetIP, logger)
	}
	if card.PerformanceBenchmarks.OPCUA != nil {
		displayProtocolBenchmark("OPC UA", card.PerformanceBenchmarks.OPCUA, logger)
	}

	// Trend analysis
	if card.Summary.TrendAnalysis != nil {
		logger.Info("📈 TREND ANALYSIS",
			zap.String("latency_trend", card.Summary.TrendAnalysis.LatencyTrend),
			zap.String("throughput_trend", card.Summary.TrendAnalysis.ThroughputTrend),
			zap.String("memory_trend", card.Summary.TrendAnalysis.MemoryTrend),
			zap.Float64("latency_change_percent", card.Summary.TrendAnalysis.ComparisonToPrevious.LatencyChange),
			zap.Float64("throughput_change_percent", card.Summary.TrendAnalysis.ComparisonToPrevious.ThroughputChange),
		)
	}

	// Known issues
	if len(card.Summary.KnownIssues) > 0 {
		logger.Info("⚠️  KNOWN ISSUES",
			zap.Strings("issues", card.Summary.KnownIssues),
		)
	}

	logger.Info(strings.Repeat("=", 80))
}

func displayProtocolBenchmark(name string, benchmark *release.ProtocolBenchmark, logger *zap.Logger) {
	logger.Info(fmt.Sprintf("🔌 %s PERFORMANCE", name),
		zap.String("throughput_single", fmt.Sprintf("%s (%s)", benchmark.Throughput.SingleConnection.Measured, benchmark.Throughput.SingleConnection.Status)),
		zap.String("throughput_concurrent", fmt.Sprintf("%s (%s)", benchmark.Throughput.Concurrent100.Measured, benchmark.Throughput.Concurrent100.Status)),
		zap.String("latency_single", fmt.Sprintf("%s (%s)", benchmark.Latency.SingleRegister.Measured, benchmark.Latency.SingleRegister.Status)),
		zap.String("memory_usage", benchmark.Resources.MemoryUsage),
		zap.String("cpu_usage", benchmark.Resources.CPUUsage),
	)
}