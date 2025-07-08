package metrics

import (
	"encoding/json"
	"expvar"
	"net/http"
	"sync/atomic"
	"time"
)

// GatewayMetrics provides simple metrics collection without external dependencies
type GatewayMetrics struct {
	ConnectionsTotal    int64 `json:"connections_total"`
	DataPointsProcessed int64 `json:"data_points_processed"`
	ErrorCount          int64 `json:"error_count"`
	ResponseTimeSum     int64 `json:"-"`
	ResponseTimeCount   int64 `json:"-"`
	
	// Additional industrial gateway metrics
	DevicesConnected    int64 `json:"devices_connected"`
	TagsRead           int64 `json:"tags_read"`
	TagsWritten        int64 `json:"tags_written"`
	ModbusRequests     int64 `json:"modbus_requests"`
	OPCUARequests      int64 `json:"opcua_requests"`
	EtherNetIPRequests int64 `json:"ethernetip_requests"`
}

// NewGatewayMetrics creates a new metrics instance
func NewGatewayMetrics() *GatewayMetrics {
	return &GatewayMetrics{}
}

// Connection metrics
func (m *GatewayMetrics) IncrementConnections() {
	atomic.AddInt64(&m.ConnectionsTotal, 1)
}

func (m *GatewayMetrics) SetDevicesConnected(count int64) {
	atomic.StoreInt64(&m.DevicesConnected, count)
}

// Data processing metrics
func (m *GatewayMetrics) RecordDataPoint() {
	atomic.AddInt64(&m.DataPointsProcessed, 1)
}

func (m *GatewayMetrics) RecordTagRead() {
	atomic.AddInt64(&m.TagsRead, 1)
}

func (m *GatewayMetrics) RecordTagWrite() {
	atomic.AddInt64(&m.TagsWritten, 1)
}

// Protocol-specific metrics
func (m *GatewayMetrics) RecordModbusRequest() {
	atomic.AddInt64(&m.ModbusRequests, 1)
}

func (m *GatewayMetrics) RecordOPCUARequest() {
	atomic.AddInt64(&m.OPCUARequests, 1)
}

func (m *GatewayMetrics) RecordEtherNetIPRequest() {
	atomic.AddInt64(&m.EtherNetIPRequests, 1)
}

// Error tracking
func (m *GatewayMetrics) RecordError() {
	atomic.AddInt64(&m.ErrorCount, 1)
}

// Response time tracking
func (m *GatewayMetrics) RecordResponseTime(duration time.Duration) {
	atomic.AddInt64(&m.ResponseTimeSum, duration.Nanoseconds())
	atomic.AddInt64(&m.ResponseTimeCount, 1)
}

func (m *GatewayMetrics) GetAverageResponseTime() time.Duration {
	sum := atomic.LoadInt64(&m.ResponseTimeSum)
	count := atomic.LoadInt64(&m.ResponseTimeCount)
	if count == 0 {
		return 0
	}
	return time.Duration(sum / count)
}

// GetSnapshot returns a snapshot of current metrics
func (m *GatewayMetrics) GetSnapshot() map[string]interface{} {
	return map[string]interface{}{
		"connections_total":     atomic.LoadInt64(&m.ConnectionsTotal),
		"data_points_processed": atomic.LoadInt64(&m.DataPointsProcessed),
		"error_count":           atomic.LoadInt64(&m.ErrorCount),
		"devices_connected":     atomic.LoadInt64(&m.DevicesConnected),
		"tags_read":            atomic.LoadInt64(&m.TagsRead),
		"tags_written":         atomic.LoadInt64(&m.TagsWritten),
		"modbus_requests":      atomic.LoadInt64(&m.ModbusRequests),
		"opcua_requests":       atomic.LoadInt64(&m.OPCUARequests),
		"ethernetip_requests":  atomic.LoadInt64(&m.EtherNetIPRequests),
		"avg_response_time_ms": float64(m.GetAverageResponseTime().Nanoseconds()) / 1e6,
		"timestamp":           time.Now().Unix(),
	}
}

// ServeHTTP implements http.Handler for metrics endpoint
func (m *GatewayMetrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	snapshot := m.GetSnapshot()
	if err := json.NewEncoder(w).Encode(snapshot); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return
	}
}

// Register with expvar for automatic export at /debug/vars
func (m *GatewayMetrics) Register() {
	expvar.Publish("gateway", expvar.Func(func() interface{} {
		return m.GetSnapshot()
	}))
}

// Reset clears all metrics (useful for testing)
func (m *GatewayMetrics) Reset() {
	atomic.StoreInt64(&m.ConnectionsTotal, 0)
	atomic.StoreInt64(&m.DataPointsProcessed, 0)
	atomic.StoreInt64(&m.ErrorCount, 0)
	atomic.StoreInt64(&m.ResponseTimeSum, 0)
	atomic.StoreInt64(&m.ResponseTimeCount, 0)
	atomic.StoreInt64(&m.DevicesConnected, 0)
	atomic.StoreInt64(&m.TagsRead, 0)
	atomic.StoreInt64(&m.TagsWritten, 0)
	atomic.StoreInt64(&m.ModbusRequests, 0)
	atomic.StoreInt64(&m.OPCUARequests, 0)
	atomic.StoreInt64(&m.EtherNetIPRequests, 0)
}