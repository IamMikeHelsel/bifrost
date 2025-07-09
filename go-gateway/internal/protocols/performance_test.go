package protocols

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// BenchmarkModbusOperations benchmarks various Modbus operations
func BenchmarkModbusOperations(b *testing.B) {
	logger := zap.NewNop()
	handler := NewModbusHandler(logger).(*ModbusHandler)

	b.Run("ParseAddress", func(b *testing.B) {
		addresses := []string{"40001", "40100", "30001", "00001", "10001"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addr := addresses[i%len(addresses)]
			_, err := handler.parseAddress(addr)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("DataConversion", func(b *testing.B) {
		testData := []byte{0x01, 0x23, 0x45, 0x67}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Convert to Modbus
			_, err := handler.convertFromModbus(testData[:2], "uint16", ReadHoldingRegisters)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ConcurrentReads", func(b *testing.B) {
		// Simulate concurrent tag reads
		tags := make([]*Tag, 100)
		for i := 0; i < 100; i++ {
			tags[i] = &Tag{
				ID:       fmt.Sprintf("tag-%d", i),
				Address:  fmt.Sprintf("4%04d", i+1),
				DataType: "uint16",
			}
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Simulate tag validation
				for _, tag := range tags {
					handler.ValidateTagAddress(tag.Address)
				}
			}
		})
	})
}

// BenchmarkEtherNetIPOperations benchmarks EtherNet/IP operations
func BenchmarkEtherNetIPOperations(b *testing.B) {
	logger := zap.NewNop()
	handler := NewEtherNetIPHandler(logger).(*EtherNetIPHandler)

	b.Run("SymbolicPathBuilding", func(b *testing.B) {
		tagNames := []string{
			"SimpleTag",
			"Array[10]",
			"Structure.Member",
			"Complex.Path.To.Tag",
			"VeryLongTagNameForTestingPerformance",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tagName := tagNames[i%len(tagNames)]
			path := handler.buildSymbolicPath(tagName)
			if len(path) == 0 {
				b.Fatal("Empty path")
			}
		}
	})

	b.Run("CIPDataConversion", func(b *testing.B) {
		values := []interface{}{
			int32(12345),
			float32(123.45),
			true,
			"Test String",
		}
		dataTypes := []uint8{
			CIPDataTypeDint,
			CIPDataTypeReal,
			CIPDataTypeBool,
			CIPDataTypeString,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			idx := i % len(values)
			data, err := handler.convertToCIP(values[idx], dataTypes[idx])
			if err != nil {
				b.Fatal(err)
			}
			_, err = handler.convertFromCIP(data, dataTypes[idx])
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestHighConcurrency tests the system under high concurrency
func TestHighConcurrency(t *testing.T) {
	logger := zap.NewNop()

	t.Run("Concurrent device operations", func(t *testing.T) {
		manager := NewDeviceManager(logger)
		numDevices := 100
		numOperationsPerDevice := 50
		
		// Create devices
		devices := make([]*Device, numDevices)
		for i := 0; i < numDevices; i++ {
			devices[i] = &Device{
				ID:       fmt.Sprintf("device-%d", i),
				Protocol: "modbus-tcp",
				Address:  fmt.Sprintf("192.168.1.%d", i+1),
				Port:     502,
			}
			err := manager.RegisterDevice(devices[i])
			require.NoError(t, err)
		}

		// Concurrent operations
		var wg sync.WaitGroup
		errors := make(chan error, numDevices*numOperationsPerDevice)
		successCount := int64(0)

		start := time.Now()

		for _, device := range devices {
			wg.Add(1)
			go func(d *Device) {
				defer wg.Done()
				
				for j := 0; j < numOperationsPerDevice; j++ {
					// Simulate read operation
					tag := &Tag{
						ID:       fmt.Sprintf("tag-%d", j),
						Address:  fmt.Sprintf("4%04d", j+1),
						DataType: "uint16",
					}
					
					// In real scenario, this would read from device
					// Here we just validate the tag
					handler := manager.GetHandler(d.Protocol)
					if handler != nil {
						err := handler.ValidateTagAddress(tag.Address)
						if err != nil {
							errors <- err
						} else {
							atomic.AddInt64(&successCount, 1)
						}
					}
				}
			}(device)
		}

		wg.Wait()
		close(errors)

		elapsed := time.Since(start)
		totalOps := int64(numDevices * numOperationsPerDevice)
		opsPerSecond := float64(totalOps) / elapsed.Seconds()

		// Check results
		errorCount := len(errors)
		assert.Equal(t, totalOps, successCount+int64(errorCount))
		assert.Equal(t, 0, errorCount, "Should have no errors")

		t.Logf("Completed %d operations in %v (%.0f ops/sec)", totalOps, elapsed, opsPerSecond)
		assert.Greater(t, opsPerSecond, float64(10000), "Should achieve > 10k ops/sec")
	})
}

// TestMemoryEfficiency tests memory usage patterns
func TestMemoryEfficiency(t *testing.T) {
	t.Run("Tag memory usage", func(t *testing.T) {
		// Force GC to get baseline
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Create many tags
		numTags := 10000
		tags := make([]*Tag, numTags)
		for i := 0; i < numTags; i++ {
			tags[i] = &Tag{
				ID:          fmt.Sprintf("tag-%d", i),
				Name:        fmt.Sprintf("Tag %d", i),
				Address:     fmt.Sprintf("4%04d", i+1),
				DataType:    "uint16",
				Description: "Test tag for memory efficiency testing",
				Writable:    true,
			}
		}

		// Force GC and measure
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		bytesPerTag := (m2.HeapAlloc - m1.HeapAlloc) / uint64(numTags)
		t.Logf("Memory per tag: %d bytes", bytesPerTag)
		
		// Should be reasonably efficient
		assert.Less(t, bytesPerTag, uint64(1024), "Tag should use less than 1KB")
	})

	t.Run("Connection pool memory", func(t *testing.T) {
		// Test that connection pools don't leak memory
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Create and destroy many connections
		for i := 0; i < 1000; i++ {
			pool := NewConnectionPool(10, func() (interface{}, error) {
				return &MockConnection{id: i}, nil
			})
			
			// Get and release connections
			conns := make([]interface{}, 5)
			for j := 0; j < 5; j++ {
				conn, _ := pool.Get(context.Background())
				conns[j] = conn
			}
			for _, conn := range conns {
				pool.Put(conn)
			}
			
			pool.Close()
		}

		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		// Memory should not grow significantly
		memoryGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
		t.Logf("Memory growth after 1000 pool cycles: %d bytes", memoryGrowth)
		
		// Allow some growth but it should be minimal
		assert.Less(t, memoryGrowth, int64(10*1024*1024), "Memory growth should be < 10MB")
	})
}

// BenchmarkProtocolHandlers benchmarks different protocol handlers
func BenchmarkProtocolHandlers(b *testing.B) {
	logger := zap.NewNop()

	b.Run("Handler creation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handler := NewModbusHandler(logger)
			if handler == nil {
				b.Fatal("Failed to create handler")
			}
		}
	})

	b.Run("Tag validation", func(b *testing.B) {
		modbusHandler := NewModbusHandler(logger)
		eipHandler := NewEtherNetIPHandler(logger)

		addresses := []struct {
			handler ProtocolHandler
			address string
		}{
			{modbusHandler, "40001"},
			{modbusHandler, "30100"},
			{eipHandler, "MyTag"},
			{eipHandler, "Array[10]"},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addr := addresses[i%len(addresses)]
			err := addr.handler.ValidateTagAddress(addr.address)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestScalability tests system scalability
func TestScalability(t *testing.T) {
	logger := zap.NewNop()

	t.Run("Device scalability", func(t *testing.T) {
		manager := NewDeviceManager(logger)
		
		// Test with increasing number of devices
		deviceCounts := []int{10, 100, 1000}
		
		for _, count := range deviceCounts {
			// Reset manager
			manager = NewDeviceManager(logger)
			
			start := time.Now()
			
			// Register devices
			for i := 0; i < count; i++ {
				device := &Device{
					ID:       fmt.Sprintf("device-%d", i),
					Protocol: "modbus-tcp",
					Address:  fmt.Sprintf("10.0.%d.%d", i/256, i%256),
					Port:     502,
				}
				err := manager.RegisterDevice(device)
				require.NoError(t, err)
			}
			
			registerTime := time.Since(start)
			
			// Lookup all devices
			start = time.Now()
			for i := 0; i < count; i++ {
				device := manager.GetDevice(fmt.Sprintf("device-%d", i))
				assert.NotNil(t, device)
			}
			lookupTime := time.Since(start)
			
			avgRegisterTime := registerTime / time.Duration(count)
			avgLookupTime := lookupTime / time.Duration(count)
			
			t.Logf("Devices: %d, Avg Register: %v, Avg Lookup: %v", 
				count, avgRegisterTime, avgLookupTime)
			
			// Performance should scale well
			assert.Less(t, avgRegisterTime, 100*time.Microsecond)
			assert.Less(t, avgLookupTime, 10*time.Microsecond)
		}
	})

	t.Run("Concurrent read scalability", func(t *testing.T) {
		// Test with different numbers of concurrent readers
		concurrencyLevels := []int{10, 50, 100, 200}
		
		for _, level := range concurrencyLevels {
			handler := NewModbusHandler(logger)
			
			var wg sync.WaitGroup
			start := time.Now()
			successCount := int64(0)
			
			for i := 0; i < level; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					
					// Each goroutine validates 1000 addresses
					for j := 0; j < 1000; j++ {
						addr := fmt.Sprintf("4%04d", (j%9999)+1)
						err := handler.ValidateTagAddress(addr)
						if err == nil {
							atomic.AddInt64(&successCount, 1)
						}
					}
				}(i)
			}
			
			wg.Wait()
			elapsed := time.Since(start)
			
			totalOps := int64(level * 1000)
			opsPerSecond := float64(totalOps) / elapsed.Seconds()
			
			t.Logf("Concurrency: %d, Total Ops: %d, Time: %v, Ops/sec: %.0f",
				level, totalOps, elapsed, opsPerSecond)
			
			assert.Equal(t, totalOps, successCount)
			// Performance should remain good even with high concurrency
			assert.Greater(t, opsPerSecond, float64(100000))
		}
	})
}

// Benchmark helper structures
type DeviceManager struct {
	devices  map[string]*Device
	handlers map[string]ProtocolHandler
	mu       sync.RWMutex
	logger   *zap.Logger
}

func NewDeviceManager(logger *zap.Logger) *DeviceManager {
	return &DeviceManager{
		devices:  make(map[string]*Device),
		handlers: make(map[string]ProtocolHandler),
		logger:   logger,
	}
}

func (dm *DeviceManager) RegisterDevice(device *Device) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	if _, exists := dm.devices[device.ID]; exists {
		return fmt.Errorf("device %s already registered", device.ID)
	}
	
	dm.devices[device.ID] = device
	
	// Register protocol handler if needed
	if _, exists := dm.handlers[device.Protocol]; !exists {
		switch device.Protocol {
		case "modbus-tcp", "modbus-rtu":
			dm.handlers[device.Protocol] = NewModbusHandler(dm.logger)
		case "ethernet-ip":
			dm.handlers[device.Protocol] = NewEtherNetIPHandler(dm.logger)
		}
	}
	
	return nil
}

func (dm *DeviceManager) GetDevice(id string) *Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.devices[id]
}

func (dm *DeviceManager) GetHandler(protocol string) ProtocolHandler {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.handlers[protocol]
}

// BenchmarkDataStructures benchmarks various data structures
func BenchmarkDataStructures(b *testing.B) {
	b.Run("Map vs Slice lookup", func(b *testing.B) {
		// Create test data
		numItems := 1000
		
		// Map-based lookup
		deviceMap := make(map[string]*Device, numItems)
		for i := 0; i < numItems; i++ {
			id := fmt.Sprintf("device-%d", i)
			deviceMap[id] = &Device{ID: id}
		}
		
		// Slice-based lookup
		deviceSlice := make([]*Device, numItems)
		for i := 0; i < numItems; i++ {
			deviceSlice[i] = &Device{ID: fmt.Sprintf("device-%d", i)}
		}
		
		b.Run("Map", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				id := fmt.Sprintf("device-%d", i%numItems)
				_ = deviceMap[id]
			}
		})
		
		b.Run("Slice", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				targetID := fmt.Sprintf("device-%d", i%numItems)
				for _, d := range deviceSlice {
					if d.ID == targetID {
						break
					}
				}
			}
		})
	})

	b.Run("Channel vs Mutex", func(b *testing.B) {
		// Channel-based counter
		type command struct {
			op    string
			value int
			resp  chan int
		}
		
		cmdChan := make(chan command, 100)
		go func() {
			counter := 0
			for cmd := range cmdChan {
				switch cmd.op {
				case "inc":
					counter += cmd.value
				case "get":
					cmd.resp <- counter
				}
			}
		}()
		
		// Mutex-based counter
		var mutexCounter int64
		var mu sync.Mutex
		
		b.Run("Channel", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					cmd := command{op: "inc", value: 1}
					cmdChan <- cmd
				}
			})
		})
		
		b.Run("Mutex", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					mu.Lock()
					mutexCounter++
					mu.Unlock()
				}
			})
		})
		
		b.Run("Atomic", func(b *testing.B) {
			var atomicCounter int64
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					atomic.AddInt64(&atomicCounter, 1)
				}
			})
		})
		
		close(cmdChan)
	})
}