package release

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// AutomationEngine handles automated release card generation and integration
type AutomationEngine struct {
	logger           *zap.Logger
	integrator       *BenchmarkIntegrator
	config           *AutomationConfig
	storageProvider  StorageProvider
}

// AutomationConfig configures the automation engine
type AutomationConfig struct {
	OutputDirectory      string   `yaml:"output_directory"`
	OutputFormats        []string `yaml:"output_formats"` // yaml, json, html, pdf
	AutoPublish          bool     `yaml:"auto_publish"`
	GitIntegration       bool     `yaml:"git_integration"`
	NotificationChannels []string `yaml:"notification_channels"`
	
	// CI/CD Integration
	TriggerOnTag         bool   `yaml:"trigger_on_tag"`
	TagPattern           string `yaml:"tag_pattern"`
	RequireManualApproval bool  `yaml:"require_manual_approval"`
	
	// Quality Gates
	MinPerformanceScore  float64 `yaml:"min_performance_score"`
	BlockOnRegression    bool    `yaml:"block_on_regression"`
	RequireTestCoverage  float64 `yaml:"require_test_coverage"`
}

// StorageProvider interface for storing release cards
type StorageProvider interface {
	Store(card *ReleaseCard, format string) error
	Load(version string) (*ReleaseCard, error)
	List() ([]*ReleaseCard, error)
	Delete(version string) error
}

// DefaultAutomationConfig returns default automation configuration
func DefaultAutomationConfig() *AutomationConfig {
	return &AutomationConfig{
		OutputDirectory:      "./release_cards",
		OutputFormats:        []string{"yaml", "json"},
		AutoPublish:          false,
		GitIntegration:       true,
		TriggerOnTag:         true,
		TagPattern:           "v*",
		RequireManualApproval: true,
		MinPerformanceScore:  80.0,
		BlockOnRegression:    true,
		RequireTestCoverage:  80.0,
	}
}

// NewAutomationEngine creates a new automation engine
func NewAutomationEngine(
	logger *zap.Logger,
	integrator *BenchmarkIntegrator,
	storage StorageProvider,
	config *AutomationConfig,
) *AutomationEngine {
	if config == nil {
		config = DefaultAutomationConfig()
	}
	
	return &AutomationEngine{
		logger:          logger,
		integrator:      integrator,
		config:          config,
		storageProvider: storage,
	}
}

// GenerateAndPublish generates a release card and publishes it according to configuration
func (ae *AutomationEngine) GenerateAndPublish(ctx context.Context, version, gitCommit, environment string) (*ReleaseCard, error) {
	ae.logger.Info("Starting automated release card generation",
		zap.String("version", version),
		zap.String("commit", gitCommit),
		zap.String("environment", environment),
	)
	
	// Create metadata
	metadata := CardMetadata{
		Version:     version,
		ReleaseDate: time.Now(),
		GitCommit:   gitCommit,
		Environment: environment,
		GeneratedAt: time.Now(),
	}
	
	// Generate release card
	releaseCard, err := ae.integrator.GenerateReleaseCard(ctx, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to generate release card: %w", err)
	}
	
	// Apply quality gates
	if err := ae.applyQualityGates(releaseCard); err != nil {
		return nil, fmt.Errorf("quality gate failed: %w", err)
	}
	
	// Store release card
	if err := ae.storeReleaseCard(releaseCard); err != nil {
		return nil, fmt.Errorf("failed to store release card: %w", err)
	}
	
	// Publish if auto-publish is enabled
	if ae.config.AutoPublish {
		if err := ae.publishReleaseCard(releaseCard); err != nil {
			ae.logger.Error("Failed to publish release card", zap.Error(err))
			// Don't fail the generation if publishing fails
		}
	}
	
	ae.logger.Info("Release card generation completed successfully",
		zap.String("version", version),
		zap.Float64("performance_score", releaseCard.Summary.PerformanceScore),
		zap.String("status", releaseCard.Summary.OverallStatus),
	)
	
	return releaseCard, nil
}

// applyQualityGates checks if the release meets quality requirements
func (ae *AutomationEngine) applyQualityGates(card *ReleaseCard) error {
	errors := make([]string, 0)
	
	// Check minimum performance score
	if card.Summary.PerformanceScore < ae.config.MinPerformanceScore {
		errors = append(errors, fmt.Sprintf(
			"Performance score %.1f below minimum threshold %.1f",
			card.Summary.PerformanceScore,
			ae.config.MinPerformanceScore,
		))
	}
	
	// Check for performance regressions
	if ae.config.BlockOnRegression && card.Summary.PerformanceRegression {
		errors = append(errors, "Performance regression detected")
	}
	
	// Check test coverage if specified
	if ae.config.RequireTestCoverage > 0 && card.Summary.TestCoverage < ae.config.RequireTestCoverage {
		errors = append(errors, fmt.Sprintf(
			"Test coverage %.1f%% below required %.1f%%",
			card.Summary.TestCoverage,
			ae.config.RequireTestCoverage,
		))
	}
	
	if len(errors) > 0 {
		ae.logger.Error("Quality gate failures",
			zap.Strings("failures", errors),
			zap.String("version", card.Metadata.Version),
		)
		return fmt.Errorf("quality gate failures: %v", errors)
	}
	
	return nil
}

// storeReleaseCard stores the release card in the configured formats
func (ae *AutomationEngine) storeReleaseCard(card *ReleaseCard) error {
	// Ensure output directory exists
	if err := os.MkdirAll(ae.config.OutputDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Store in each configured format
	for _, format := range ae.config.OutputFormats {
		if err := ae.storeInFormat(card, format); err != nil {
			return fmt.Errorf("failed to store in %s format: %w", format, err)
		}
	}
	
	// Store via storage provider if available
	if ae.storageProvider != nil {
		for _, format := range ae.config.OutputFormats {
			if err := ae.storageProvider.Store(card, format); err != nil {
				ae.logger.Warn("Failed to store via storage provider",
					zap.String("format", format),
					zap.Error(err),
				)
			}
		}
	}
	
	return nil
}

// storeInFormat stores the release card in a specific format
func (ae *AutomationEngine) storeInFormat(card *ReleaseCard, format string) error {
	filename := filepath.Join(ae.config.OutputDirectory, 
		fmt.Sprintf("release-card-%s.%s", card.Metadata.Version, format))
	
	var data []byte
	var err error
	
	switch format {
	case "yaml", "yml":
		data, err = card.ToYAML()
	case "json":
		data, err = card.ToJSON()
	case "html":
		data, err = ae.generateHTML(card)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return fmt.Errorf("failed to convert to %s: %w", format, err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}
	
	ae.logger.Info("Stored release card",
		zap.String("format", format),
		zap.String("file", filename),
	)
	
	return nil
}

// generateHTML generates an HTML representation of the release card
func (ae *AutomationEngine) generateHTML(card *ReleaseCard) ([]byte, error) {
	// Simple HTML template - in a real implementation, this would use a proper template engine
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Release Card - %s</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; }
        .section { margin: 20px 0; padding: 15px; border-left: 3px solid #007cba; }
        .metric { display: inline-block; margin: 10px; padding: 10px; background: #f9f9f9; border-radius: 3px; }
        .pass { color: green; }
        .fail { color: red; }
        .warning { color: orange; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Release Card - Version %s</h1>
        <p>Generated: %s</p>
        <p>Environment: %s</p>
        <p>Performance Score: <span class="%s">%.1f</span></p>
    </div>
    
    <div class="section">
        <h2>Performance Benchmarks</h2>
        %s
    </div>
    
    <div class="section">
        <h2>Summary</h2>
        <p>Overall Status: <span class="%s">%s</span></p>
        <p>Protocols Supported: %d</p>
        <p>Performance Regression: %t</p>
    </div>
</body>
</html>`,
		card.Metadata.Version,
		card.Metadata.Version,
		card.Metadata.GeneratedAt.Format("2006-01-02 15:04:05"),
		card.Metadata.Environment,
		ae.getScoreClass(card.Summary.PerformanceScore),
		card.Summary.PerformanceScore,
		ae.generatePerformanceHTML(card),
		card.Summary.OverallStatus,
		card.Summary.OverallStatus,
		card.Summary.ProtocolsSupported,
		card.Summary.PerformanceRegression,
	)
	
	return []byte(html), nil
}

// generatePerformanceHTML generates HTML for performance benchmarks
func (ae *AutomationEngine) generatePerformanceHTML(card *ReleaseCard) string {
	html := ""
	
	if card.PerformanceBenchmarks.ModbusTCP != nil {
		html += ae.generateProtocolHTML("Modbus TCP", card.PerformanceBenchmarks.ModbusTCP)
	}
	
	if card.PerformanceBenchmarks.EtherNetIP != nil {
		html += ae.generateProtocolHTML("EtherNet/IP", card.PerformanceBenchmarks.EtherNetIP)
	}
	
	if card.PerformanceBenchmarks.OPCUA != nil {
		html += ae.generateProtocolHTML("OPC UA", card.PerformanceBenchmarks.OPCUA)
	}
	
	return html
}

// generateProtocolHTML generates HTML for a single protocol's benchmarks
func (ae *AutomationEngine) generateProtocolHTML(name string, protocol *ProtocolBenchmark) string {
	return fmt.Sprintf(`
        <h3>%s</h3>
        <div class="metric">
            <strong>Single Connection Throughput:</strong>
            <span class="%s">%s (Target: %s)</span>
        </div>
        <div class="metric">
            <strong>Single Register Latency:</strong>
            <span class="%s">%s (Target: %s)</span>
        </div>
        <div class="metric">
            <strong>Memory Usage:</strong> %s
        </div>
        <div class="metric">
            <strong>CPU Usage:</strong> %s
        </div>`,
		name,
		protocol.Throughput.SingleConnection.Status,
		protocol.Throughput.SingleConnection.Measured,
		protocol.Throughput.SingleConnection.Target,
		protocol.Latency.SingleRegister.Status,
		protocol.Latency.SingleRegister.Measured,
		protocol.Latency.SingleRegister.Target,
		protocol.Resources.MemoryUsage,
		protocol.Resources.CPUUsage,
	)
}

// getScoreClass returns CSS class based on performance score
func (ae *AutomationEngine) getScoreClass(score float64) string {
	if score >= 85 {
		return "pass"
	} else if score >= 70 {
		return "warning"
	}
	return "fail"
}

// publishReleaseCard publishes the release card to configured channels
func (ae *AutomationEngine) publishReleaseCard(card *ReleaseCard) error {
	ae.logger.Info("Publishing release card",
		zap.String("version", card.Metadata.Version),
		zap.Strings("channels", ae.config.NotificationChannels),
	)
	
	// In a real implementation, this would publish to various channels:
	// - GitHub releases
	// - Documentation sites
	// - Slack/Teams notifications
	// - Email notifications
	// - Artifact repositories
	
	for _, channel := range ae.config.NotificationChannels {
		switch channel {
		case "github_release":
			ae.logger.Info("Would publish to GitHub release", zap.String("version", card.Metadata.Version))
		case "documentation_site":
			ae.logger.Info("Would publish to documentation site", zap.String("version", card.Metadata.Version))
		case "slack":
			ae.logger.Info("Would send Slack notification", zap.String("version", card.Metadata.Version))
		default:
			ae.logger.Warn("Unknown notification channel", zap.String("channel", channel))
		}
	}
	
	return nil
}

// LoadHistoricalCards loads historical release cards for trend analysis
func (ae *AutomationEngine) LoadHistoricalCards() error {
	if ae.storageProvider == nil {
		ae.logger.Warn("No storage provider configured, skipping historical data load")
		return nil
	}
	
	cards, err := ae.storageProvider.List()
	if err != nil {
		return fmt.Errorf("failed to load historical cards: %w", err)
	}
	
	ae.integrator.LoadHistoricalData(cards)
	
	ae.logger.Info("Loaded historical release cards",
		zap.Int("count", len(cards)),
	)
	
	return nil
}