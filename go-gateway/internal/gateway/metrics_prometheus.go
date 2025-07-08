//go:build prometheus
// +build prometheus

package gateway

import (
	"time"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

// initMetrics initializes Prometheus metrics (when prometheus build tag is used)
func (g *IndustrialGateway) initMetrics() {
	// Initialize Prometheus metrics as before
	g.metrics.connectionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bifrost_connections_total",
		Help: "Total number of connections to the gateway",
	})

	g.metrics.dataPointsProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bifrost_data_points_processed_total",
		Help: "Total number of data points processed",
	})

	g.metrics.errorRate = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bifrost_errors_total",
		Help: "Total number of errors encountered",
	})

	g.metrics.responseTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "bifrost_response_time_seconds",
		Help:    "Response time histogram",
		Buckets: prometheus.DefBuckets,
	})

	// Register metrics
	prometheus.MustRegister(
		g.metrics.connectionsTotal.(prometheus.Counter),
		g.metrics.dataPointsProcessed.(prometheus.Counter),
		g.metrics.errorRate.(prometheus.Counter),
		g.metrics.responseTime.(prometheus.Histogram),
	)
}

// Helper methods for Prometheus
func (g *IndustrialGateway) recordConnection() {
	if counter, ok := g.metrics.connectionsTotal.(prometheus.Counter); ok {
		counter.Inc()
	}
}

func (g *IndustrialGateway) recordDataPoint() {
	if counter, ok := g.metrics.dataPointsProcessed.(prometheus.Counter); ok {
		counter.Inc()
	}
}

func (g *IndustrialGateway) recordError() {
	if counter, ok := g.metrics.errorRate.(prometheus.Counter); ok {
		counter.Inc()
	}
}

func (g *IndustrialGateway) recordResponseTime(duration time.Duration) {
	if histogram, ok := g.metrics.responseTime.(prometheus.Histogram); ok {
		histogram.Observe(duration.Seconds())
	}
}

// Override metrics endpoint setup for Prometheus
func (g *IndustrialGateway) setupMetricsEndpoint(mux *http.ServeMux) {
	if g.config.EnableMetrics {
		mux.Handle("/metrics", promhttp.Handler())
	}
}