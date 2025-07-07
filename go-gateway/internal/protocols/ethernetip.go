package protocols

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EtherNetIPHandler implements the ProtocolHandler interface for EtherNet/IP
type EtherNetIPHandler struct {
	logger      *zap.Logger
	connections sync.Map // map[string]*EtherNetIPConnection
	config      *EtherNetIPConfig
	sessions    sync.Map // map[string]*CIPSession
}

// EtherNetIPConnection represents an EtherNet/IP connection with CIP session management
type EtherNetIPConnection struct {
	tcpConn      net.Conn
	udpConn      net.Conn
	sessionID    uint32
	connectionID uint32
	lastUsed     time.Time
	deviceID     string
	isConnected  bool
	mutex        sync.RWMutex

	// Connection state
	inUse          bool
	createdAt      time.Time
	sequenceNumber uint16

	// CIP-specific state
	vendorID     uint16
	deviceType   uint16
	productCode  uint16
	revision     uint16
	serialNumber uint32
	productName  string
}

// EtherNetIPConfig holds EtherNet/IP specific configuration
type EtherNetIPConfig struct {
	DefaultTimeout    time.Duration `yaml:"default_timeout"`
	TCPPort           int           `yaml:"tcp_port"`
	UDPPort           int           `yaml:"udp_port"`
	MaxConnections    int           `yaml:"max_connections"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
	SessionTimeout    time.Duration `yaml:"session_timeout"`
	EnableImplicitIO  bool          `yaml:"enable_implicit_io"`
	MaxPacketSize     int           `yaml:"max_packet_size"`
}

// CIP Constants
const (
	// EtherNet/IP Encapsulation Command Codes
	CIPCommandNOP               = 0x0000
	CIPCommandListServices      = 0x0004
	CIPCommandListIdentity      = 0x0063
	CIPCommandListInterfaces    = 0x0064
	CIPCommandRegisterSession   = 0x0065
	CIPCommandUnregisterSession = 0x0066
	CIPCommandSendRRData        = 0x006F
	CIPCommandSendUnitData      = 0x0070
	CIPCommandIndicateStatus    = 0x0072
	CIPCommandCancel            = 0x0073

	// CIP Service Codes
	CIPServiceGetAll                = 0x01
	CIPServiceSetAll                = 0x02
	CIPServiceGetAttributeList      = 0x03
	CIPServiceSetAttributeList      = 0x04
	CIPServiceReset                 = 0x05
	CIPServiceStart                 = 0x06
	CIPServiceStop                  = 0x07
	CIPServiceCreate                = 0x08
	CIPServiceDelete                = 0x09
	CIPServiceMultipleServicePacket = 0x0A
	CIPServiceApplyAttributes       = 0x0D
	CIPServiceGetAttributeSingle    = 0x0E
	CIPServiceSetAttributeSingle    = 0x10
	CIPServiceFindNextObject        = 0x11
	CIPServiceRestore               = 0x15
	CIPServiceSave                  = 0x16
	CIPServiceNOP                   = 0x17
	CIPServiceGetMember             = 0x18
	CIPServiceSetMember             = 0x19
	CIPServiceInsertMember          = 0x1A
	CIPServiceRemoveMember          = 0x1B
	CIPServiceGroupSync             = 0x1C

	// CIP Object Class IDs
	CIPClassIdentity          = 0x01
	CIPClassMessageRouter     = 0x02
	CIPClassDeviceNet         = 0x03
	CIPClassAssembly          = 0x04
	CIPClassConnection        = 0x05
	CIPClassConnectionManager = 0x06
	CIPClassRegister          = 0x07
	CIPClassDiscreteInput     = 0x08
	CIPClassDiscreteOutput    = 0x09
	CIPClassAnalogInput       = 0x0A
	CIPClassAnalogOutput      = 0x0B
	CIPClassPresenceSensing   = 0x0E
	CIPClassParameter         = 0x0F
	CIPClassParameterGroup    = 0x10
	CIPClassGroup             = 0x12
	CIPClassFile              = 0x37
	CIPClassSymbol            = 0x6B
	CIPClassTemplate          = 0x6C

	// CIP Data Types
	CIPDataTypeBool   = 0xC1
	CIPDataTypeSint   = 0xC2
	CIPDataTypeInt    = 0xC3
	CIPDataTypeDint   = 0xC4
	CIPDataTypeLint   = 0xC5
	CIPDataTypeUsint  = 0xC6
	CIPDataTypeUint   = 0xC7
	CIPDataTypeUdint  = 0xC8
	CIPDataTypeUlint  = 0xC9
	CIPDataTypeReal   = 0xCA
	CIPDataTypeLreal  = 0xCB
	CIPDataTypeString = 0xD0
	CIPDataTypeStruct = 0xA0

	// Default ports
	DefaultTCPPort = 44818
	DefaultUDPPort = 2222
)

// CIP Encapsulation Header
type CIPEncapsulationHeader struct {
	Command       uint16
	Length        uint16
	SessionHandle uint32
	Status        uint32
	Context       [8]byte
	Options       uint32
}

// CIP Common Packet Format
type CIPCommonPacketFormat struct {
	ItemCount uint16
	TypeID    uint16
	Length    uint16
	Data      []byte
}

// CIP Request/Response structures
type CIPRequest struct {
	Service        uint8
	RequestPath    []byte
	RequestData    []byte
	RequestPathLen uint8
}

type CIPResponse struct {
	Service        uint8
	GeneralStatus  uint8
	ExtendedStatus []byte
	ResponseData   []byte
}

// CIP Identity Object attributes
type CIPIdentityObject struct {
	VendorID     uint16
	DeviceType   uint16
	ProductCode  uint16
	Revision     uint16
	Status       uint16
	SerialNumber uint32
	ProductName  string
	State        uint8
}

// EtherNet/IP Address represents parsed EtherNet/IP tag addresses
type EtherNetIPAddress struct {
	TagName      string
	InstanceID   uint32
	AttributeID  uint8
	ElementIndex uint32
	DataType     uint8
	IsSymbolic   bool
	IsArray      bool
	ArraySize    uint32
}

// NewEtherNetIPHandler creates a new EtherNet/IP protocol handler
func NewEtherNetIPHandler(logger *zap.Logger) ProtocolHandler {
	return &EtherNetIPHandler{
		logger: logger,
		config: &EtherNetIPConfig{
			DefaultTimeout:    10 * time.Second,
			TCPPort:           DefaultTCPPort,
			UDPPort:           DefaultUDPPort,
			MaxConnections:    50,
			ConnectionTimeout: 15 * time.Second,
			SessionTimeout:    30 * time.Second,
			EnableImplicitIO:  true,
			MaxPacketSize:     1500,
		},
	}
}

// Connect establishes a connection to an EtherNet/IP device
func (e *EtherNetIPHandler) Connect(device *Device) error {
	connectionKey := fmt.Sprintf("%s:%d", device.Address, device.Port)

	// Check if connection already exists
	if connInterface, exists := e.connections.Load(connectionKey); exists {
		conn := connInterface.(*EtherNetIPConnection)
		conn.mutex.RLock()
		isConnected := conn.isConnected
		conn.mutex.RUnlock()

		if isConnected {
			device.ConnectionID = connectionKey
			return nil
		}
	}

	// Set default port if not specified
	port := device.Port
	if port == 0 {
		port = DefaultTCPPort
	}

	// Establish TCP connection for explicit messaging
	tcpConn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", device.Address, port), e.config.ConnectionTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to EtherNet/IP device: %w", err)
	}

	conn := &EtherNetIPConnection{
		tcpConn:        tcpConn,
		lastUsed:       time.Now(),
		deviceID:       device.ID,
		isConnected:    true,
		createdAt:      time.Now(),
		sequenceNumber: 0,
	}

	// Register CIP session
	sessionID, err := e.registerSession(conn)
	if err != nil {
		tcpConn.Close()
		return fmt.Errorf("failed to register CIP session: %w", err)
	}

	conn.sessionID = sessionID

	// Get device identity information
	identity, err := e.getDeviceIdentity(conn)
	if err != nil {
		e.logger.Warn("Failed to get device identity", zap.Error(err))
	} else {
		conn.vendorID = identity.VendorID
		conn.deviceType = identity.DeviceType
		conn.productCode = identity.ProductCode
		conn.revision = identity.Revision
		conn.serialNumber = identity.SerialNumber
		conn.productName = identity.ProductName
	}

	e.connections.Store(connectionKey, conn)
	device.ConnectionID = connectionKey

	e.logger.Info("EtherNet/IP connection established",
		zap.String("device_id", device.ID),
		zap.String("address", device.Address),
		zap.Int("port", port),
		zap.Uint32("session_id", sessionID),
		zap.String("product_name", conn.productName),
	)

	return nil
}

// Disconnect closes the connection to an EtherNet/IP device
func (e *EtherNetIPHandler) Disconnect(device *Device) error {
	if device.ConnectionID == "" {
		return nil
	}

	connInterface, exists := e.connections.Load(device.ConnectionID)
	if !exists {
		return nil
	}

	conn := connInterface.(*EtherNetIPConnection)
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	// Unregister CIP session
	if conn.sessionID != 0 {
		e.unregisterSession(conn)
	}

	// Close TCP connection
	if conn.tcpConn != nil {
		conn.tcpConn.Close()
	}

	// Close UDP connection if exists
	if conn.udpConn != nil {
		conn.udpConn.Close()
	}

	conn.isConnected = false
	e.connections.Delete(device.ConnectionID)

	e.logger.Info("EtherNet/IP connection closed", zap.String("device_id", device.ID))
	return nil
}

// IsConnected checks if the device is connected
func (e *EtherNetIPHandler) IsConnected(device *Device) bool {
	if device.ConnectionID == "" {
		return false
	}

	connInterface, exists := e.connections.Load(device.ConnectionID)
	if !exists {
		return false
	}

	conn := connInterface.(*EtherNetIPConnection)
	conn.mutex.RLock()
	defer conn.mutex.RUnlock()

	return conn.isConnected
}

// ReadTag reads a single tag value from an EtherNet/IP device
func (e *EtherNetIPHandler) ReadTag(device *Device, tag *Tag) (interface{}, error) {
	conn, err := e.getConnection(device)
	if err != nil {
		return nil, err
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	addr, err := e.parseAddress(tag.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid EtherNet/IP address %s: %w", tag.Address, err)
	}

	conn.lastUsed = time.Now()

	// Build CIP request for tag read
	var request *CIPRequest
	if addr.IsSymbolic {
		// Use symbolic addressing (tag name)
		request = &CIPRequest{
			Service:     CIPServiceGetAttributeSingle,
			RequestPath: e.buildSymbolicPath(addr.TagName),
			RequestData: []byte{},
		}
	} else {
		// Use instance-based addressing
		request = &CIPRequest{
			Service:     CIPServiceGetAttributeSingle,
			RequestPath: e.buildInstancePath(CIPClassSymbol, addr.InstanceID, addr.AttributeID),
			RequestData: []byte{},
		}
	}

	// Send CIP request
	response, err := e.sendCIPRequest(conn, request)
	if err != nil {
		return nil, err
	}

	if response.GeneralStatus != 0 {
		return nil, fmt.Errorf("CIP error: status 0x%02X", response.GeneralStatus)
	}

	// Convert response data to appropriate Go type
	return e.convertFromCIP(response.ResponseData, addr.DataType)
}

// WriteTag writes a value to an EtherNet/IP device
func (e *EtherNetIPHandler) WriteTag(device *Device, tag *Tag, value interface{}) error {
	if !tag.Writable {
		return fmt.Errorf("tag %s is not writable", tag.ID)
	}

	conn, err := e.getConnection(device)
	if err != nil {
		return err
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	addr, err := e.parseAddress(tag.Address)
	if err != nil {
		return fmt.Errorf("invalid EtherNet/IP address %s: %w", tag.Address, err)
	}

	conn.lastUsed = time.Now()

	// Convert value to CIP format
	cipData, err := e.convertToCIP(value, addr.DataType)
	if err != nil {
		return err
	}

	// Build CIP request for tag write
	var request *CIPRequest
	if addr.IsSymbolic {
		request = &CIPRequest{
			Service:     CIPServiceSetAttributeSingle,
			RequestPath: e.buildSymbolicPath(addr.TagName),
			RequestData: cipData,
		}
	} else {
		request = &CIPRequest{
			Service:     CIPServiceSetAttributeSingle,
			RequestPath: e.buildInstancePath(CIPClassSymbol, addr.InstanceID, addr.AttributeID),
			RequestData: cipData,
		}
	}

	// Send CIP request
	response, err := e.sendCIPRequest(conn, request)
	if err != nil {
		return err
	}

	if response.GeneralStatus != 0 {
		return fmt.Errorf("CIP write error: status 0x%02X", response.GeneralStatus)
	}

	return nil
}

// ReadMultipleTags efficiently reads multiple tags using Multiple Service Packet
func (e *EtherNetIPHandler) ReadMultipleTags(device *Device, tags []*Tag) (map[string]interface{}, error) {
	if len(tags) == 0 {
		return make(map[string]interface{}), nil
	}

	conn, err := e.getConnection(device)
	if err != nil {
		return nil, err
	}

	results := make(map[string]interface{})

	// Group tags into batches for optimal performance
	batches := e.groupTagsForBatchRead(tags, 50) // Max 50 tags per batch

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	for _, batch := range batches {
		batchResults, err := e.readTagBatch(conn, batch)
		if err != nil {
			// Fall back to individual reads if batch fails
			for _, tag := range batch {
				if value, readErr := e.readSingleTag(conn, tag); readErr == nil {
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

// DiscoverDevices scans a network range for EtherNet/IP devices
func (e *EtherNetIPHandler) DiscoverDevices(ctx context.Context, networkRange string) ([]*Device, error) {
	devices := make([]*Device, 0)

	// Parse network range
	_, network, err := net.ParseCIDR(networkRange)
	if err != nil {
		return devices, fmt.Errorf("invalid network range: %w", err)
	}

	// Common EtherNet/IP ports to scan
	ports := []int{DefaultTCPPort, 44818, 2222}

	// Scan network with timeout
	scanCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for ip := network.IP.Mask(network.Mask); network.Contains(ip); e.incrementIP(ip) {
		for _, port := range ports {
			select {
			case <-scanCtx.Done():
				return devices, scanCtx.Err()
			default:
				if device := e.probeEtherNetIPDevice(scanCtx, ip.String(), port); device != nil {
					devices = append(devices, device)
				}
			}
		}
	}

	return devices, nil
}

// GetDeviceInfo retrieves detailed information about an EtherNet/IP device
func (e *EtherNetIPHandler) GetDeviceInfo(device *Device) (*DeviceInfo, error) {
	conn, err := e.getConnection(device)
	if err != nil {
		return nil, err
	}

	conn.mutex.RLock()
	defer conn.mutex.RUnlock()

	info := &DeviceInfo{
		Vendor:          e.getVendorName(conn.vendorID),
		Model:           conn.productName,
		SerialNumber:    fmt.Sprintf("%d", conn.serialNumber),
		FirmwareVersion: fmt.Sprintf("%d.%d", conn.revision>>8, conn.revision&0xFF),
		Capabilities:    []string{"ethernet-ip", "cip-explicit", "cip-implicit"},
		MaxConnections:  10,                   // Typical for Allen-Bradley PLCs
		SupportedRates:  []int{10, 100, 1000}, // Ethernet speeds in Mbps
		CustomInfo: map[string]string{
			"vendor_id":    fmt.Sprintf("0x%04X", conn.vendorID),
			"device_type":  fmt.Sprintf("0x%04X", conn.deviceType),
			"product_code": fmt.Sprintf("0x%04X", conn.productCode),
		},
	}

	return info, nil
}

// GetSupportedDataTypes returns the data types supported by EtherNet/IP
func (e *EtherNetIPHandler) GetSupportedDataTypes() []string {
	return []string{
		string(DataTypeBool),
		string(DataTypeInt16),
		string(DataTypeUInt16),
		string(DataTypeInt32),
		string(DataTypeUInt32),
		string(DataTypeInt64),
		string(DataTypeUInt64),
		string(DataTypeFloat32),
		string(DataTypeFloat64),
		string(DataTypeString),
	}
}

// ValidateTagAddress validates an EtherNet/IP tag address format
func (e *EtherNetIPHandler) ValidateTagAddress(address string) error {
	_, err := e.parseAddress(address)
	return err
}

// Ping tests connectivity to an EtherNet/IP device
func (e *EtherNetIPHandler) Ping(device *Device) error {
	conn, err := e.getConnection(device)
	if err != nil {
		return err
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	// Send NOP command as ping
	header := CIPEncapsulationHeader{
		Command:       CIPCommandNOP,
		Length:        0,
		SessionHandle: conn.sessionID,
		Status:        0,
		Options:       0,
	}

	err = e.sendEncapsulationHeader(conn, &header)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Read response
	_, err = e.readEncapsulationHeader(conn)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	conn.lastUsed = time.Now()
	return nil
}

// GetDiagnostics returns diagnostic information for an EtherNet/IP device
func (e *EtherNetIPHandler) GetDiagnostics(device *Device) (*Diagnostics, error) {
	conn, err := e.getConnection(device)
	if err != nil {
		return nil, err
	}

	conn.mutex.RLock()
	defer conn.mutex.RUnlock()

	return &Diagnostics{
		IsHealthy:         conn.isConnected,
		LastCommunication: conn.lastUsed,
		ConnectionUptime:  time.Since(conn.createdAt),
		ProtocolDiagnostics: map[string]interface{}{
			"session_id":      conn.sessionID,
			"connection_id":   conn.connectionID,
			"vendor_id":       conn.vendorID,
			"device_type":     conn.deviceType,
			"product_code":    conn.productCode,
			"product_name":    conn.productName,
			"sequence_number": conn.sequenceNumber,
		},
	}, nil
}

// Helper methods will be implemented in the next part due to length constraints
// This includes CIP session management, packet building, data conversion, etc.

// getConnection retrieves an active connection for a device
func (e *EtherNetIPHandler) getConnection(device *Device) (*EtherNetIPConnection, error) {
	if device.ConnectionID == "" {
		return nil, fmt.Errorf("device not connected")
	}

	connInterface, exists := e.connections.Load(device.ConnectionID)
	if !exists {
		return nil, fmt.Errorf("connection not found")
	}

	conn := connInterface.(*EtherNetIPConnection)
	if !conn.isConnected {
		return nil, fmt.Errorf("connection is closed")
	}

	return conn, nil
}

// incrementIP increments an IP address for network scanning
func (e *EtherNetIPHandler) incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// parseAddress parses an EtherNet/IP tag address string
func (e *EtherNetIPHandler) parseAddress(address string) (*EtherNetIPAddress, error) {
	addr := &EtherNetIPAddress{
		IsSymbolic:  true,
		AttributeID: 1, // Default attribute ID for symbolic tags
		DataType:    CIPDataTypeDint, // Default data type
	}

	// Check for array index (e.g., "MyArray[5]")
	openBracket := strings.Index(address, "[")
	closeBracket := strings.LastIndex(address, "]")

	if openBracket != -1 && closeBracket != -1 && openBracket < closeBracket {
		addr.TagName = address[:openBracket]
		addr.IsArray = true
		indexStr := address[openBracket+1 : closeBracket]
		index, err := strconv.ParseUint(indexStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid array index in address %s", address)
		}
		addr.ElementIndex = uint32(index)
	} else if strings.Contains(address, "@") {
		// Instance-based addressing (e.g., "Symbol@100.1")
		parts := strings.Split(address, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid instance address format: %s", address)
		}
		addr.IsSymbolic = false
		instanceParts := strings.Split(parts[1], ".")
		if len(instanceParts) != 2 {
			return nil, fmt.Errorf("invalid instance.attribute format: %s", instanceParts[1])
		}
		instanceID, err := strconv.ParseUint(instanceParts[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid instance ID in address %s", address)
		}
		addr.InstanceID = uint32(instanceID)

		attributeID, err := strconv.ParseUint(instanceParts[1], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid attribute ID in address %s", address)
		}
		addr.AttributeID = uint8(attributeID)
	} else {
		// Simple symbolic tag name
		addr.TagName = address
	}

	return addr, nil
}


