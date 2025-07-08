package cloud

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// HealthChecker monitors the health of cloud connectors
type HealthChecker struct {
	logger     *zap.Logger
	connectors map[string]CloudConnector
	interval   time.Duration
	mutex      sync.RWMutex
	
	// Control
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(logger *zap.Logger, interval time.Duration) *HealthChecker {
	hc := &HealthChecker{
		logger:     logger,
		connectors: make(map[string]CloudConnector),
		interval:   interval,
		stopCh:     make(chan struct{}),
	}
	
	// Start health checking
	hc.start()
	
	return hc
}

// AddConnector adds a connector to health monitoring
func (hc *HealthChecker) AddConnector(name string, connector CloudConnector) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	
	hc.connectors[name] = connector
	hc.logger.Debug("Added connector to health monitoring", zap.String("name", name))
}

// RemoveConnector removes a connector from health monitoring
func (hc *HealthChecker) RemoveConnector(name string) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	
	delete(hc.connectors, name)
	hc.logger.Debug("Removed connector from health monitoring", zap.String("name", name))
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
	hc.logger.Info("Health checker stopped")
}

// start starts the health checking routine
func (hc *HealthChecker) start() {
	if hc.interval <= 0 {
		return
	}
	
	hc.wg.Add(1)
	go func() {
		defer hc.wg.Done()
		
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-hc.stopCh:
				return
			case <-ticker.C:
				hc.checkHealth()
			}
		}
	}()
	
	hc.logger.Info("Health checker started", zap.Duration("interval", hc.interval))
}

// checkHealth performs health checks on all connectors
func (hc *HealthChecker) checkHealth() {
	hc.mutex.RLock()
	connectors := make(map[string]CloudConnector)
	for name, connector := range hc.connectors {
		connectors[name] = connector
	}
	hc.mutex.RUnlock()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	var wg sync.WaitGroup
	for name, connector := range connectors {
		wg.Add(1)
		go func(connectorName string, conn CloudConnector) {
			defer wg.Done()
			hc.checkConnectorHealth(ctx, connectorName, conn)
		}(name, connector)
	}
	
	wg.Wait()
}

// checkConnectorHealth checks the health of a single connector
func (hc *HealthChecker) checkConnectorHealth(ctx context.Context, name string, connector CloudConnector) {
	startTime := time.Now()
	
	// Ping the connector
	err := connector.Ping(ctx)
	duration := time.Since(startTime)
	
	health := connector.GetHealth()
	
	if err != nil {
		hc.logger.Warn("Connector health check failed", 
			zap.String("connector", name),
			zap.Error(err),
			zap.Duration("duration", duration),
			zap.Bool("connected", connector.IsConnected()))
		
		// Try to reconnect if not connected
		if !connector.IsConnected() {
			hc.logger.Info("Attempting to reconnect connector", zap.String("connector", name))
			if reconnectErr := connector.Connect(ctx); reconnectErr != nil {
				hc.logger.Error("Failed to reconnect connector", 
					zap.String("connector", name), zap.Error(reconnectErr))
			} else {
				hc.logger.Info("Successfully reconnected connector", zap.String("connector", name))
			}
		}
	} else {
		hc.logger.Debug("Connector health check passed", 
			zap.String("connector", name),
			zap.Duration("duration", duration),
			zap.Duration("responseTime", health.ResponseTime),
			zap.Float64("successRate", health.SuccessRate))
	}
}