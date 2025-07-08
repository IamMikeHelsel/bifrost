package examples

import (
	"fmt"
	"log"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/bifrost/go-gateway/internal/protocols"
)

// PerformanceTest demonstrates the high-performance capabilities of the Go gateway
func performanceDemo() {
	logger, _ := zap.NewDevelopment()

	fmt.Println("Bifrost Go Gateway Performance Test")
	fmt.Println("===================================")

	// Test Modbus address parsing performance
	testModbusAddressParsing(logger)

	// Test data conversion performance
	testDataConversion(logger)

	// Test concurrent operations
	testConcurrentOperations(logger)

	fmt.Println("\nPerformance test completed!")
}

func testModbusAddressParsing(logger *zap.Logger) {
	fmt.Println("\n1. Testing Modbus Address Validation Performance...")

	handler := protocols.NewModbusHandler(logger)
	addresses := []string{
		"40001", "40100", "40500", "41000", "41500",
		"30001", "30100", "30500", "31000", "31500",
		"00001", "00100", "00500", "01000", "01500",
		"10001", "10100", "10500", "11000", "11500",
	}

	iterations := 100000 // Reduced for validation
	start := time.Now()

	for i := 0; i < iterations; i++ {
		for _, addr := range addresses {
			err := handler.ValidateTagAddress(addr)
			if err != nil {
				log.Fatalf("Failed to validate address %s: %v", addr, err)
			}
		}
	}

	duration := time.Since(start)
	totalOps := iterations * len(addresses)
	opsPerSecond := float64(totalOps) / duration.Seconds()

	fmt.Printf("   Validated %d addresses in %v\n", totalOps, duration)
	fmt.Printf("   Performance: %.0f validations/second\n", opsPerSecond)
	fmt.Printf("   Average time per validation: %v\n", duration/time.Duration(totalOps))
}

func testDataConversion(logger *zap.Logger) {
	fmt.Println("\n2. Testing Supported Data Types...")

	handler := protocols.NewModbusHandler(logger)

	// Get supported data types
	dataTypes := handler.GetSupportedDataTypes()

	fmt.Printf("   Supported data types: %v\n", dataTypes)
	fmt.Printf("   Total supported types: %d\n", len(dataTypes))

	// Test multiple calls to demonstrate performance
	iterations := 100000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		handler.GetSupportedDataTypes()
	}

	duration := time.Since(start)
	opsPerSecond := float64(iterations) / duration.Seconds()

	fmt.Printf("   GetSupportedDataTypes(): %.0f calls/second\n", opsPerSecond)
}

func testConcurrentOperations(logger *zap.Logger) {
	fmt.Println("\n3. Testing Concurrent Operations...")

	handler := protocols.NewModbusHandler(logger)

	// Create mock devices
	devices := make([]*protocols.Device, 100)
	for i := 0; i < 100; i++ {
		devices[i] = &protocols.Device{
			ID:       fmt.Sprintf("device-%d", i),
			Name:     fmt.Sprintf("Test Device %d", i),
			Protocol: "modbus-tcp",
			Address:  fmt.Sprintf("192.168.1.%d", i+1),
			Port:     502,
			Config:   make(map[string]interface{}),
		}
	}

	// Test concurrent device info retrieval
	start := time.Now()
	var wg sync.WaitGroup

	for _, device := range devices {
		wg.Add(1)
		go func(d *protocols.Device) {
			defer wg.Done()

			// Get device info
			_, err := handler.GetDeviceInfo(d)
			if err != nil {
				log.Printf("Failed to get device info for %s: %v", d.ID, err)
			}

			// Validate addresses
			testAddresses := []string{"40001", "40100", "30001", "10001"}
			for _, addr := range testAddresses {
				if err := handler.ValidateTagAddress(addr); err != nil {
					log.Printf("Address validation failed for %s: %v", addr, err)
				}
			}
		}(device)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("   Processed %d devices concurrently in %v\n", len(devices), duration)
	fmt.Printf("   Average time per device: %v\n", duration/time.Duration(len(devices)))
}

// Performance test demonstrates the high-speed capabilities of the Go gateway
