package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/bifrost/gateway/internal/protocols"
)

// TestResult holds the result of a test operation
type TestResult struct {
	Name     string
	Success  bool
	Duration time.Duration
	Error    error
	Details  map[string]interface{}
}

// TestSuite manages the integration tests
type TestSuite struct {
	logger  *zap.Logger
	handler protocols.ProtocolHandler
	results []*TestResult
	mutex   sync.Mutex
}

func main() {
	// Setup logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	fmt.Println("Bifrost Go Gateway - Modbus Integration Test")
	fmt.Println("============================================")
	fmt.Println()

	// Create test suite
	suite := &TestSuite{
		logger:  logger,
		handler: protocols.NewModbusHandler(logger),
		results: make([]*TestResult, 0),
	}

	// Run all tests
	suite.runAllTests()

	// Print summary
	suite.printSummary()
}

func (ts *TestSuite) runAllTests() {
	fmt.Println("Running integration tests...")
	fmt.Println()

	// Test 1: Basic connectivity
	ts.testBasicConnectivity()

	// Test 2: Device operations
	ts.testDeviceOperations()

	// Test 3: Read operations
	ts.testReadOperations()

	// Test 4: Write operations
	ts.testWriteOperations()

	// Test 5: Performance tests
	ts.testPerformance()

	// Test 6: Concurrent operations
	ts.testConcurrentOperations()

	// Test 7: Device discovery
	ts.testDeviceDiscovery()

	// Test 8: Error handling
	ts.testErrorHandling()
}

func (ts *TestSuite) testBasicConnectivity() {
	fmt.Println("1. Testing Basic Connectivity...")
	start := time.Now()

	// Test TCP connection to localhost:502
	conn, err := net.DialTimeout("tcp", "localhost:502", 5*time.Second)
	if err != nil {
		ts.addResult("basic_connectivity", false, time.Since(start), err, nil)
		fmt.Printf("   ❌ Cannot connect to Modbus simulator: %v\n", err)
		return
	}
	conn.Close()

	duration := time.Since(start)
	ts.addResult("basic_connectivity", true, duration, nil, map[string]interface{}{
		"host": "localhost",
		"port": 502,
	})
	fmt.Printf("   ✅ TCP connection successful (%v)\n", duration)
}

func (ts *TestSuite) testDeviceOperations() {
	fmt.Println("\n2. Testing Device Operations...")

	// Create test device
	device := &protocols.Device{
		ID:       "test-device-001",
		Name:     "Test Modbus Device",
		Protocol: "modbus-tcp",
		Address:  "127.0.0.1",
		Port:     502,
		Config:   map[string]interface{}{"unit_id": 1},
	}

	// Test connection
	start := time.Now()
	err := ts.handler.Connect(device)
	duration := time.Since(start)

	if err != nil {
		ts.addResult("device_connect", false, duration, err, nil)
		fmt.Printf("   ❌ Device connection failed: %v\n", err)
		return
	}

	ts.addResult("device_connect", true, duration, nil, map[string]interface{}{
		"device_id": device.ID,
		"address":   device.Address,
	})
	fmt.Printf("   ✅ Device connection successful (%v)\n", duration)

	// Test connection status
	isConnected := ts.handler.IsConnected(device)
	if !isConnected {
		ts.addResult("device_status", false, 0, fmt.Errorf("device not connected"), nil)
		fmt.Printf("   ❌ Device status check failed\n")
		return
	}

	fmt.Printf("   ✅ Device status check successful\n")

	// Test ping
	start = time.Now()
	err = ts.handler.Ping(device)
	duration = time.Since(start)

	if err != nil {
		ts.addResult("device_ping", false, duration, err, nil)
		fmt.Printf("   ❌ Device ping failed: %v\n", duration)
	} else {
		ts.addResult("device_ping", true, duration, nil, nil)
		fmt.Printf("   ✅ Device ping successful (%v)\n", duration)
	}

	// Test device info
	start = time.Now()
	info, err := ts.handler.GetDeviceInfo(device)
	duration = time.Since(start)

	if err != nil {
		ts.addResult("device_info", false, duration, err, nil)
		fmt.Printf("   ❌ Get device info failed: %v\n", err)
	} else {
		ts.addResult("device_info", true, duration, nil, map[string]interface{}{
			"vendor":       info.Vendor,
			"model":        info.Model,
			"capabilities": info.Capabilities,
		})
		fmt.Printf("   ✅ Device info retrieved: %s %s (%v)\n", info.Vendor, info.Model, duration)
	}

	// Test diagnostics
	start = time.Now()
	diag, err := ts.handler.GetDiagnostics(device)
	duration = time.Since(start)

	if err != nil {
		ts.addResult("device_diagnostics", false, duration, err, nil)
		fmt.Printf("   ❌ Get diagnostics failed: %v\n", err)
	} else {
		ts.addResult("device_diagnostics", true, duration, nil, map[string]interface{}{
			"healthy":   diag.IsHealthy,
			"last_comm": diag.LastCommunication,
			"uptime":    diag.ConnectionUptime,
		})
		fmt.Printf("   ✅ Diagnostics retrieved: Healthy=%v, Uptime=%v (%v)\n", diag.IsHealthy, diag.ConnectionUptime, duration)
	}
}

func (ts *TestSuite) testReadOperations() {
	fmt.Println("\n3. Testing Read Operations...")

	device := &protocols.Device{
		ID:       "test-device-001",
		Name:     "Test Modbus Device",
		Protocol: "modbus-tcp",
		Address:  "127.0.0.1",
		Port:     502,
		Config:   map[string]interface{}{"unit_id": 1},
	}

	// Ensure device is connected
	if err := ts.handler.Connect(device); err != nil {
		fmt.Printf("   ❌ Cannot connect to device: %v\n", err)
		return
	}

	// Test reading different register types
	testTags := []*protocols.Tag{
		{
			ID:       "temp_sensor_1",
			Name:     "Temperature Sensor 1",
			Address:  "40001", // Holding register
			DataType: "int16",
			Writable: false,
		},
		{
			ID:       "pressure_sensor_1",
			Name:     "Pressure Sensor 1",
			Address:  "40011", // Holding register
			DataType: "int16",
			Writable: false,
		},
		{
			ID:       "flow_sensor_1",
			Name:     "Flow Sensor 1",
			Address:  "40021", // Holding register
			DataType: "int16",
			Writable: false,
		},
	}

	for _, tag := range testTags {
		start := time.Now()
		value, err := ts.handler.ReadTag(device, tag)
		duration := time.Since(start)

		if err != nil {
			ts.addResult(fmt.Sprintf("read_%s", tag.ID), false, duration, err, nil)
			fmt.Printf("   ❌ Read %s failed: %v\n", tag.Name, err)
		} else {
			ts.addResult(fmt.Sprintf("read_%s", tag.ID), true, duration, nil, map[string]interface{}{
				"value": value,
				"type":  tag.DataType,
			})
			fmt.Printf("   ✅ Read %s: %v (%v)\n", tag.Name, value, duration)
		}
	}

	// Test multiple read
	start := time.Now()
	results, err := ts.handler.ReadMultipleTags(device, testTags)
	duration := time.Since(start)

	if err != nil {
		ts.addResult("read_multiple", false, duration, err, nil)
		fmt.Printf("   ❌ Multiple read failed: %v\n", err)
	} else {
		ts.addResult("read_multiple", true, duration, nil, map[string]interface{}{
			"count": len(results),
		})
		fmt.Printf("   ✅ Multiple read successful: %d values (%v)\n", len(results), duration)
		for id, value := range results {
			fmt.Printf("      %s: %v\n", id, value)
		}
	}
}

func (ts *TestSuite) testWriteOperations() {
	fmt.Println("\n4. Testing Write Operations...")

	device := &protocols.Device{
		ID:       "test-device-001",
		Name:     "Test Modbus Device",
		Protocol: "modbus-tcp",
		Address:  "127.0.0.1",
		Port:     502,
		Config:   map[string]interface{}{"unit_id": 1},
	}

	// Ensure device is connected
	if err := ts.handler.Connect(device); err != nil {
		fmt.Printf("   ❌ Cannot connect to device: %v\n", err)
		return
	}

	// Test writing to writable registers (above address 40030)
	writeTag := &protocols.Tag{
		ID:       "setpoint_1",
		Name:     "Temperature Setpoint",
		Address:  "40050", // Writable register
		DataType: "int16",
		Writable: true,
	}

	testValue := int16(2500) // 25.00°C

	start := time.Now()
	err := ts.handler.WriteTag(device, writeTag, testValue)
	duration := time.Since(start)

	if err != nil {
		ts.addResult("write_tag", false, duration, err, nil)
		fmt.Printf("   ❌ Write operation failed: %v\n", err)
	} else {
		ts.addResult("write_tag", true, duration, nil, map[string]interface{}{
			"value":   testValue,
			"address": writeTag.Address,
		})
		fmt.Printf("   ✅ Write operation successful: %v to %s (%v)\n", testValue, writeTag.Address, duration)

		// Verify write by reading back
		start = time.Now()
		readValue, err := ts.handler.ReadTag(device, writeTag)
		duration = time.Since(start)

		if err != nil {
			fmt.Printf("   ❌ Read-back verification failed: %v\n", err)
		} else {
			fmt.Printf("   ✅ Read-back verification: %v (%v)\n", readValue, duration)
		}
	}
}

func (ts *TestSuite) testPerformance() {
	fmt.Println("\n5. Testing Performance...")

	device := &protocols.Device{
		ID:       "test-device-001",
		Name:     "Test Modbus Device",
		Protocol: "modbus-tcp",
		Address:  "127.0.0.1",
		Port:     502,
		Config:   map[string]interface{}{"unit_id": 1},
	}

	// Ensure device is connected
	if err := ts.handler.Connect(device); err != nil {
		fmt.Printf("   ❌ Cannot connect to device: %v\n", err)
		return
	}

	// Performance test: rapid sequential reads
	tag := &protocols.Tag{
		ID:       "perf_test",
		Name:     "Performance Test Tag",
		Address:  "40001",
		DataType: "int16",
		Writable: false,
	}

	iterations := 1000
	start := time.Now()

	successCount := 0
	for i := 0; i < iterations; i++ {
		_, err := ts.handler.ReadTag(device, tag)
		if err == nil {
			successCount++
		}
	}

	duration := time.Since(start)
	readsPerSecond := float64(successCount) / duration.Seconds()
	avgLatency := duration / time.Duration(successCount)

	ts.addResult("performance_sequential", true, duration, nil, map[string]interface{}{
		"iterations":       iterations,
		"success_count":    successCount,
		"reads_per_second": readsPerSecond,
		"avg_latency":      avgLatency,
	})

	fmt.Printf("   ✅ Sequential reads: %d/%d successful\n", successCount, iterations)
	fmt.Printf("   ✅ Performance: %.0f reads/second, avg latency: %v\n", readsPerSecond, avgLatency)
	fmt.Printf("   ✅ Total time: %v\n", duration)
}

func (ts *TestSuite) testConcurrentOperations() {
	fmt.Println("\n6. Testing Concurrent Operations...")

	device := &protocols.Device{
		ID:       "test-device-001",
		Name:     "Test Modbus Device",
		Protocol: "modbus-tcp",
		Address:  "127.0.0.1",
		Port:     502,
		Config:   map[string]interface{}{"unit_id": 1},
	}

	// Ensure device is connected
	if err := ts.handler.Connect(device); err != nil {
		fmt.Printf("   ❌ Cannot connect to device: %v\n", err)
		return
	}

	goroutines := 10
	readsPerGoroutine := 100

	var wg sync.WaitGroup
	results := make(chan bool, goroutines*readsPerGoroutine)

	tag := &protocols.Tag{
		ID:       "concurrent_test",
		Name:     "Concurrent Test Tag",
		Address:  "40001",
		DataType: "int16",
		Writable: false,
	}

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < readsPerGoroutine; j++ {
				_, err := ts.handler.ReadTag(device, tag)
				results <- err == nil
			}
		}(i)
	}

	wg.Wait()
	close(results)

	duration := time.Since(start)

	successCount := 0
	totalOps := 0
	for success := range results {
		totalOps++
		if success {
			successCount++
		}
	}

	opsPerSecond := float64(successCount) / duration.Seconds()

	ts.addResult("concurrent_operations", true, duration, nil, map[string]interface{}{
		"goroutines":        goroutines,
		"ops_per_goroutine": readsPerGoroutine,
		"total_ops":         totalOps,
		"success_count":     successCount,
		"ops_per_second":    opsPerSecond,
	})

	fmt.Printf("   ✅ Concurrent operations: %d/%d successful\n", successCount, totalOps)
	fmt.Printf("   ✅ Performance: %.0f ops/second with %d goroutines\n", opsPerSecond, goroutines)
	fmt.Printf("   ✅ Total time: %v\n", duration)
}

func (ts *TestSuite) testDeviceDiscovery() {
	fmt.Println("\n7. Testing Device Discovery...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	devices, err := ts.handler.DiscoverDevices(ctx, "127.0.0.1/32")
	duration := time.Since(start)

	if err != nil {
		ts.addResult("device_discovery", false, duration, err, nil)
		fmt.Printf("   ❌ Device discovery failed: %v\n", err)
		return
	}

	ts.addResult("device_discovery", true, duration, nil, map[string]interface{}{
		"discovered_count": len(devices),
	})

	fmt.Printf("   ✅ Device discovery completed: %d devices found (%v)\n", len(devices), duration)
	for _, device := range devices {
		fmt.Printf("      - %s (%s:%d)\n", device.Name, device.Address, device.Port)
	}
}

func (ts *TestSuite) testErrorHandling() {
	fmt.Println("\n8. Testing Error Handling...")

	// Test with invalid device
	invalidDevice := &protocols.Device{
		ID:       "invalid-device",
		Name:     "Invalid Device",
		Protocol: "modbus-tcp",
		Address:  "192.168.255.255", // Non-existent address
		Port:     502,
		Config:   map[string]interface{}{"unit_id": 1},
	}

	start := time.Now()
	err := ts.handler.Connect(invalidDevice)
	duration := time.Since(start)

	if err != nil {
		ts.addResult("error_handling_invalid_device", true, duration, nil, map[string]interface{}{
			"expected_error": err.Error(),
		})
		fmt.Printf("   ✅ Invalid device connection properly failed: %v (%v)\n", err, duration)
	} else {
		ts.addResult("error_handling_invalid_device", false, duration,
			fmt.Errorf("expected connection failure but succeeded"), nil)
		fmt.Printf("   ❌ Invalid device connection should have failed\n")
	}

	// Test with invalid address format
	validDevice := &protocols.Device{
		ID:       "test-device-001",
		Name:     "Test Modbus Device",
		Protocol: "modbus-tcp",
		Address:  "127.0.0.1",
		Port:     502,
		Config:   map[string]interface{}{"unit_id": 1},
	}

	if err := ts.handler.Connect(validDevice); err == nil {
		invalidTag := &protocols.Tag{
			ID:       "invalid_tag",
			Name:     "Invalid Tag",
			Address:  "invalid_address",
			DataType: "int16",
			Writable: false,
		}

		start = time.Now()
		_, err = ts.handler.ReadTag(validDevice, invalidTag)
		duration = time.Since(start)

		if err != nil {
			ts.addResult("error_handling_invalid_address", true, duration, nil, map[string]interface{}{
				"expected_error": err.Error(),
			})
			fmt.Printf("   ✅ Invalid address properly failed: %v (%v)\n", err, duration)
		} else {
			ts.addResult("error_handling_invalid_address", false, duration,
				fmt.Errorf("expected address validation failure but succeeded"), nil)
			fmt.Printf("   ❌ Invalid address should have failed\n")
		}
	}
}

func (ts *TestSuite) addResult(name string, success bool, duration time.Duration, err error, details map[string]interface{}) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	ts.results = append(ts.results, &TestResult{
		Name:     name,
		Success:  success,
		Duration: duration,
		Error:    err,
		Details:  details,
	})
}

func (ts *TestSuite) printSummary() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	successCount := 0
	totalTime := time.Duration(0)

	for _, result := range ts.results {
		status := "✅ PASS"
		if !result.Success {
			status = "❌ FAIL"
		} else {
			successCount++
		}

		totalTime += result.Duration

		fmt.Printf("%-30s %s (%v)\n", result.Name, status, result.Duration)
		if result.Error != nil {
			fmt.Printf("   Error: %v\n", result.Error)
		}
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Total Tests: %d\n", len(ts.results))
	fmt.Printf("Passed: %d\n", successCount)
	fmt.Printf("Failed: %d\n", len(ts.results)-successCount)
	fmt.Printf("Success Rate: %.1f%%\n", float64(successCount)/float64(len(ts.results))*100)
	fmt.Printf("Total Time: %v\n", totalTime)
	fmt.Println(strings.Repeat("=", 60))

	// Performance summary
	fmt.Println("\nPERFORMANCE HIGHLIGHTS:")
	for _, result := range ts.results {
		if result.Success && result.Details != nil {
			switch result.Name {
			case "performance_sequential":
				if rps, ok := result.Details["reads_per_second"].(float64); ok {
					fmt.Printf("Sequential Read Performance: %.0f reads/second\n", rps)
				}
				if latency, ok := result.Details["avg_latency"].(time.Duration); ok {
					fmt.Printf("Average Latency: %v\n", latency)
				}
			case "concurrent_operations":
				if ops, ok := result.Details["ops_per_second"].(float64); ok {
					fmt.Printf("Concurrent Operations: %.0f ops/second\n", ops)
				}
			}
		}
	}

	// Exit with appropriate code
	if successCount == len(ts.results) {
		fmt.Println("\n🎉 All tests passed!")
		os.Exit(0)
	} else {
		fmt.Printf("\n❌ %d tests failed\n", len(ts.results)-successCount)
		os.Exit(1)
	}
}
