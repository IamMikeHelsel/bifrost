package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"

	"bifrost-gateway/internal/hardware"
	"bifrost-gateway/internal/protocols"
)

func main() {
	var (
		configPath    = flag.String("config", "configs/hardware_test_lab.yaml", "Path to hardware test lab configuration")
		scenariosPath = flag.String("scenarios", "configs/test_scenarios.yaml", "Path to test scenarios configuration")
		command       = flag.String("cmd", "run", "Command to execute: run, list-devices, list-scenarios, test, schedule")
		deviceID      = flag.String("device", "", "Device ID for test command")
		scenario      = flag.String("scenario", "", "Scenario name for test command")
		daemon        = flag.Bool("daemon", false, "Run as daemon with scheduler")
		verbose       = flag.Bool("v", false, "Verbose logging")
	)
	flag.Parse()

	// Setup logger
	var logger *zap.Logger
	var err error
	if *verbose {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	// Resolve absolute paths
	configAbsPath, err := filepath.Abs(*configPath)
	if err != nil {
		logger.Fatal("Failed to resolve config path", zap.Error(err))
	}
	
	scenariosAbsPath, err := filepath.Abs(*scenariosPath)
	if err != nil {
		logger.Fatal("Failed to resolve scenarios path", zap.Error(err))
	}

	// Create hardware test manager
	manager := hardware.NewHardwareTestManager(logger, configAbsPath)

	// Initialize manager
	if err := manager.Initialize(); err != nil {
		logger.Fatal("Failed to initialize hardware test manager", zap.Error(err))
	}

	// Load test scenarios
	if err := manager.LoadTestScenarios(scenariosAbsPath); err != nil {
		logger.Fatal("Failed to load test scenarios", zap.Error(err))
	}

	// Register protocol handlers
	manager.RegisterProtocolHandler("modbus_tcp", protocols.NewModbusHandler(logger))
	manager.RegisterProtocolHandler("ethernet_ip", protocols.NewEtherNetIPHandler(logger))
	manager.RegisterProtocolHandler("opcua", protocols.NewOPCUAHandler(logger))

	// Execute command
	switch *command {
	case "run":
		runDaemon(manager, logger, *daemon)
	case "list-devices":
		listDevices(manager)
	case "list-scenarios":
		listScenarios(manager)
	case "test":
		runSingleTest(manager, *deviceID, *scenario, logger)
	case "schedule":
		showSchedule(manager)
	case "status":
		showStatus(manager)
	default:
		fmt.Printf("Unknown command: %s\n", *command)
		flag.Usage()
		os.Exit(1)
	}
}

func runDaemon(manager *hardware.HardwareTestManager, logger *zap.Logger, isDaemon bool) {
	fmt.Println("🌉 Bifrost Hardware Testing Framework")
	fmt.Println("====================================")
	fmt.Println()

	if isDaemon {
		logger.Info("Starting hardware test manager in daemon mode")
		
		// Start the manager with scheduler
		if err := manager.Start(); err != nil {
			logger.Fatal("Failed to start hardware test manager", zap.Error(err))
		}

		// Setup signal handling
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		fmt.Println("✅ Hardware test manager started in daemon mode")
		fmt.Println("📊 Scheduler is running and will execute tests based on device configuration")
		fmt.Println("🛑 Press Ctrl+C to stop")
		fmt.Println()

		// Wait for shutdown signal
		<-sigCh
		logger.Info("Received shutdown signal")

		// Stop the manager
		if err := manager.Stop(); err != nil {
			logger.Error("Error stopping hardware test manager", zap.Error(err))
		}

		fmt.Println("🛑 Hardware test manager stopped")
	} else {
		// Just show status and exit
		showStatus(manager)
	}
}

func listDevices(manager *hardware.HardwareTestManager) {
	fmt.Println("📋 Registered Hardware Devices")
	fmt.Println("==============================")
	fmt.Println()

	devices := manager.GetRegistry().GetAllDevices()
	if len(devices) == 0 {
		fmt.Println("No devices registered")
		return
	}

	for _, device := range devices {
		fmt.Printf("🖥️  Device: %s\n", device.DeviceID)
		fmt.Printf("   Manufacturer: %s\n", device.Manufacturer)
		fmt.Printf("   Model: %s\n", device.Model)
		fmt.Printf("   Firmware: %s\n", device.Firmware)
		fmt.Printf("   Address: %s:%d\n", device.Network.IP, device.Network.Port)
		fmt.Printf("   Protocols: %v\n", device.Protocols)
		fmt.Printf("   Status: %s\n", device.Status)
		fmt.Printf("   Schedule: %s (%v scenarios)\n", device.TestSchedule.Frequency, len(device.TestSchedule.Scenarios))
		fmt.Printf("   Enabled: %t\n", device.TestSchedule.Enabled)
		if !device.LastTested.IsZero() {
			fmt.Printf("   Last Tested: %s\n", device.LastTested.Format(time.RFC3339))
		}
		fmt.Println()
	}
}

func listScenarios(manager *hardware.HardwareTestManager) {
	fmt.Println("📝 Available Test Scenarios")
	fmt.Println("===========================")
	fmt.Println()

	// Note: This would require adding a method to access scenarios from the scheduler
	// For now, we'll show a placeholder
	fmt.Println("Test scenarios are loaded from configuration file.")
	fmt.Println("Use 'cat configs/test_scenarios.yaml' to view available scenarios.")
}

func runSingleTest(manager *hardware.HardwareTestManager, deviceID, scenarioName string, logger *zap.Logger) {
	if deviceID == "" {
		fmt.Println("❌ Device ID is required for test command")
		os.Exit(1)
	}
	if scenarioName == "" {
		fmt.Println("❌ Scenario name is required for test command")
		os.Exit(1)
	}

	fmt.Printf("🧪 Running Test: %s on %s\n", scenarioName, deviceID)
	fmt.Println("================================")
	fmt.Println()

	// Check if device exists
	device, err := manager.GetRegistry().GetDevice(deviceID)
	if err != nil {
		fmt.Printf("❌ Device not found: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("📱 Device: %s %s (%s)\n", device.Manufacturer, device.Model, device.Network.IP)
	fmt.Printf("🔧 Test Scenario: %s\n", scenarioName)
	fmt.Println()

	// This would require exposing scenario lookup from scheduler
	// For now, create a basic scenario
	scenario := hardware.TestScenario{
		Name:        scenarioName,
		Description: fmt.Sprintf("On-demand test: %s", scenarioName),
		Protocol:    device.Protocols[0], // Use first protocol
		Timeout:     2 * time.Minute,
		Steps: []hardware.TestStep{
			{Name: "Connect", Type: "connect", Timeout: 30 * time.Second},
			{Name: "Ping", Type: "ping", Timeout: 10 * time.Second},
			{Name: "Device Info", Type: "device_info", Timeout: 15 * time.Second},
			{Name: "Disconnect", Type: "disconnect", Timeout: 10 * time.Second},
		},
	}

	// Execute the test
	ctx, cancel := context.WithTimeout(context.Background(), scenario.Timeout)
	defer cancel()

	execution, err := manager.GetExecutor().ExecuteTest(ctx, deviceID, scenario)
	if err != nil {
		fmt.Printf("❌ Failed to start test: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("🚀 Test execution started (ID: %s)\n", execution.ID)
	fmt.Println("⏳ Waiting for test to complete...")
	fmt.Println()

	// Poll for completion
	for {
		time.Sleep(500 * time.Millisecond)
		
		execution, err := manager.GetExecutor().GetExecution(execution.ID)
		if err != nil {
			fmt.Printf("❌ Error checking execution status: %s\n", err)
			break
		}

		if execution.Status != "running" {
			// Test completed
			if execution.Status == "completed" {
				fmt.Println("✅ Test completed successfully!")
			} else {
				fmt.Printf("❌ Test failed with status: %s\n", execution.Status)
				if execution.Error != "" {
					fmt.Printf("   Error: %s\n", execution.Error)
				}
			}

			// Show step results
			fmt.Println("\n📊 Test Step Results:")
			fmt.Println("=====================")
			for i, step := range execution.Steps {
				status := "✅"
				if !step.Success {
					status = "❌"
				}
				fmt.Printf("%s Step %d: %s (%.2fs)\n", status, i+1, step.Step.Name, step.Duration.Seconds())
				if step.Error != "" {
					fmt.Printf("   Error: %s\n", step.Error)
				}
			}

			break
		}

		// Show progress
		fmt.Printf("⏳ Step %d/%d: %s\n", execution.CurrentStep+1, len(execution.Scenario.Steps), 
			execution.Scenario.Steps[execution.CurrentStep].Name)
	}
}

func showSchedule(manager *hardware.HardwareTestManager) {
	fmt.Println("📅 Test Schedule")
	fmt.Println("================")
	fmt.Println()

	tasks := manager.GetScheduler().GetScheduledTasks()
	if len(tasks) == 0 {
		fmt.Println("No scheduled tasks")
		return
	}

	for _, task := range tasks {
		status := "🔴 Disabled"
		if task.Enabled {
			status = "🟢 Enabled"
		}

		fmt.Printf("📋 Task: %s\n", task.ID)
		fmt.Printf("   Device: %s\n", task.DeviceID)
		fmt.Printf("   Scenario: %s\n", task.Scenario.Name)
		fmt.Printf("   Schedule: %s\n", task.Schedule)
		fmt.Printf("   Status: %s\n", status)
		fmt.Printf("   Next Run: %s\n", task.NextRun.Format(time.RFC3339))
		if task.LastRun != nil {
			fmt.Printf("   Last Run: %s\n", task.LastRun.Format(time.RFC3339))
		}
		fmt.Printf("   Priority: %d\n", task.Priority)
		fmt.Println()
	}
}

func showStatus(manager *hardware.HardwareTestManager) {
	fmt.Println("📊 Hardware Test Manager Status")
	fmt.Println("===============================")
	fmt.Println()

	status := manager.GetStatus()
	
	// Convert to pretty JSON for display
	jsonBytes, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting status: %s\n", err)
		return
	}

	fmt.Println(string(jsonBytes))
	fmt.Println()

	// Show active executions
	executions := manager.GetExecutor().GetActiveExecutions()
	if len(executions) > 0 {
		fmt.Println("🔄 Active Test Executions:")
		for _, exec := range executions {
			fmt.Printf("   %s: %s on %s (Step %d/%d)\n", 
				exec.ID, exec.Scenario.Name, exec.DeviceID, 
				exec.CurrentStep+1, len(exec.Scenario.Steps))
		}
		fmt.Println()
	}
}