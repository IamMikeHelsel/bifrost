package protocols

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goburrow/modbus"
	"go.uber.org/zap"
)

// ModbusHandler implements the ProtocolHandler interface for Modbus TCP/RTU
type ModbusHandler struct {
	logger      *zap.Logger
	connections sync.Map // map[string]*ModbusConnection
	config      *ModbusConfig
}

// ModbusConnection represents a Modbus client connection
type ModbusConnection struct {
	client      modbus.Client
	handler     *modbus.TCPClientHandler
	lastUsed    time.Time
	deviceID    string
	isConnected bool
	mutex       sync.RWMutex

	// Connection pooling
	inUse     bool
	createdAt time.Time
}

// ModbusConfig holds Modbus-specific configuration
type ModbusConfig struct {
	DefaultTimeout    time.Duration `yaml:"default_timeout"`
	DefaultUnitID     byte          `yaml:"default_unit_id"`
	MaxConnections    int           `yaml:"max_connections"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	EnableKeepAlive   bool          `yaml:"enable_keep_alive"`
}

// ModbusAddress represents parsed Modbus address information
type ModbusAddress struct {
	FunctionCode ModbusFunctionCode
	Address      uint16
	Count        uint16
	UnitID       byte
}

// ModbusFunctionCode represents Modbus function codes
type ModbusFunctionCode int

const (
	// Read functions
	ReadCoils            ModbusFunctionCode = 1
	ReadDiscreteInputs   ModbusFunctionCode = 2
	ReadHoldingRegisters ModbusFunctionCode = 3
	ReadInputRegisters   ModbusFunctionCode = 4

	// Write functions
	WriteSingleCoil        ModbusFunctionCode = 5
	WriteSingleRegister    ModbusFunctionCode = 6
	WriteMultipleCoils     ModbusFunctionCode = 15
	WriteMultipleRegisters ModbusFunctionCode = 16
)

// NewModbusHandler creates a new Modbus protocol handler
func NewModbusHandler(logger *zap.Logger) ProtocolHandler {
	return &ModbusHandler{
		logger: logger,
		config: &ModbusConfig{
			DefaultTimeout:    5 * time.Second,
			DefaultUnitID:     1,
			MaxConnections:    100,
			ConnectionTimeout: 10 * time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			EnableKeepAlive:   true,
		},
	}
}

// Connect establishes a connection to a Modbus device
func (m *ModbusHandler) Connect(device *Device) error {
	connectionKey := fmt.Sprintf("%s:%d", device.Address, device.Port)

	// Check if connection already exists
	if connInterface, exists := m.connections.Load(connectionKey); exists {
		conn := connInterface.(*ModbusConnection)
		conn.mutex.RLock()
		isConnected := conn.isConnected
		conn.mutex.RUnlock()

		if isConnected {
			device.ConnectionID = connectionKey
			return nil
		}
	}

	// Create new connection
	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", device.Address, device.Port))
	handler.Timeout = m.config.DefaultTimeout
	handler.SlaveId = m.getUnitID(device)

	if err := handler.Connect(); err != nil {
		return fmt.Errorf("failed to connect to Modbus device: %w", err)
	}

	client := modbus.NewClient(handler)

	conn := &ModbusConnection{
		client:      client,
		handler:     handler,
		lastUsed:    time.Now(),
		deviceID:    device.ID,
		isConnected: true,
		createdAt:   time.Now(),
	}

	m.connections.Store(connectionKey, conn)
	device.ConnectionID = connectionKey

	m.logger.Info("Modbus connection established",
		zap.String("device_id", device.ID),
		zap.String("address", device.Address),
		zap.Int("port", device.Port),
	)

	return nil
}

// Disconnect closes the connection to a Modbus device
func (m *ModbusHandler) Disconnect(device *Device) error {
	if device.ConnectionID == "" {
		return nil
	}

	connInterface, exists := m.connections.Load(device.ConnectionID)
	if !exists {
		return nil
	}

	conn := connInterface.(*ModbusConnection)
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if conn.handler != nil {
		conn.handler.Close()
	}

	conn.isConnected = false
	m.connections.Delete(device.ConnectionID)

	m.logger.Info("Modbus connection closed", zap.String("device_id", device.ID))
	return nil
}

// IsConnected checks if the device is connected
func (m *ModbusHandler) IsConnected(device *Device) bool {
	if device.ConnectionID == "" {
		return false
	}

	connInterface, exists := m.connections.Load(device.ConnectionID)
	if !exists {
		return false
	}

	conn := connInterface.(*ModbusConnection)
	conn.mutex.RLock()
	defer conn.mutex.RUnlock()

	return conn.isConnected
}

// ReadTag reads a single tag value from a Modbus device
func (m *ModbusHandler) ReadTag(device *Device, tag *Tag) (interface{}, error) {
	conn, err := m.getConnection(device)
	if err != nil {
		return nil, err
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	addr, err := m.parseAddress(tag.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid Modbus address %s: %w", tag.Address, err)
	}

	// Update last used time
	conn.lastUsed = time.Now()

	var result []byte

	switch addr.FunctionCode {
	case ReadCoils:
		coils, err := conn.client.ReadCoils(addr.Address, addr.Count)
		if err != nil {
			return nil, err
		}
		result = coils

	case ReadDiscreteInputs:
		inputs, err := conn.client.ReadDiscreteInputs(addr.Address, addr.Count)
		if err != nil {
			return nil, err
		}
		result = inputs

	case ReadHoldingRegisters:
		registers, err := conn.client.ReadHoldingRegisters(addr.Address, addr.Count)
		if err != nil {
			return nil, err
		}
		result = registers

	case ReadInputRegisters:
		registers, err := conn.client.ReadInputRegisters(addr.Address, addr.Count)
		if err != nil {
			return nil, err
		}
		result = registers

	default:
		return nil, fmt.Errorf("unsupported read function code: %d", addr.FunctionCode)
	}

	// Convert binary result to appropriate data type
	return m.convertFromModbus(result, tag.DataType, addr.FunctionCode)
}

// WriteTag writes a value to a Modbus device
func (m *ModbusHandler) WriteTag(device *Device, tag *Tag, value interface{}) error {
	if !tag.Writable {
		return fmt.Errorf("tag %s is not writable", tag.ID)
	}

	conn, err := m.getConnection(device)
	if err != nil {
		return err
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	addr, err := m.parseAddressForWrite(tag.Address)
	if err != nil {
		return fmt.Errorf("invalid Modbus address %s: %w", tag.Address, err)
	}

	conn.lastUsed = time.Now()

	switch addr.FunctionCode {
	case WriteSingleCoil:
		boolVal, ok := value.(bool)
		if !ok {
			return fmt.Errorf("expected boolean value for coil write")
		}
		var coilValue uint16
		if boolVal {
			coilValue = 0xFF00
		}
		_, err = conn.client.WriteSingleCoil(addr.Address, coilValue)

	case WriteSingleRegister:
		regValue, err := m.convertToModbusRegister(value, tag.DataType)
		if err != nil {
			return err
		}
		_, err = conn.client.WriteSingleRegister(addr.Address, regValue)

	case WriteMultipleRegisters:
		regValues, err := m.convertToModbusRegisters(value, tag.DataType, addr.Count)
		if err != nil {
			return err
		}
		_, err = conn.client.WriteMultipleRegisters(addr.Address, addr.Count, regValues)

	default:
		return fmt.Errorf("unsupported write function code: %d", addr.FunctionCode)
	}

	return err
}

// ReadMultipleTags efficiently reads multiple tags in a single operation
func (m *ModbusHandler) ReadMultipleTags(device *Device, tags []*Tag) (map[string]interface{}, error) {
	if len(tags) == 0 {
		return make(map[string]interface{}), nil
	}

	conn, err := m.getConnection(device)
	if err != nil {
		return nil, err
	}

	results := make(map[string]interface{})

	// Group tags by function code and consecutive addresses for batch reading
	batches := m.groupTagsForBatchRead(tags)

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	for _, batch := range batches {
		batchResults, err := m.readTagBatch(conn, batch)
		if err != nil {
			// If batch read fails, fall back to individual reads
			for _, tag := range batch {
				if value, readErr := m.readSingleTag(conn, tag); readErr == nil {
					results[tag.ID] = value
				}
			}
		} else {
			for tagID, value := range batchResults {
				results[tagID] = value
			}
		}
	}

	return results, nil
}

// DiscoverDevices scans a network range for Modbus devices
func (m *ModbusHandler) DiscoverDevices(ctx context.Context, networkRange string) ([]*Device, error) {
	devices := make([]*Device, 0)

	// Parse network range (e.g., "192.168.1.0/24")
	_, network, err := net.ParseCIDR(networkRange)
	if err != nil {
		return devices, fmt.Errorf("invalid network range: %w", err)
	}

	// Common Modbus ports to scan
	ports := []int{502, 503, 10502}

	// Scan network
	for ip := network.IP.Mask(network.Mask); network.Contains(ip); m.incrementIP(ip) {
		for _, port := range ports {
			select {
			case <-ctx.Done():
				return devices, ctx.Err()
			default:
				if device := m.probeModbusDevice(ctx, ip.String(), port); device != nil {
					devices = append(devices, device)
				}
			}
		}
	}

	return devices, nil
}

// GetDeviceInfo retrieves detailed information about a Modbus device
func (m *ModbusHandler) GetDeviceInfo(device *Device) (*DeviceInfo, error) {
	// Modbus doesn't have built-in device identification
	// This would need to be implemented based on specific device protocols
	return &DeviceInfo{
		Vendor:         "Unknown",
		Model:          "Modbus Device",
		Capabilities:   []string{"modbus-tcp", "modbus-rtu"},
		MaxConnections: 1, // Most Modbus devices support single connection
		SupportedRates: []int{9600, 19200, 38400, 57600, 115200},
	}, nil
}

// GetSupportedDataTypes returns the data types supported by Modbus
func (m *ModbusHandler) GetSupportedDataTypes() []string {
	return []string{
		string(DataTypeBool),
		string(DataTypeInt16),
		string(DataTypeUInt16),
		string(DataTypeInt32),
		string(DataTypeUInt32),
		string(DataTypeFloat32),
	}
}

// ValidateTagAddress validates a Modbus address format
func (m *ModbusHandler) ValidateTagAddress(address string) error {
	_, err := m.parseAddress(address)
	return err
}

// Ping tests connectivity to a Modbus device
func (m *ModbusHandler) Ping(device *Device) error {
	conn, err := m.getConnection(device)
	if err != nil {
		return err
	}

	// Perform a simple read operation to test connectivity
	// Try to read a single coil at address 0 (if it exists)
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	// Try to read a single holding register (address 0)
	_, err = conn.client.ReadHoldingRegisters(0, 1)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	conn.lastUsed = time.Now()
	return nil
}

// GetDiagnostics returns diagnostic information for a Modbus device
func (m *ModbusHandler) GetDiagnostics(device *Device) (*Diagnostics, error) {
	conn, err := m.getConnection(device)
	if err != nil {
		return &Diagnostics{
			IsHealthy: false,
			ErrorCount: 1,
			SuccessRate: 0.0,
		}, nil
	}

	conn.mutex.RLock()
	defer conn.mutex.RUnlock()

	return &Diagnostics{
		IsHealthy:         conn.isConnected,
		LastCommunication: conn.lastUsed,
		ConnectionUptime:  time.Since(conn.createdAt),
		SuccessRate:       1.0, // TODO: Calculate actual success rate
	}, nil
}

// cleanupStaleConnections removes stale connections from the pool
func (m *ModbusHandler) cleanupStaleConnections() {
	var staleConnections []string
	
	m.connections.Range(func(key, value interface{}) bool {
		conn := value.(*ModbusConnection)
		conn.mutex.RLock()
		isConnected := conn.isConnected
		lastUsed := conn.lastUsed
		conn.mutex.RUnlock()
		
		// Remove connections that are disconnected or haven't been used recently
		if !isConnected || time.Since(lastUsed) > m.config.ConnectionTimeout*2 {
			staleConnections = append(staleConnections, key.(string))
		}
		return true
	})
	
	// Remove stale connections
	for _, connKey := range staleConnections {
		if connInterface, exists := m.connections.Load(connKey); exists {
			conn := connInterface.(*ModbusConnection)
			conn.mutex.Lock()
			if conn.handler != nil {
				conn.handler.Close()
			}
			conn.mutex.Unlock()
			m.connections.Delete(connKey)
		}
	}
}

// Helper methods

func (m *ModbusHandler) getConnection(device *Device) (*ModbusConnection, error) {
	if device.ConnectionID == "" {
		return nil, fmt.Errorf("device not connected")
	}

	connInterface, exists := m.connections.Load(device.ConnectionID)
	if !exists {
		return nil, fmt.Errorf("connection not found")
	}

	conn := connInterface.(*ModbusConnection)
	
	// Check connection health with proper locking
	conn.mutex.RLock()
	isConnected := conn.isConnected
	conn.mutex.RUnlock()
	
	if !isConnected {
		// Clean up stale connection
		m.connections.Delete(device.ConnectionID)
		return nil, fmt.Errorf("connection is closed")
	}

	// Validate connection health
	if conn.handler == nil {
		m.connections.Delete(device.ConnectionID)
		return nil, fmt.Errorf("connection handler is nil")
	}

	return conn, nil
}

func (m *ModbusHandler) getUnitID(device *Device) byte {
	if unitID, exists := device.Config["unit_id"]; exists {
		if id, ok := unitID.(int); ok {
			return byte(id)
		}
	}
	return m.config.DefaultUnitID
}

// parseAddress parses Modbus address strings like "40001", "00001", "30001", etc.
func (m *ModbusHandler) parseAddress(address string) (*ModbusAddress, error) {
	// Remove any spaces
	address = strings.TrimSpace(address)

	if len(address) < 5 {
		return nil, fmt.Errorf("address too short")
	}

	// Parse the address number
	addr, err := strconv.ParseUint(address, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid address format: %w", err)
	}

	var funcCode ModbusFunctionCode
	var baseAddr uint16

	// Determine function code based on address range
	switch {
	case addr >= 1 && addr <= 9999: // Coils (0x)
		funcCode = ReadCoils
		baseAddr = uint16(addr - 1)
	case addr >= 10001 && addr <= 19999: // Discrete Inputs (1x)
		funcCode = ReadDiscreteInputs
		baseAddr = uint16(addr - 10001)
	case addr >= 30001 && addr <= 39999: // Input Registers (3x)
		funcCode = ReadInputRegisters
		baseAddr = uint16(addr - 30001)
	case addr >= 40001 && addr <= 49999: // Holding Registers (4x)
		funcCode = ReadHoldingRegisters
		baseAddr = uint16(addr - 40001)
	default:
		return nil, fmt.Errorf("address out of range: %d", addr)
	}

	return &ModbusAddress{
		FunctionCode: funcCode,
		Address:      baseAddr,
		Count:        1, // Default to single register/coil
		UnitID:       m.config.DefaultUnitID,
	}, nil
}

// parseAddressForWrite parses Modbus address strings for write operations
func (m *ModbusHandler) parseAddressForWrite(address string) (*ModbusAddress, error) {
	// Remove any spaces
	address = strings.TrimSpace(address)

	if len(address) < 5 {
		return nil, fmt.Errorf("address too short")
	}

	// Parse the address number
	addr, err := strconv.ParseUint(address, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid address format: %w", err)
	}

	var funcCode ModbusFunctionCode
	var baseAddr uint16

	// Determine function code based on address range for write operations
	switch {
	case addr >= 1 && addr <= 9999: // Coils (0x)
		funcCode = WriteSingleCoil
		baseAddr = uint16(addr - 1)
	case addr >= 40001 && addr <= 49999: // Holding Registers (4x)
		funcCode = WriteSingleRegister
		baseAddr = uint16(addr - 40001)
	default:
		return nil, fmt.Errorf("address not writable: %d", addr)
	}

	return &ModbusAddress{
		FunctionCode: funcCode,
		Address:      baseAddr,
		Count:        1, // Default to single register/coil
		UnitID:       m.config.DefaultUnitID,
	}, nil
}

func (m *ModbusHandler) convertFromModbus(data []byte, dataType string, funcCode ModbusFunctionCode) (interface{}, error) {
	switch DataType(dataType) {
	case DataTypeBool:
		if funcCode == ReadCoils || funcCode == ReadDiscreteInputs {
			return len(data) > 0 && data[0]&0x01 != 0, nil
		}
		return binary.BigEndian.Uint16(data) != 0, nil

	case DataTypeInt16:
		if len(data) < 2 {
			return nil, fmt.Errorf("insufficient data for int16")
		}
		return int16(binary.BigEndian.Uint16(data)), nil

	case DataTypeUInt16:
		if len(data) < 2 {
			return nil, fmt.Errorf("insufficient data for uint16")
		}
		return binary.BigEndian.Uint16(data), nil

	case DataTypeInt32:
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for int32")
		}
		return int32(binary.BigEndian.Uint32(data)), nil

	case DataTypeUInt32:
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for uint32")
		}
		return binary.BigEndian.Uint32(data), nil

	case DataTypeFloat32:
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for float32")
		}
		// Convert to IEEE 754 float32
		bits := binary.BigEndian.Uint32(data)
		return float32(bits), nil

	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}

func (m *ModbusHandler) convertToModbusRegister(value interface{}, dataType string) (uint16, error) {
	switch DataType(dataType) {
	case DataTypeBool:
		if boolVal, ok := value.(bool); ok {
			if boolVal {
				return 1, nil
			}
			return 0, nil
		}
		return 0, fmt.Errorf("expected boolean value")

	case DataTypeUInt16:
		if intVal, ok := value.(int); ok {
			return uint16(intVal), nil
		}
		if uintVal, ok := value.(uint16); ok {
			return uintVal, nil
		}
		return 0, fmt.Errorf("expected uint16 value")

	case DataTypeInt16:
		if intVal, ok := value.(int); ok {
			return uint16(int16(intVal)), nil
		}
		if int16Val, ok := value.(int16); ok {
			return uint16(int16Val), nil
		}
		return 0, fmt.Errorf("expected int16 value")

	default:
		return 0, fmt.Errorf("unsupported data type for single register: %s", dataType)
	}
}

func (m *ModbusHandler) convertToModbusRegisters(value interface{}, dataType string, count uint16) ([]byte, error) {
	// Implementation for multi-register values (int32, uint32, float32, etc.)
	// This would handle the conversion of larger data types to byte arrays
	return nil, fmt.Errorf("multi-register conversion not implemented")
}

func (m *ModbusHandler) groupTagsForBatchRead(tags []*Tag) [][]*Tag {
	// Group consecutive tags by function code for efficient batch reading
	// This is a simplified implementation - real optimization would be more complex
	var batches [][]*Tag

	for _, tag := range tags {
		// For now, just put each tag in its own batch
		// Real implementation would group consecutive addresses
		batches = append(batches, []*Tag{tag})
	}

	return batches
}

func (m *ModbusHandler) readTagBatch(conn *ModbusConnection, tags []*Tag) (map[string]interface{}, error) {
	// Batch read implementation - simplified for now
	results := make(map[string]interface{})

	for _, tag := range tags {
		if value, err := m.readSingleTag(conn, tag); err == nil {
			results[tag.ID] = value
		}
	}

	return results, nil
}

func (m *ModbusHandler) readSingleTag(conn *ModbusConnection, tag *Tag) (interface{}, error) {
	// Individual tag read implementation
	addr, err := m.parseAddress(tag.Address)
	if err != nil {
		return nil, err
	}

	var result []byte

	switch addr.FunctionCode {
	case ReadHoldingRegisters:
		registers, err := conn.client.ReadHoldingRegisters(addr.Address, 1)
		if err != nil {
			return nil, err
		}
		result = registers
		// Add other function codes as needed
	}

	return m.convertFromModbus(result, tag.DataType, addr.FunctionCode)
}

func (m *ModbusHandler) probeModbusDevice(ctx context.Context, ip string, port int) *Device {
	// Quick probe to see if a Modbus device responds at this address
	timeout := 2 * time.Second

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), timeout)
	if err != nil {
		return nil
	}
	conn.Close()

	// Create a minimal device entry for discovered device
	return &Device{
		ID:       fmt.Sprintf("modbus-%s-%d", ip, port),
		Name:     fmt.Sprintf("Modbus Device at %s:%d", ip, port),
		Protocol: "modbus-tcp",
		Address:  ip,
		Port:     port,
		Config:   make(map[string]interface{}),
	}
}

func (m *ModbusHandler) incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
