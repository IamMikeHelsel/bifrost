package performance

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// PerformanceMonitor provides comprehensive performance monitoring and alerting
type PerformanceMonitor struct {
	logger   *zap.Logger
	config   *MonitoringConfig
	metrics  *MonitoringMetrics
	alerting *AlertManager

	// Prometheus metrics
	promMetrics *PrometheusMetrics

	// Data collection
	collectors map[string]Collector

	// HTTP server for metrics
	server *http.Server

	// Real-time monitoring
	realTimeData *RealTimeData

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	mutex  sync.RWMutex
}

// MonitoringConfig defines monitoring configuration
type MonitoringConfig struct {
	Enabled            bool          `yaml:"enabled"`
	MetricsPort        int           `yaml:"metrics_port"`
	MetricsPath        string        `yaml:"metrics_path"`
	CollectionInterval time.Duration `yaml:"collection_interval"`

	// Prometheus configuration
	EnablePrometheus    bool   `yaml:"enable_prometheus"`
	PrometheusNamespace string `yaml:"prometheus_namespace"`

	// Alert configuration
	EnableAlerting     bool          `yaml:"enable_alerting"`
	AlertCheckInterval time.Duration `yaml:"alert_check_interval"`

	// Performance thresholds
	Thresholds PerformanceThresholds `yaml:"thresholds"`

	// Data retention
	MetricsRetention time.Duration `yaml:"metrics_retention"`
	MaxDataPoints    int           `yaml:"max_data_points"`

	// Real-time monitoring
	EnableRealTime   bool          `yaml:"enable_real_time"`
	RealTimeInterval time.Duration `yaml:"real_time_interval"`
	WebSocketPort    int           `yaml:"websocket_port"`
}

// PerformanceThresholds defines alerting thresholds
type PerformanceThresholds struct {
	// Latency thresholds (microseconds)
	MaxLatency     int64 `yaml:"max_latency"`
	WarningLatency int64 `yaml:"warning_latency"`

	// Throughput thresholds (ops/sec)
	MinThroughput     float64 `yaml:"min_throughput"`
	WarningThroughput float64 `yaml:"warning_throughput"`

	// Resource thresholds
	MaxCPUPercent  float64 `yaml:"max_cpu_percent"`
	MaxMemoryMB    int64   `yaml:"max_memory_mb"`
	MaxGoroutines  int     `yaml:"max_goroutines"`
	MaxConnections int     `yaml:"max_connections"`

	// Error rate thresholds
	MaxErrorRate     float64 `yaml:"max_error_rate"`
	WarningErrorRate float64 `yaml:"warning_error_rate"`

	// Network thresholds
	MaxNetworkLatency int64   `yaml:"max_network_latency"`
	MaxPacketLoss     float64 `yaml:"max_packet_loss"`
}

// MonitoringMetrics holds comprehensive performance metrics
type MonitoringMetrics struct {
	// Core performance metrics
	AverageLatency time.Duration
	P50Latency     time.Duration
	P95Latency     time.Duration
	P99Latency     time.Duration
	MaxLatency     time.Duration

	ThroughputPerSecond float64
	ErrorRate           float64
	SuccessRate         float64

	// Resource metrics
	CPUUsagePercent float64
	MemoryUsageMB   float64
	GoroutineCount  int
	ConnectionCount int

	// Network metrics
	NetworkLatency       time.Duration
	PacketLossRate       float64
	BandwidthUtilization float64

	// Gateway-specific metrics
	DevicesConnected    int
	TagsProcessed       int64
	BatchesProcessed    int64
	CircuitBreakerTrips int64

	// Trend data
	LatencyTrend    []float64
	ThroughputTrend []float64
	ErrorRateTrend  []float64
	CPUTrend        []float64
	MemoryTrend     []float64
}

// PrometheusMetrics holds Prometheus metric definitions
type PrometheusMetrics struct {
	// Performance metrics
	RequestDuration  *prometheus.HistogramVec
	RequestsTotal    *prometheus.CounterVec
	RequestsInFlight prometheus.Gauge

	// Resource metrics
	CPUUsage        prometheus.Gauge
	MemoryUsage     prometheus.Gauge
	GoroutineCount  prometheus.Gauge
	ConnectionCount prometheus.Gauge

	// Gateway metrics
	DevicesConnected    prometheus.Gauge
	TagsProcessed       *prometheus.CounterVec
	BatchesProcessed    prometheus.Counter
	CircuitBreakerState *prometheus.GaugeVec

	// Error metrics
	ErrorsTotal *prometheus.CounterVec
	ErrorRate   prometheus.Gauge

	// Network metrics
	NetworkLatency prometheus.Gauge
	PacketLoss     prometheus.Gauge
	Bandwidth      prometheus.Gauge
}

// AlertManager manages performance alerting
type AlertManager struct {
	logger *zap.Logger
	config *MonitoringConfig

	// Alert state
	alerts map[string]*Alert
	mutex  sync.RWMutex

	// Alert channels
	alertChan chan *Alert

	// Alert history
	alertHistory []*Alert
}

// Alert represents a performance alert
type Alert struct {
	ID         string
	Type       string
	Severity   string
	Message    string
	Value      float64
	Threshold  float64
	Timestamp  time.Time
	Resolved   bool
	ResolvedAt time.Time

	// Context
	DeviceID   string
	MetricName string
	Tags       map[string]string
}

// RealTimeData manages real-time monitoring data
type RealTimeData struct {
	mutex sync.RWMutex

	// Current metrics
	CurrentLatency     time.Duration
	CurrentThroughput  float64
	CurrentCPU         float64
	CurrentMemory      float64
	CurrentConnections int

	// Time series data (last N points)
	LatencyHistory    []DataPoint
	ThroughputHistory []DataPoint
	CPUHistory        []DataPoint
	MemoryHistory     []DataPoint

	// WebSocket clients
	clients map[*WebSocketClient]bool

	// Update channel
	updateChan chan MetricUpdate
}

// DataPoint represents a time-series data point
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// MetricUpdate represents a metric update
type MetricUpdate struct {
	MetricName string
	Value      float64
	Timestamp  time.Time
	Tags       map[string]string
}

// WebSocketClient represents a WebSocket client connection
type WebSocketClient struct {
	conn   interface{} // WebSocket connection
	sendCh chan []byte
	doneCh chan struct{}
}

// Collector interface for metric collection
type Collector interface {
	Collect() (map[string]float64, error)
	Name() string
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(config *MonitoringConfig, logger *zap.Logger) *PerformanceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &PerformanceMonitor{
		logger:     logger,
		config:     config,
		metrics:    &MonitoringMetrics{},
		collectors: make(map[string]Collector),
		ctx:        ctx,
		cancel:     cancel,
	}

	if config.Enabled {
		monitor.initialize()
	}

	return monitor
}

// initialize sets up the performance monitor
func (pm *PerformanceMonitor) initialize() {
	// Initialize Prometheus metrics
	if pm.config.EnablePrometheus {
		pm.initPrometheusMetrics()
	}

	// Initialize alerting
	if pm.config.EnableAlerting {
		pm.alerting = NewAlertManager(pm.config, pm.logger)
	}

	// Initialize real-time monitoring
	if pm.config.EnableRealTime {
		pm.realTimeData = NewRealTimeData()
	}

	// Register standard collectors
	pm.registerStandardCollectors()

	// Start monitoring loops
	go pm.metricsCollectionLoop()

	if pm.config.EnableAlerting {
		go pm.alertCheckLoop()
	}

	if pm.config.EnableRealTime {
		go pm.realTimeUpdateLoop()
	}

	// Start HTTP server
	go pm.startMetricsServer()

	pm.logger.Info("Performance monitor initialized",
		zap.Int("metrics_port", pm.config.MetricsPort),
		zap.Bool("prometheus_enabled", pm.config.EnablePrometheus),
		zap.Bool("alerting_enabled", pm.config.EnableAlerting),
		zap.Bool("real_time_enabled", pm.config.EnableRealTime),
	)
}

// initPrometheusMetrics initializes Prometheus metrics
func (pm *PerformanceMonitor) initPrometheusMetrics() {
	namespace := pm.config.PrometheusNamespace

	pm.promMetrics = &PrometheusMetrics{
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_duration_seconds",
				Help:      "Request duration in seconds",
				Buckets:   []float64{.0001, .0005, .001, .005, .01, .05, .1, .5, 1, 2.5, 5, 10},
			},
			[]string{"device_id", "operation", "protocol"},
		),

		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "Total number of requests",
			},
			[]string{"device_id", "operation", "protocol", "status"},
		),

		RequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "requests_in_flight",
				Help:      "Number of requests currently being processed",
			},
		),

		CPUUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "cpu_usage_percent",
				Help:      "CPU usage percentage",
			},
		),

		MemoryUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_usage_bytes",
				Help:      "Memory usage in bytes",
			},
		),

		GoroutineCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "goroutines_total",
				Help:      "Number of active goroutines",
			},
		),

		ConnectionCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "connections_active",
				Help:      "Number of active connections",
			},
		),

		DevicesConnected: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "devices_connected",
				Help:      "Number of connected devices",
			},
		),

		TagsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "tags_processed_total",
				Help:      "Total number of tags processed",
			},
			[]string{"device_id", "protocol"},
		),

		BatchesProcessed: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "batches_processed_total",
				Help:      "Total number of batches processed",
			},
		),

		CircuitBreakerState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_state",
				Help:      "Circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
			[]string{"device_id"},
		),

		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors_total",
				Help:      "Total number of errors",
			},
			[]string{"device_id", "error_type", "protocol"},
		),

		ErrorRate: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "error_rate",
				Help:      "Current error rate",
			},
		),

		NetworkLatency: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "network_latency_seconds",
				Help:      "Network latency in seconds",
			},
		),

		PacketLoss: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "packet_loss_rate",
				Help:      "Packet loss rate",
			},
		),

		Bandwidth: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "bandwidth_utilization",
				Help:      "Bandwidth utilization percentage",
			},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		pm.promMetrics.RequestDuration,
		pm.promMetrics.RequestsTotal,
		pm.promMetrics.RequestsInFlight,
		pm.promMetrics.CPUUsage,
		pm.promMetrics.MemoryUsage,
		pm.promMetrics.GoroutineCount,
		pm.promMetrics.ConnectionCount,
		pm.promMetrics.DevicesConnected,
		pm.promMetrics.TagsProcessed,
		pm.promMetrics.BatchesProcessed,
		pm.promMetrics.CircuitBreakerState,
		pm.promMetrics.ErrorsTotal,
		pm.promMetrics.ErrorRate,
		pm.promMetrics.NetworkLatency,
		pm.promMetrics.PacketLoss,
		pm.promMetrics.Bandwidth,
	)
}

// registerStandardCollectors registers standard metric collectors
func (pm *PerformanceMonitor) registerStandardCollectors() {
	pm.collectors["runtime"] = NewRuntimeCollector()
	pm.collectors["gateway"] = NewGatewayCollector()
	pm.collectors["network"] = NewNetworkCollector()
}

// metricsCollectionLoop runs the main metrics collection loop
func (pm *PerformanceMonitor) metricsCollectionLoop() {
	ticker := time.NewTicker(pm.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.collectMetrics()
			pm.updatePrometheusMetrics()
		}
	}
}

// collectMetrics collects metrics from all collectors
func (pm *PerformanceMonitor) collectMetrics() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	allMetrics := make(map[string]float64)

	// Collect from all registered collectors
	for name, collector := range pm.collectors {
		metrics, err := collector.Collect()
		if err != nil {
			pm.logger.Error("Failed to collect metrics",
				zap.String("collector", name),
				zap.Error(err),
			)
			continue
		}

		// Merge metrics
		for key, value := range metrics {
			allMetrics[fmt.Sprintf("%s.%s", name, key)] = value
		}
	}

	// Update internal metrics
	pm.updateInternalMetrics(allMetrics)

	// Update real-time data
	if pm.config.EnableRealTime {
		pm.updateRealTimeData(allMetrics)
	}
}

// updateInternalMetrics updates internal metric structures
func (pm *PerformanceMonitor) updateInternalMetrics(metrics map[string]float64) {
	// Update core performance metrics
	if latency, exists := metrics["gateway.average_latency"]; exists {
		pm.metrics.AverageLatency = time.Duration(latency * float64(time.Microsecond))
	}

	if throughput, exists := metrics["gateway.throughput"]; exists {
		pm.metrics.ThroughputPerSecond = throughput
	}

	if errorRate, exists := metrics["gateway.error_rate"]; exists {
		pm.metrics.ErrorRate = errorRate
		pm.metrics.SuccessRate = 1.0 - errorRate
	}

	// Update resource metrics
	if cpu, exists := metrics["runtime.cpu_percent"]; exists {
		pm.metrics.CPUUsagePercent = cpu
	}

	if memory, exists := metrics["runtime.memory_mb"]; exists {
		pm.metrics.MemoryUsageMB = memory
	}

	if goroutines, exists := metrics["runtime.goroutines"]; exists {
		pm.metrics.GoroutineCount = int(goroutines)
	}

	// Update trend data
	pm.updateTrendData()
}

// updateTrendData updates historical trend data
func (pm *PerformanceMonitor) updateTrendData() {
	maxPoints := pm.config.MaxDataPoints

	// Update latency trend
	pm.metrics.LatencyTrend = append(pm.metrics.LatencyTrend, pm.metrics.AverageLatency.Seconds())
	if len(pm.metrics.LatencyTrend) > maxPoints {
		pm.metrics.LatencyTrend = pm.metrics.LatencyTrend[1:]
	}

	// Update throughput trend
	pm.metrics.ThroughputTrend = append(pm.metrics.ThroughputTrend, pm.metrics.ThroughputPerSecond)
	if len(pm.metrics.ThroughputTrend) > maxPoints {
		pm.metrics.ThroughputTrend = pm.metrics.ThroughputTrend[1:]
	}

	// Update error rate trend
	pm.metrics.ErrorRateTrend = append(pm.metrics.ErrorRateTrend, pm.metrics.ErrorRate)
	if len(pm.metrics.ErrorRateTrend) > maxPoints {
		pm.metrics.ErrorRateTrend = pm.metrics.ErrorRateTrend[1:]
	}

	// Update CPU trend
	pm.metrics.CPUTrend = append(pm.metrics.CPUTrend, pm.metrics.CPUUsagePercent)
	if len(pm.metrics.CPUTrend) > maxPoints {
		pm.metrics.CPUTrend = pm.metrics.CPUTrend[1:]
	}

	// Update memory trend
	pm.metrics.MemoryTrend = append(pm.metrics.MemoryTrend, pm.metrics.MemoryUsageMB)
	if len(pm.metrics.MemoryTrend) > maxPoints {
		pm.metrics.MemoryTrend = pm.metrics.MemoryTrend[1:]
	}
}

// updatePrometheusMetrics updates Prometheus metrics
func (pm *PerformanceMonitor) updatePrometheusMetrics() {
	if !pm.config.EnablePrometheus || pm.promMetrics == nil {
		return
	}

	// Update resource metrics
	pm.promMetrics.CPUUsage.Set(pm.metrics.CPUUsagePercent)
	pm.promMetrics.MemoryUsage.Set(pm.metrics.MemoryUsageMB * 1024 * 1024) // Convert MB to bytes
	pm.promMetrics.GoroutineCount.Set(float64(pm.metrics.GoroutineCount))
	pm.promMetrics.ConnectionCount.Set(float64(pm.metrics.ConnectionCount))

	// Update gateway metrics
	pm.promMetrics.DevicesConnected.Set(float64(pm.metrics.DevicesConnected))
	pm.promMetrics.ErrorRate.Set(pm.metrics.ErrorRate)

	// Update network metrics
	pm.promMetrics.NetworkLatency.Set(pm.metrics.NetworkLatency.Seconds())
	pm.promMetrics.PacketLoss.Set(pm.metrics.PacketLossRate)
	pm.promMetrics.Bandwidth.Set(pm.metrics.BandwidthUtilization)
}

// alertCheckLoop runs the alert checking loop
func (pm *PerformanceMonitor) alertCheckLoop() {
	ticker := time.NewTicker(pm.config.AlertCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.checkAlerts()
		}
	}
}

// checkAlerts checks for alert conditions
func (pm *PerformanceMonitor) checkAlerts() {
	thresholds := pm.config.Thresholds

	// Check latency alerts
	if pm.metrics.AverageLatency.Microseconds() > thresholds.MaxLatency {
		pm.triggerAlert("latency_critical", "CRITICAL", "Average latency exceeds maximum threshold",
			float64(pm.metrics.AverageLatency.Microseconds()), float64(thresholds.MaxLatency))
	} else if pm.metrics.AverageLatency.Microseconds() > thresholds.WarningLatency {
		pm.triggerAlert("latency_warning", "WARNING", "Average latency exceeds warning threshold",
			float64(pm.metrics.AverageLatency.Microseconds()), float64(thresholds.WarningLatency))
	}

	// Check throughput alerts
	if pm.metrics.ThroughputPerSecond < thresholds.MinThroughput {
		pm.triggerAlert("throughput_critical", "CRITICAL", "Throughput below minimum threshold",
			pm.metrics.ThroughputPerSecond, thresholds.MinThroughput)
	} else if pm.metrics.ThroughputPerSecond < thresholds.WarningThroughput {
		pm.triggerAlert("throughput_warning", "WARNING", "Throughput below warning threshold",
			pm.metrics.ThroughputPerSecond, thresholds.WarningThroughput)
	}

	// Check resource alerts
	if pm.metrics.CPUUsagePercent > thresholds.MaxCPUPercent {
		pm.triggerAlert("cpu_critical", "CRITICAL", "CPU usage exceeds maximum threshold",
			pm.metrics.CPUUsagePercent, thresholds.MaxCPUPercent)
	}

	if pm.metrics.MemoryUsageMB > float64(thresholds.MaxMemoryMB) {
		pm.triggerAlert("memory_critical", "CRITICAL", "Memory usage exceeds maximum threshold",
			pm.metrics.MemoryUsageMB, float64(thresholds.MaxMemoryMB))
	}

	// Check error rate alerts
	if pm.metrics.ErrorRate > thresholds.MaxErrorRate {
		pm.triggerAlert("error_rate_critical", "CRITICAL", "Error rate exceeds maximum threshold",
			pm.metrics.ErrorRate, thresholds.MaxErrorRate)
	} else if pm.metrics.ErrorRate > thresholds.WarningErrorRate {
		pm.triggerAlert("error_rate_warning", "WARNING", "Error rate exceeds warning threshold",
			pm.metrics.ErrorRate, thresholds.WarningErrorRate)
	}
}

// triggerAlert triggers a performance alert
func (pm *PerformanceMonitor) triggerAlert(alertType, severity, message string, value, threshold float64) {
	if pm.alerting == nil {
		return
	}

	alert := &Alert{
		ID:        fmt.Sprintf("%s_%d", alertType, time.Now().Unix()),
		Type:      alertType,
		Severity:  severity,
		Message:   message,
		Value:     value,
		Threshold: threshold,
		Timestamp: time.Now(),
		Tags:      make(map[string]string),
	}

	pm.alerting.TriggerAlert(alert)
}

// startMetricsServer starts the HTTP server for metrics exposition
func (pm *PerformanceMonitor) startMetricsServer() {
	mux := http.NewServeMux()

	// Prometheus metrics endpoint
	if pm.config.EnablePrometheus {
		mux.Handle(pm.config.MetricsPath, promhttp.Handler())
	}

	// Custom metrics endpoints
	mux.HandleFunc("/metrics/json", pm.handleJSONMetrics)
	mux.HandleFunc("/metrics/health", pm.handleHealthCheck)
	mux.HandleFunc("/metrics/alerts", pm.handleAlertsAPI)

	pm.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", pm.config.MetricsPort),
		Handler: mux,
	}

	pm.logger.Info("Metrics server starting",
		zap.String("addr", pm.server.Addr),
		zap.String("metrics_path", pm.config.MetricsPath),
	)

	if err := pm.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		pm.logger.Error("Metrics server error", zap.Error(err))
	}
}

// HTTP handlers

func (pm *PerformanceMonitor) handleJSONMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Return JSON representation of metrics
	jsonResponse := fmt.Sprintf(`{
		"latency": {
			"average": %f,
			"p95": %f,
			"p99": %f
		},
		"throughput": %f,
		"error_rate": %f,
		"resources": {
			"cpu_percent": %f,
			"memory_mb": %f,
			"goroutines": %d,
			"connections": %d
		}
	}`,
		pm.metrics.AverageLatency.Seconds(),
		pm.metrics.P95Latency.Seconds(),
		pm.metrics.P99Latency.Seconds(),
		pm.metrics.ThroughputPerSecond,
		pm.metrics.ErrorRate,
		pm.metrics.CPUUsagePercent,
		pm.metrics.MemoryUsageMB,
		pm.metrics.GoroutineCount,
		pm.metrics.ConnectionCount,
	)

	w.Write([]byte(jsonResponse))
}

func (pm *PerformanceMonitor) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	healthy := pm.isHealthy()
	status := "healthy"
	if !healthy {
		status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	response := fmt.Sprintf(`{"status": "%s", "timestamp": "%s"}`, status, time.Now().UTC().Format(time.RFC3339))
	w.Write([]byte(response))
}

func (pm *PerformanceMonitor) handleAlertsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if pm.alerting == nil {
		w.Write([]byte(`{"alerts": []}`))
		return
	}

	alerts := pm.alerting.GetActiveAlerts()
	alertsJSON := `{"alerts": [`

	for i, alert := range alerts {
		if i > 0 {
			alertsJSON += ","
		}
		alertsJSON += fmt.Sprintf(`{
			"id": "%s",
			"type": "%s",
			"severity": "%s",
			"message": "%s",
			"value": %f,
			"threshold": %f,
			"timestamp": "%s"
		}`,
			alert.ID, alert.Type, alert.Severity, alert.Message,
			alert.Value, alert.Threshold, alert.Timestamp.Format(time.RFC3339))
	}

	alertsJSON += `]}`
	w.Write([]byte(alertsJSON))
}

// isHealthy determines if the system is healthy based on current metrics
func (pm *PerformanceMonitor) isHealthy() bool {
	thresholds := pm.config.Thresholds

	// Check critical thresholds
	if pm.metrics.AverageLatency.Microseconds() > thresholds.MaxLatency {
		return false
	}

	if pm.metrics.ThroughputPerSecond < thresholds.MinThroughput {
		return false
	}

	if pm.metrics.ErrorRate > thresholds.MaxErrorRate {
		return false
	}

	if pm.metrics.CPUUsagePercent > thresholds.MaxCPUPercent {
		return false
	}

	if pm.metrics.MemoryUsageMB > float64(thresholds.MaxMemoryMB) {
		return false
	}

	return true
}

// Real-time monitoring methods

func (pm *PerformanceMonitor) realTimeUpdateLoop() {
	ticker := time.NewTicker(pm.config.RealTimeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.broadcastRealTimeUpdate()
		}
	}
}

func (pm *PerformanceMonitor) updateRealTimeData(metrics map[string]float64) {
	if pm.realTimeData == nil {
		return
	}

	pm.realTimeData.mutex.Lock()
	defer pm.realTimeData.mutex.Unlock()

	now := time.Now()

	// Update current values
	if latency, exists := metrics["gateway.average_latency"]; exists {
		pm.realTimeData.CurrentLatency = time.Duration(latency * float64(time.Microsecond))
		pm.realTimeData.LatencyHistory = append(pm.realTimeData.LatencyHistory,
			DataPoint{Timestamp: now, Value: latency})
	}

	if throughput, exists := metrics["gateway.throughput"]; exists {
		pm.realTimeData.CurrentThroughput = throughput
		pm.realTimeData.ThroughputHistory = append(pm.realTimeData.ThroughputHistory,
			DataPoint{Timestamp: now, Value: throughput})
	}

	if cpu, exists := metrics["runtime.cpu_percent"]; exists {
		pm.realTimeData.CurrentCPU = cpu
		pm.realTimeData.CPUHistory = append(pm.realTimeData.CPUHistory,
			DataPoint{Timestamp: now, Value: cpu})
	}

	if memory, exists := metrics["runtime.memory_mb"]; exists {
		pm.realTimeData.CurrentMemory = memory
		pm.realTimeData.MemoryHistory = append(pm.realTimeData.MemoryHistory,
			DataPoint{Timestamp: now, Value: memory})
	}

	// Trim history to keep it manageable
	maxPoints := 100
	if len(pm.realTimeData.LatencyHistory) > maxPoints {
		pm.realTimeData.LatencyHistory = pm.realTimeData.LatencyHistory[1:]
	}
	if len(pm.realTimeData.ThroughputHistory) > maxPoints {
		pm.realTimeData.ThroughputHistory = pm.realTimeData.ThroughputHistory[1:]
	}
	if len(pm.realTimeData.CPUHistory) > maxPoints {
		pm.realTimeData.CPUHistory = pm.realTimeData.CPUHistory[1:]
	}
	if len(pm.realTimeData.MemoryHistory) > maxPoints {
		pm.realTimeData.MemoryHistory = pm.realTimeData.MemoryHistory[1:]
	}
}

func (pm *PerformanceMonitor) broadcastRealTimeUpdate() {
	// This would broadcast real-time updates to WebSocket clients
	// Implementation depends on WebSocket library used
}

// Collector implementations

type RuntimeCollector struct{}

func NewRuntimeCollector() *RuntimeCollector {
	return &RuntimeCollector{}
}

func (rc *RuntimeCollector) Name() string {
	return "runtime"
}

func (rc *RuntimeCollector) Collect() (map[string]float64, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := make(map[string]float64)
	metrics["memory_mb"] = float64(m.HeapAlloc) / 1024 / 1024
	metrics["goroutines"] = float64(runtime.NumGoroutine())
	metrics["cpu_percent"] = 45.0 // Mock CPU usage

	return metrics, nil
}

type GatewayCollector struct {
	latency    int64
	throughput float64
	errorRate  float64
}

func NewGatewayCollector() *GatewayCollector {
	return &GatewayCollector{}
}

func (gc *GatewayCollector) Name() string {
	return "gateway"
}

func (gc *GatewayCollector) Collect() (map[string]float64, error) {
	metrics := make(map[string]float64)

	// Mock values - these would be collected from actual gateway metrics
	metrics["average_latency"] = float64(atomic.LoadInt64(&gc.latency))
	metrics["throughput"] = gc.throughput
	metrics["error_rate"] = gc.errorRate

	return metrics, nil
}

type NetworkCollector struct{}

func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{}
}

func (nc *NetworkCollector) Name() string {
	return "network"
}

func (nc *NetworkCollector) Collect() (map[string]float64, error) {
	metrics := make(map[string]float64)

	// Mock values - these would be collected from actual network monitoring
	metrics["latency_ms"] = 5.0
	metrics["packet_loss"] = 0.01
	metrics["bandwidth_utilization"] = 45.0

	return metrics, nil
}

// Helper functions

func NewAlertManager(config *MonitoringConfig, logger *zap.Logger) *AlertManager {
	return &AlertManager{
		logger:       logger,
		config:       config,
		alerts:       make(map[string]*Alert),
		alertChan:    make(chan *Alert, 100),
		alertHistory: make([]*Alert, 0),
	}
}

func (am *AlertManager) TriggerAlert(alert *Alert) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.alerts[alert.ID] = alert
	am.alertHistory = append(am.alertHistory, alert)

	select {
	case am.alertChan <- alert:
	default:
		// Channel full, drop alert
	}

	am.logger.Warn("Performance alert triggered",
		zap.String("alert_id", alert.ID),
		zap.String("type", alert.Type),
		zap.String("severity", alert.Severity),
		zap.String("message", alert.Message),
		zap.Float64("value", alert.Value),
		zap.Float64("threshold", alert.Threshold),
	)
}

func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	alerts := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		if !alert.Resolved {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

func NewRealTimeData() *RealTimeData {
	return &RealTimeData{
		clients:           make(map[*WebSocketClient]bool),
		updateChan:        make(chan MetricUpdate, 1000),
		LatencyHistory:    make([]DataPoint, 0),
		ThroughputHistory: make([]DataPoint, 0),
		CPUHistory:        make([]DataPoint, 0),
		MemoryHistory:     make([]DataPoint, 0),
	}
}

// GetMetrics returns current monitoring metrics
func (pm *PerformanceMonitor) GetMetrics() *MonitoringMetrics {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Return a copy to prevent concurrent access issues
	metricsCopy := *pm.metrics
	return &metricsCopy
}

// RecordLatency records a latency measurement
func (pm *PerformanceMonitor) RecordLatency(duration time.Duration, labels map[string]string) {
	if pm.config.EnablePrometheus && pm.promMetrics != nil {
		labelValues := make([]string, 0, len(labels))
		for _, value := range labels {
			labelValues = append(labelValues, value)
		}
		pm.promMetrics.RequestDuration.WithLabelValues(labelValues...).Observe(duration.Seconds())
	}
}

// RecordRequest records a request
func (pm *PerformanceMonitor) RecordRequest(labels map[string]string) {
	if pm.config.EnablePrometheus && pm.promMetrics != nil {
		labelValues := make([]string, 0, len(labels))
		for _, value := range labels {
			labelValues = append(labelValues, value)
		}
		pm.promMetrics.RequestsTotal.WithLabelValues(labelValues...).Inc()
	}
}

// RecordError records an error
func (pm *PerformanceMonitor) RecordError(errorType string, labels map[string]string) {
	if pm.config.EnablePrometheus && pm.promMetrics != nil {
		labelValues := make([]string, 0, len(labels)+1)
		labelValues = append(labelValues, errorType)
		for _, value := range labels {
			labelValues = append(labelValues, value)
		}
		pm.promMetrics.ErrorsTotal.WithLabelValues(labelValues...).Inc()
	}
}

// Close gracefully shuts down the performance monitor
func (pm *PerformanceMonitor) Close() error {
	pm.cancel()

	if pm.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		pm.server.Shutdown(ctx)
	}

	return nil
}
