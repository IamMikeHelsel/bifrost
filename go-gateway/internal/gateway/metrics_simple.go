//go:build !prometheus
// +build !prometheus

package gateway

import (
	"time"
	
	"bifrost-gateway/internal/metrics"
)

// initMetrics initializes simple custom metrics (default build)
func (g *IndustrialGateway) initMetrics() {
	g.customMetrics = metrics.NewGatewayMetrics()
	g.customMetrics.Register()
}

// Metrics helper methods for custom metrics
func (g *IndustrialGateway) recordConnection() {
	if g.customMetrics != nil {
		g.customMetrics.IncrementConnections()
	}
}

func (g *IndustrialGateway) recordDataPoint() {
	if g.customMetrics != nil {
		g.customMetrics.RecordDataPoint()
	}
}

func (g *IndustrialGateway) recordError() {
	if g.customMetrics != nil {
		g.customMetrics.RecordError()
	}
}

func (g *IndustrialGateway) recordResponseTime(duration time.Duration) {
	if g.customMetrics != nil {
		g.customMetrics.RecordResponseTime(duration)
	}
}