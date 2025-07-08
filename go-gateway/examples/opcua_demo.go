package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"bifrost-gateway/internal/protocols"
	"go.uber.org/zap"
)

// demonstrateOPCUAClient shows how to use the OPC UA client implementation
func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create OPC UA handler
	handler := protocols.NewOPCUAHandler(logger)

	// Define a test device (requires a running OPC UA server)
	device := &protocols.Device{
		ID:       "opcua-demo-device",
		Name:     "OPC UA Demo Server",
		Protocol: "opcua",
		Address:  "localhost",
		Port:     4840,
		Config: map[string]interface{}{
			"security_policy": "None",
			"security_mode":   "None",
			"auth_policy":     "Anonymous",
		},
	}

	fmt.Println("=== OPC UA Client Demo ===")

	// 1. Test device discovery
	fmt.Println("\n1. Device Discovery")
	ctx := context.Background()
	devices, err := handler.DiscoverDevices(ctx, "localhost")
	if err != nil {
		log.Printf("Discovery failed: %v", err)
	} else {
		fmt.Printf("Discovered %d devices\n", len(devices))
		for _, dev := range devices {
			fmt.Printf("  - Device: %s (%s)\n", dev.Name, dev.Address)
		}
	}

	// 2. Test address validation
	fmt.Println("\n2. Address Validation")
	testAddresses := []string{
		"ns=1;i=1001",
		"ns=2;s=Temperature",
		"ns=3;g=550e8400-e29b-41d4-a716-446655440000",
		"",
	}

	for _, addr := range testAddresses {
		err := handler.ValidateTagAddress(addr)
		status := "✓ Valid"
		if err != nil {
			status = "✗ Invalid: " + err.Error()
		}
		fmt.Printf("  %s -> %s\n", addr, status)
	}

	// 3. Show supported data types
	fmt.Println("\n3. Supported Data Types")
	dataTypes := handler.GetSupportedDataTypes()
	fmt.Printf("  Supports %d data types: %v\n", len(dataTypes), dataTypes)

	// 4. Test connection (will only work if OPC UA server is running)
	fmt.Println("\n4. Connection Test")
	err = handler.Connect(device)
	if err != nil {
		fmt.Printf("  Connection failed: %v\n", err)
		fmt.Println("  Note: Start the virtual OPC UA server for full testing:")
		fmt.Println("    cd virtual-devices/opcua-sim && python opcua_server.py")
		return
	}

	defer handler.Disconnect(device)
	fmt.Printf("  ✓ Connected to %s\n", device.Address)

	// 5. Test ping
	fmt.Println("\n5. Connectivity Test")
	err = handler.Ping(device)
	if err != nil {
		fmt.Printf("  ✗ Ping failed: %v\n", err)
	} else {
		fmt.Println("  ✓ Device is reachable")
	}

	// 6. Get device information
	fmt.Println("\n6. Device Information")
	info, err := handler.GetDeviceInfo(device)
	if err != nil {
		fmt.Printf("  Failed to get device info: %v\n", err)
	} else {
		fmt.Printf("  Vendor: %s\n", info.Vendor)
		fmt.Printf("  Model: %s\n", info.Model)
		fmt.Printf("  Capabilities: %v\n", info.Capabilities)
	}

	// 7. Get diagnostics
	fmt.Println("\n7. Diagnostics")
	diag, err := handler.GetDiagnostics(device)
	if err != nil {
		fmt.Printf("  Failed to get diagnostics: %v\n", err)
	} else {
		fmt.Printf("  Healthy: %v\n", diag.IsHealthy)
		fmt.Printf("  Response Time: %v\n", diag.ResponseTime)
		fmt.Printf("  Last Communication: %v\n", diag.LastCommunication.Format(time.RFC3339))
	}

	// 8. Test browsing (if connected)
	fmt.Println("\n8. Node Browsing")
	browseResults, err := handler.BrowseNodes(device, "", 2) // Browse from root, max depth 2
	if err != nil {
		fmt.Printf("  Browse failed: %v\n", err)
	} else {
		fmt.Printf("  Found %d nodes\n", len(browseResults))
		for i, node := range browseResults {
			if i >= 5 { // Limit output
				fmt.Printf("  ... and %d more nodes\n", len(browseResults)-i)
				break
			}
			fmt.Printf("  - %s (%s) [%s]\n", node.DisplayName, node.NodeID, node.NodeClass)
		}
	}

	// 9. Performance demonstration
	fmt.Println("\n9. Performance Test")
	
	// Test bulk address validation
	addresses := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		addresses[i] = fmt.Sprintf("ns=1;i=%d", i+1000)
	}

	start := time.Now()
	validCount := 0
	for _, addr := range addresses {
		if handler.ValidateTagAddress(addr) == nil {
			validCount++
		}
	}
	duration := time.Since(start)

	fmt.Printf("  Validated %d/%d addresses in %v\n", validCount, len(addresses), duration)
	fmt.Printf("  Performance: %.0f addresses/second\n", float64(len(addresses))/duration.Seconds())

	// Performance target check
	if duration < time.Second {
		fmt.Println("  ✓ Performance target met (< 1 second for 1000 operations)")
	} else {
		fmt.Println("  ✗ Performance target not met")
	}

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("For full functionality, ensure the virtual OPC UA server is running:")
	fmt.Println("  cd virtual-devices/opcua-sim && python opcua_server.py")
}