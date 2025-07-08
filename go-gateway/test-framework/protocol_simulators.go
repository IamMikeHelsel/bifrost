package testframework

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ProtocolSimulator defines the interface for protocol simulators
type ProtocolSimulator interface {
	Start(ctx context.Context, port int) error
	Stop() error
	GetAddr() net.Addr
	SimulateDevice(deviceID string, tags map[string]interface{}) error
	InjectFault(faultType string, duration time.Duration) error
	GetMetrics() *SimulatorMetrics
}

// SimulatorMetrics tracks simulator performance
type SimulatorMetrics struct {
	RequestsHandled   int64
	ResponsesSent     int64
	ErrorsGenerated   int64
	AverageLatency    time.Duration
	ConnectionCount   int
	FaultsInjected    int
	UptimeSeconds     int64
}

// ModbusSimulator implements a Modbus TCP simulator
type ModbusSimulator struct {
	listener net.Listener
	devices  map[string]*SimulatedDevice
	metrics  *SimulatorMetrics
	logger   *zap.Logger
	mu       sync.RWMutex
	running  bool
	startTime time.Time
}

// SimulatedDevice represents a simulated Modbus device
type SimulatedDevice struct {
	ID             string
	CoilValues     map[uint16]bool
	DiscreteInputs map[uint16]bool
	HoldingRegs    map[uint16]uint16
	InputRegs      map[uint16]uint16
	FaultActive    bool
	FaultType      string
	FaultEndTime   time.Time
}

// NewModbusSimulator creates a new Modbus simulator
func NewModbusSimulator(logger *zap.Logger) *ModbusSimulator {
	return &ModbusSimulator{
		devices: make(map[string]*SimulatedDevice),
		metrics: &SimulatorMetrics{},
		logger:  logger,
	}
}

// Start starts the Modbus simulator on the specified port
func (s *ModbusSimulator) Start(ctx context.Context, port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("simulator already running")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.listener = listener
	s.running = true
	s.startTime = time.Now()

	// Start accepting connections
	go s.acceptConnections(ctx)

	s.logger.Info("Modbus simulator started", zap.Int("port", port))
	return nil
}

// Stop stops the simulator
func (s *ModbusSimulator) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}

	s.logger.Info("Modbus simulator stopped")
	return nil
}

// GetAddr returns the simulator's listening address
func (s *ModbusSimulator) GetAddr() net.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener != nil {
		return s.listener.Addr()
	}
	return nil
}

// SimulateDevice adds a device to the simulator
func (s *ModbusSimulator) SimulateDevice(deviceID string, tags map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	device := &SimulatedDevice{
		ID:             deviceID,
		CoilValues:     make(map[uint16]bool),
		DiscreteInputs: make(map[uint16]bool),
		HoldingRegs:    make(map[uint16]uint16),
		InputRegs:      make(map[uint16]uint16),
	}

	// Initialize device registers from tags
	for _, value := range tags {
		// Simple mapping for demo - in real implementation would parse address formats
		switch v := value.(type) {
		case bool:
			device.CoilValues[0] = v
		case int:
			device.HoldingRegs[0] = uint16(v)
		case uint16:
			device.HoldingRegs[0] = v
		}
	}

	s.devices[deviceID] = device
	s.logger.Info("Device added to simulator", zap.String("device_id", deviceID))
	return nil
}

// InjectFault injects a fault into the simulator
func (s *ModbusSimulator) InjectFault(faultType string, duration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Apply fault to all devices
	for _, device := range s.devices {
		device.FaultActive = true
		device.FaultType = faultType
		device.FaultEndTime = time.Now().Add(duration)
	}

	s.metrics.FaultsInjected++
	s.logger.Info("Fault injected", zap.String("type", faultType), zap.Duration("duration", duration))
	return nil
}

// GetMetrics returns simulator metrics
func (s *ModbusSimulator) GetMetrics() *SimulatorMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Update uptime
	if s.running {
		s.metrics.UptimeSeconds = int64(time.Since(s.startTime).Seconds())
	}

	return s.metrics
}

// acceptConnections handles incoming connections
func (s *ModbusSimulator) acceptConnections(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if s.running {
					s.logger.Error("Failed to accept connection", zap.Error(err))
				}
				continue
			}

			go s.handleConnection(conn)
		}
	}
}

// handleConnection handles a single client connection
func (s *ModbusSimulator) handleConnection(conn net.Conn) {
	defer conn.Close()

	s.mu.Lock()
	s.metrics.ConnectionCount++
	s.mu.Unlock()

	buffer := make([]byte, 1024)
	for {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(buffer)
		if err != nil {
			return
		}

		// Process Modbus request and send response
		response := s.processModbusRequest(buffer[:n])
		if response != nil {
			conn.Write(response)
			
			s.mu.Lock()
			s.metrics.ResponsesSent++
			s.mu.Unlock()
		}
	}
}

// processModbusRequest processes a Modbus request and returns a response
func (s *ModbusSimulator) processModbusRequest(request []byte) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.RequestsHandled++

	// Check for active faults
	for _, device := range s.devices {
		if device.FaultActive && time.Now().Before(device.FaultEndTime) {
			switch device.FaultType {
			case "timeout":
				time.Sleep(5 * time.Second) // Simulate timeout
			case "error":
				s.metrics.ErrorsGenerated++
				return s.createErrorResponse(request)
			case "disconnect":
				return nil // Simulate connection drop
			}
		}
	}

	// Basic Modbus TCP response simulation
	// In a real implementation, this would parse the full Modbus protocol
	if len(request) < 8 {
		return s.createErrorResponse(request)
	}

	// Extract transaction ID and function code
	transactionID := request[0:2]
	functionCode := request[7]

	// Create a simple response based on function code
	response := make([]byte, 9)
	copy(response[0:2], transactionID) // Transaction ID
	response[2] = 0x00 // Protocol ID
	response[3] = 0x00
	response[4] = 0x00 // Length
	response[5] = 0x03
	response[6] = 0x01 // Unit ID
	response[7] = functionCode
	response[8] = 0x00 // Data byte

	return response
}

// createErrorResponse creates a Modbus error response
func (s *ModbusSimulator) createErrorResponse(request []byte) []byte {
	if len(request) < 8 {
		return nil
	}

	response := make([]byte, 9)
	copy(response[0:2], request[0:2]) // Transaction ID
	response[2] = 0x00 // Protocol ID
	response[3] = 0x00
	response[4] = 0x00 // Length
	response[5] = 0x03
	response[6] = request[6] // Unit ID
	response[7] = request[7] | 0x80 // Error function code
	response[8] = 0x02 // Exception code (illegal data address)

	return response
}