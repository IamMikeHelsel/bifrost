package main

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"bifrost-gateway/test-framework"
)

// TestComprehensiveFuzzingSuite runs the complete fuzzing test suite
func TestComprehensiveFuzzingSuite(t *testing.T) {
	logger := zap.NewNop()
	
	t.Log("Starting comprehensive fuzzing test suite")
	testframework.RunFuzzTestSuite(t, logger)
	t.Log("Fuzzing test suite completed")
}

// TestProtocolSimulators tests the protocol simulators
func TestProtocolSimulators(t *testing.T) {
	logger := zap.NewNop()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("ModbusSimulator", func(t *testing.T) {
		simulator := testframework.NewModbusSimulator(logger)
		
		// Start simulator
		err := simulator.Start(ctx, 0) // Use any available port
		if err != nil {
			t.Fatalf("Failed to start Modbus simulator: %v", err)
		}
		defer simulator.Stop()

		// Add a test device
		tags := map[string]interface{}{
			"temperature": uint16(250),
			"pressure":    uint16(1013),
			"running":     true,
		}
		err = simulator.SimulateDevice("device1", tags)
		if err != nil {
			t.Fatalf("Failed to add device: %v", err)
		}

		// Test fault injection
		err = simulator.InjectFault("timeout", 5*time.Second)
		if err != nil {
			t.Fatalf("Failed to inject fault: %v", err)
		}

		// Get metrics
		metrics := simulator.GetMetrics()
		if metrics.FaultsInjected != 1 {
			t.Errorf("Expected 1 fault injected, got %d", metrics.FaultsInjected)
		}

		t.Log("Modbus simulator test passed")
	})
}

// TestPerformanceBenchmarks runs performance benchmark tests
func TestPerformanceBenchmarks(t *testing.T) {
	logger := zap.NewNop()
	suite := testframework.NewPerformanceSuite(logger)

	// Add standard performance tests
	for _, test := range testframework.CreateStandardPerformanceTests() {
		suite.AddTest(test)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	t.Log("Starting performance benchmark tests")
	results, err := suite.RunAll(ctx)
	if err != nil {
		t.Fatalf("Performance test suite failed: %v", err)
	}

	// Validate results
	for _, result := range results {
		t.Logf("Performance test %s (concurrency %d): %.2f ops/sec, %.2f%% errors, P99: %v",
			result.TestName, result.ConcurrencyLevel, result.OperationsPerSec, 
			result.ErrorRate, result.P99Latency)

		// Basic performance validation
		if result.OperationsPerSec <= 0 {
			t.Errorf("Test %s reported zero operations per second", result.TestName)
		}

		if result.ErrorRate > 10.0 {
			t.Errorf("Test %s has high error rate: %.2f%%", result.TestName, result.ErrorRate)
		}
	}

	t.Log("Performance benchmark tests completed")
}

// TestIntegrationSuite runs comprehensive integration tests
func TestIntegrationSuite(t *testing.T) {
	logger := zap.NewNop()
	
	t.Log("Starting integration test suite")
	testframework.RunIntegrationTestSuite(t, logger)
	t.Log("Integration test suite completed")
}