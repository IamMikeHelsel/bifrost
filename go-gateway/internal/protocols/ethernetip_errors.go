package protocols

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EtherNet/IP Error Handling and Diagnostics

// EtherNetIPError represents an EtherNet/IP specific error
type EtherNetIPError struct {
	*ProtocolError
	SessionID      uint32
	ConnectionID   uint32
	CIPStatus      uint8
	ExtendedStatus []byte
	DeviceAddress  string
	TagAddress     string
	ErrorCategory  EtherNetIPErrorCategory
}

// EtherNetIPErrorCategory categorizes different types of EtherNet/IP errors
type EtherNetIPErrorCategory string

const (
	ErrorCategoryConnection    EtherNetIPErrorCategory = "CONNECTION"
	ErrorCategorySession       EtherNetIPErrorCategory = "SESSION"
	ErrorCategoryCIP           EtherNetIPErrorCategory = "CIP"
	ErrorCategoryEncapsulation EtherNetIPErrorCategory = "ENCAPSULATION"
	ErrorCategoryNetwork       EtherNetIPErrorCategory = "NETWORK"
	ErrorCategoryTimeout       EtherNetIPErrorCategory = "TIMEOUT"
	ErrorCategorySecurity      EtherNetIPErrorCategory = "SECURITY"
	ErrorCategoryData          EtherNetIPErrorCategory = "DATA"
	ErrorCategoryDevice        EtherNetIPErrorCategory = "DEVICE"
	ErrorCategoryProtocol      EtherNetIPErrorCategory = "PROTOCOL"
)

// CIP Status Codes (Common)
const (
	CIPStatusSuccess                    uint8 = 0x00
	CIPStatusConnectionFailure          uint8 = 0x01
	CIPStatusResourceUnavailable        uint8 = 0x02
	CIPStatusInvalidParameter           uint8 = 0x03
	CIPStatusPathSegmentError           uint8 = 0x04
	CIPStatusPathDestinationUnknown     uint8 = 0x05
	CIPStatusPartialTransfer            uint8 = 0x06
	CIPStatusConnectionLost             uint8 = 0x07
	CIPStatusServiceNotSupported        uint8 = 0x08
	CIPStatusInvalidAttributeValue      uint8 = 0x09
	CIPStatusAttributeListError         uint8 = 0x0A
	CIPStatusAlreadyInRequestedMode     uint8 = 0x0B
	CIPStatusObjectStateConflict        uint8 = 0x0C
	CIPStatusObjectAlreadyExists        uint8 = 0x0D
	CIPStatusAttributeNotSettable       uint8 = 0x0E
	CIPStatusPrivilegeViolation         uint8 = 0x0F
	CIPStatusDeviceStateConflict        uint8 = 0x10
	CIPStatusReplyDataTooLarge          uint8 = 0x11
	CIPStatusFragmentationOfPrimitive   uint8 = 0x12
	CIPStatusNotEnoughData              uint8 = 0x13
	CIPStatusAttributeNotSupported      uint8 = 0x14
	CIPStatusTooMuchData                uint8 = 0x15
	CIPStatusObjectDoesNotExist         uint8 = 0x16
	CIPStatusServiceFragmentationError  uint8 = 0x17
	CIPStatusNoStoredAttributeData      uint8 = 0x18
	CIPStatusStoreOperationFailure      uint8 = 0x19
	CIPStatusRoutingFailure             uint8 = 0x1A
	CIPStatusRoutingFailureRequestTooLarge uint8 = 0x1B
	CIPStatusRoutingFailureResponseTooLarge uint8 = 0x1C
	CIPStatusMissingAttributeListEntry  uint8 = 0x1D
	CIPStatusInvalidAttributeValueList  uint8 = 0x1E
	CIPStatusEmbeddedServiceError       uint8 = 0x1F
	CIPStatusVendorSpecificError        uint8 = 0x20
	CIPStatusInvalidParameter2          uint8 = 0x21
	CIPStatusWriteOnceValueOrMediumAlreadyWritten uint8 = 0x22
	CIPStatusInvalidReplyReceived       uint8 = 0x23
	CIPStatusBufferOverflow             uint8 = 0x24
	CIPStatusMessageFormatError         uint8 = 0x25
	CIPStatusKeyFailureInPath           uint8 = 0x26
	CIPStatusPathSizeInvalid            uint8 = 0x27
	CIPStatusUnexpectedAttributeInList  uint8 = 0x28
	CIPStatusInvalidMemberID            uint8 = 0x29
	CIPStatusMemberNotSettable          uint8 = 0x2A
	CIPStatusGroup2OnlyServerGeneralFailure uint8 = 0x2B
	CIPStatusUnknownModNetworkError     uint8 = 0x2C
	CIPStatusResponseTimeout            uint8 = 0x2D
)

// Encapsulation Status Codes
const (
	EncapStatusSuccess                 uint32 = 0x0000
	EncapStatusInvalidCommand          uint32 = 0x0001
	EncapStatusInsufficientMemory      uint32 = 0x0002
	EncapStatusIncorrectData           uint32 = 0x0003
	EncapStatusInvalidSessionHandle    uint32 = 0x0064
	EncapStatusInvalidLength           uint32 = 0x0065
	EncapStatusUnsupportedProtocol     uint32 = 0x0069
)

// NewEtherNetIPError creates a new EtherNet/IP specific error
func NewEtherNetIPError(category EtherNetIPErrorCategory, message string, operation string) *EtherNetIPError {
	return &EtherNetIPError{
		ProtocolError: NewProtocolError(string(category), message, operation),
		ErrorCategory: category,
	}
}

// NewCIPError creates a new CIP-specific error
func NewCIPError(cipStatus uint8, extendedStatus []byte, message string, operation string) *EtherNetIPError {
	cipMessage := fmt.Sprintf("%s (CIP Status: 0x%02X - %s)", message, cipStatus, GetCIPStatusDescription(cipStatus))
	
	if len(extendedStatus) > 0 {
		cipMessage += fmt.Sprintf(" [Extended Status: %v]", extendedStatus)
	}
	
	return &EtherNetIPError{
		ProtocolError:  NewProtocolError("CIP_ERROR", cipMessage, operation),
		CIPStatus:      cipStatus,
		ExtendedStatus: extendedStatus,
		ErrorCategory:  ErrorCategoryCIP,
	}
}

// NewEncapsulationError creates a new encapsulation-specific error
func NewEncapsulationError(encapStatus uint32, message string, operation string) *EtherNetIPError {
	encapMessage := fmt.Sprintf("%s (Encapsulation Status: 0x%08X - %s)", 
		message, encapStatus, GetEncapsulationStatusDescription(encapStatus))
	
	return &EtherNetIPError{
		ProtocolError: NewProtocolError("ENCAP_ERROR", encapMessage, operation),
		ErrorCategory: ErrorCategoryEncapsulation,
	}
}

// Error returns the error message
func (e *EtherNetIPError) Error() string {
	var parts []string
	
	parts = append(parts, fmt.Sprintf("[%s]", e.ErrorCategory))
	
	if e.DeviceAddress != "" {
		parts = append(parts, fmt.Sprintf("Device: %s", e.DeviceAddress))
	}
	
	if e.TagAddress != "" {
		parts = append(parts, fmt.Sprintf("Tag: %s", e.TagAddress))
	}
	
	if e.SessionID != 0 {
		parts = append(parts, fmt.Sprintf("Session: 0x%08X", e.SessionID))
	}
	
	parts = append(parts, e.ProtocolError.Error())
	
	return strings.Join(parts, " ")
}

// IsRecoverable determines if the error is recoverable
func (e *EtherNetIPError) IsRecoverable() bool {
	switch e.ErrorCategory {
	case ErrorCategoryTimeout, ErrorCategoryNetwork:
		return true
	case ErrorCategoryCIP:
		return e.isCIPErrorRecoverable()
	case ErrorCategoryConnection:
		return true // Connection errors can often be recovered by reconnecting
	default:
		return e.ProtocolError.Recoverable
	}
}

// isCIPErrorRecoverable determines if a CIP error is recoverable
func (e *EtherNetIPError) isCIPErrorRecoverable() bool {
	switch e.CIPStatus {
	case CIPStatusSuccess:
		return false // Not an error
	case CIPStatusConnectionFailure, CIPStatusConnectionLost:
		return true // Connection issues can be recovered
	case CIPStatusResourceUnavailable:
		return true // Resource might become available later
	case CIPStatusResponseTimeout:
		return true // Timeout can be retried
	case CIPStatusObjectDoesNotExist, CIPStatusAttributeNotSupported:
		return false // These are permanent configuration issues
	case CIPStatusPrivilegeViolation, CIPStatusAttributeNotSettable:
		return false // Permission/configuration issues
	default:
		return true // Most other errors might be temporary
	}
}

// GetCIPStatusDescription returns a human-readable description of a CIP status code
func GetCIPStatusDescription(status uint8) string {
	switch status {
	case CIPStatusSuccess:
		return "Success"
	case CIPStatusConnectionFailure:
		return "Connection failure"
	case CIPStatusResourceUnavailable:
		return "Resource unavailable"
	case CIPStatusInvalidParameter:
		return "Invalid parameter value"
	case CIPStatusPathSegmentError:
		return "Path segment error"
	case CIPStatusPathDestinationUnknown:
		return "Path destination unknown"
	case CIPStatusPartialTransfer:
		return "Partial transfer"
	case CIPStatusConnectionLost:
		return "Connection lost"
	case CIPStatusServiceNotSupported:
		return "Service not supported"
	case CIPStatusInvalidAttributeValue:
		return "Invalid attribute value"
	case CIPStatusAttributeListError:
		return "Attribute list error"
	case CIPStatusAlreadyInRequestedMode:
		return "Already in requested mode or state"
	case CIPStatusObjectStateConflict:
		return "Object state conflict"
	case CIPStatusObjectAlreadyExists:
		return "Object already exists"
	case CIPStatusAttributeNotSettable:
		return "Attribute not settable"
	case CIPStatusPrivilegeViolation:
		return "Privilege violation"
	case CIPStatusDeviceStateConflict:
		return "Device state conflict"
	case CIPStatusReplyDataTooLarge:
		return "Reply data too large"
	case CIPStatusFragmentationOfPrimitive:
		return "Fragmentation of a primitive value"
	case CIPStatusNotEnoughData:
		return "Not enough data"
	case CIPStatusAttributeNotSupported:
		return "Attribute not supported"
	case CIPStatusTooMuchData:
		return "Too much data"
	case CIPStatusObjectDoesNotExist:
		return "Object does not exist"
	case CIPStatusServiceFragmentationError:
		return "Service fragmentation sequence not in progress"
	case CIPStatusNoStoredAttributeData:
		return "No stored attribute data"
	case CIPStatusStoreOperationFailure:
		return "Store operation failure"
	case CIPStatusRoutingFailure:
		return "Routing failure, request packet too large"
	case CIPStatusRoutingFailureRequestTooLarge:
		return "Routing failure, request packet too large"
	case CIPStatusRoutingFailureResponseTooLarge:
		return "Routing failure, response packet too large"
	case CIPStatusMissingAttributeListEntry:
		return "Missing attribute list entry data"
	case CIPStatusInvalidAttributeValueList:
		return "Invalid attribute value list"
	case CIPStatusEmbeddedServiceError:
		return "Embedded service error"
	case CIPStatusVendorSpecificError:
		return "Vendor specific error"
	case CIPStatusInvalidParameter2:
		return "Invalid parameter"
	case CIPStatusWriteOnceValueOrMediumAlreadyWritten:
		return "Write-once value or medium already written"
	case CIPStatusInvalidReplyReceived:
		return "Invalid reply received"
	case CIPStatusBufferOverflow:
		return "Buffer overflow"
	case CIPStatusMessageFormatError:
		return "Message format error"
	case CIPStatusKeyFailureInPath:
		return "Key failure in path"
	case CIPStatusPathSizeInvalid:
		return "Path size invalid"
	case CIPStatusUnexpectedAttributeInList:
		return "Unexpected attribute in list"
	case CIPStatusInvalidMemberID:
		return "Invalid member ID"
	case CIPStatusMemberNotSettable:
		return "Member not settable"
	case CIPStatusGroup2OnlyServerGeneralFailure:
		return "Group 2 only server general failure"
	case CIPStatusUnknownModNetworkError:
		return "Unknown modbus error"
	case CIPStatusResponseTimeout:
		return "Response timeout"
	default:
		if status >= 0x80 && status <= 0xFF {
			return fmt.Sprintf("Vendor specific error (0x%02X)", status)
		}
		return fmt.Sprintf("Unknown error (0x%02X)", status)
	}
}

// GetEncapsulationStatusDescription returns a human-readable description of an encapsulation status code
func GetEncapsulationStatusDescription(status uint32) string {
	switch status {
	case EncapStatusSuccess:
		return "Success"
	case EncapStatusInvalidCommand:
		return "Invalid or unsupported encapsulation command"
	case EncapStatusInsufficientMemory:
		return "Insufficient memory resources to handle command"
	case EncapStatusIncorrectData:
		return "Poorly formed or incorrect data"
	case EncapStatusInvalidSessionHandle:
		return "Invalid session handle"
	case EncapStatusInvalidLength:
		return "Invalid length"
	case EncapStatusUnsupportedProtocol:
		return "Unsupported encapsulation protocol revision"
	default:
		return fmt.Sprintf("Unknown encapsulation error (0x%08X)", status)
	}
}

// EtherNetIP Diagnostic Information

// EtherNetIPDiagnostics provides detailed diagnostic information for EtherNet/IP devices
type EtherNetIPDiagnostics struct {
	*Diagnostics
	
	// Connection-specific diagnostics
	SessionInfo       *SessionDiagnostics       `json:"session_info"`
	EncapsulationInfo *EncapsulationDiagnostics `json:"encapsulation_info"`
	CIPInfo           *CIPDiagnostics           `json:"cip_info"`
	NetworkInfo       *NetworkDiagnostics       `json:"network_info"`
	PerformanceInfo   *PerformanceDiagnostics   `json:"performance_info"`
	
	// Error tracking
	ErrorHistory  []*EtherNetIPError `json:"error_history"`
	ErrorCounters map[string]uint64  `json:"error_counters"`
}

// SessionDiagnostics contains CIP session diagnostic information
type SessionDiagnostics struct {
	SessionID           uint32        `json:"session_id"`
	ConnectionID        uint32        `json:"connection_id"`
	SessionEstablished  time.Time     `json:"session_established"`
	SessionUptime       time.Duration `json:"session_uptime"`
	LastActivity        time.Time     `json:"last_activity"`
	RequestCount        uint64        `json:"request_count"`
	ResponseCount       uint64        `json:"response_count"`
	SessionErrors       uint64        `json:"session_errors"`
	KeepAliveCount      uint64        `json:"keep_alive_count"`
	SequenceNumber      uint16        `json:"sequence_number"`
}

// EncapsulationDiagnostics contains encapsulation layer diagnostic information
type EncapsulationDiagnostics struct {
	ProtocolVersion     uint16    `json:"protocol_version"`
	SupportedCommands   []uint16  `json:"supported_commands"`
	LastCommand         uint16    `json:"last_command"`
	LastCommandTime     time.Time `json:"last_command_time"`
	CommandCounts       map[uint16]uint64 `json:"command_counts"`
	EncapsulationErrors uint64    `json:"encapsulation_errors"`
	MaxPacketSize       uint16    `json:"max_packet_size"`
}

// CIPDiagnostics contains CIP layer diagnostic information
type CIPDiagnostics struct {
	VendorID            uint16            `json:"vendor_id"`
	VendorName          string            `json:"vendor_name"`
	DeviceType          uint16            `json:"device_type"`
	ProductCode         uint16            `json:"product_code"`
	ProductName         string            `json:"product_name"`
	Revision            uint16            `json:"revision"`
	SerialNumber        uint32            `json:"serial_number"`
	DeviceState         uint8             `json:"device_state"`
	SupportedServices   []uint8           `json:"supported_services"`
	ServiceCounts       map[uint8]uint64  `json:"service_counts"`
	CIPErrors           uint64            `json:"cip_errors"`
	LastCIPError        *EtherNetIPError  `json:"last_cip_error"`
	StatusWordCounts    map[uint8]uint64  `json:"status_word_counts"`
}

// NetworkDiagnostics contains network-level diagnostic information
type NetworkDiagnostics struct {
	LocalAddress       string        `json:"local_address"`
	RemoteAddress      string        `json:"remote_address"`
	RemotePort         int           `json:"remote_port"`
	ConnectionType     string        `json:"connection_type"` // TCP/UDP
	TCPState           string        `json:"tcp_state"`
	BytesSent          uint64        `json:"bytes_sent"`
	BytesReceived      uint64        `json:"bytes_received"`
	PacketsSent        uint64        `json:"packets_sent"`
	PacketsReceived    uint64        `json:"packets_received"`
	PacketsLost        uint64        `json:"packets_lost"`
	Retransmissions    uint64        `json:"retransmissions"`
	NetworkLatency     time.Duration `json:"network_latency"`
	LastNetworkError   error         `json:"last_network_error"`
	NetworkErrorCount  uint64        `json:"network_error_count"`
}

// PerformanceDiagnostics contains performance-related diagnostic information
type PerformanceDiagnostics struct {
	AverageRequestTime    time.Duration `json:"average_request_time"`
	MinRequestTime        time.Duration `json:"min_request_time"`
	MaxRequestTime        time.Duration `json:"max_request_time"`
	RequestsPerSecond     float64       `json:"requests_per_second"`
	TagsReadPerSecond     float64       `json:"tags_read_per_second"`
	TagsWrittenPerSecond  float64       `json:"tags_written_per_second"`
	CacheHitRate          float64       `json:"cache_hit_rate"`
	BatchEfficiency       float64       `json:"batch_efficiency"`
	ConnectionPoolSize    int           `json:"connection_pool_size"`
	ActiveConnections     int           `json:"active_connections"`
	QueuedRequests        int           `json:"queued_requests"`
	MemoryUsage           uint64        `json:"memory_usage"`
}

// Enhanced diagnostic methods for the EtherNet/IP handler

// GetEnhancedDiagnostics returns comprehensive diagnostic information
func (e *EtherNetIPHandler) GetEnhancedDiagnostics(device *Device) (*EtherNetIPDiagnostics, error) {
	conn, err := e.getConnection(device)
	if err != nil {
		return nil, err
	}
	
	conn.mutex.RLock()
	defer conn.mutex.RUnlock()
	
	// Get basic diagnostics
	basicDiag, err := e.GetDiagnostics(device)
	if err != nil {
		return nil, err
	}
	
	// Build enhanced diagnostics
	enhanced := &EtherNetIPDiagnostics{
		Diagnostics: basicDiag,
		SessionInfo: &SessionDiagnostics{
			SessionID:          conn.sessionID,
			ConnectionID:       conn.connectionID,
			SessionEstablished: conn.createdAt,
			SessionUptime:      time.Since(conn.createdAt),
			LastActivity:       conn.lastUsed,
			SequenceNumber:     conn.sequenceNumber,
		},
		EncapsulationInfo: &EncapsulationDiagnostics{
			ProtocolVersion: 1, // EtherNet/IP uses version 1
			SupportedCommands: []uint16{
				CIPCommandNOP,
				CIPCommandListServices,
				CIPCommandListIdentity,
				CIPCommandListInterfaces,
				CIPCommandRegisterSession,
				CIPCommandUnregisterSession,
				CIPCommandSendRRData,
				CIPCommandSendUnitData,
			},
		},
		CIPInfo: &CIPDiagnostics{
			VendorID:     conn.vendorID,
			VendorName:   e.getVendorName(conn.vendorID),
			DeviceType:   conn.deviceType,
			ProductCode:  conn.productCode,
			ProductName:  conn.productName,
			Revision:     conn.revision,
			SerialNumber: conn.serialNumber,
			SupportedServices: []uint8{
				CIPServiceGetAttributeSingle,
				CIPServiceSetAttributeSingle,
				CIPServiceGetAll,
				CIPServiceSetAll,
				CIPServiceMultipleServicePacket,
			},
		},
		NetworkInfo: &NetworkDiagnostics{
			RemoteAddress:  device.Address,
			RemotePort:     device.Port,
			ConnectionType: "TCP",
		},
		PerformanceInfo: &PerformanceDiagnostics{
			MinRequestTime: time.Hour, // Initialize to high value
		},
		ErrorCounters: make(map[string]uint64),
	}
	
	return enhanced, nil
}

// DiagnosticCollector collects and aggregates diagnostic information
type DiagnosticCollector struct {
	diagnostics    map[string]*EtherNetIPDiagnostics
	mutex          sync.RWMutex
	collectInterval time.Duration
	stopChan       chan struct{}
	logger         *zap.Logger
}

// NewDiagnosticCollector creates a new diagnostic collector
func NewDiagnosticCollector(logger *zap.Logger) *DiagnosticCollector {
	return &DiagnosticCollector{
		diagnostics:     make(map[string]*EtherNetIPDiagnostics),
		collectInterval: 30 * time.Second,
		stopChan:        make(chan struct{}),
		logger:          logger,
	}
}

// StartCollection begins automatic diagnostic collection
func (dc *DiagnosticCollector) StartCollection(handler *EtherNetIPHandler, devices []*Device) {
	ticker := time.NewTicker(dc.collectInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			dc.collectDiagnostics(handler, devices)
		case <-dc.stopChan:
			return
		}
	}
}

// collectDiagnostics collects diagnostics for all devices
func (dc *DiagnosticCollector) collectDiagnostics(handler *EtherNetIPHandler, devices []*Device) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	
	for _, device := range devices {
		if diag, err := handler.GetEnhancedDiagnostics(device); err == nil {
			dc.diagnostics[device.ID] = diag
		} else {
			dc.logger.Warn("Failed to collect diagnostics", 
				zap.String("device_id", device.ID),
				zap.Error(err),
			)
		}
	}
}

// GetDiagnostics returns diagnostic information for a specific device
func (dc *DiagnosticCollector) GetDiagnostics(deviceID string) (*EtherNetIPDiagnostics, bool) {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	
	diag, exists := dc.diagnostics[deviceID]
	return diag, exists
}

// GetAllDiagnostics returns diagnostic information for all devices
func (dc *DiagnosticCollector) GetAllDiagnostics() map[string]*EtherNetIPDiagnostics {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	
	// Return a copy to avoid race conditions
	result := make(map[string]*EtherNetIPDiagnostics)
	for k, v := range dc.diagnostics {
		result[k] = v
	}
	
	return result
}

// StopCollection stops automatic diagnostic collection
func (dc *DiagnosticCollector) StopCollection() {
	close(dc.stopChan)
}

// Health check functionality

// HealthStatus represents the health status of an EtherNet/IP device
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "HEALTHY"
	HealthStatusWarning   HealthStatus = "WARNING"
	HealthStatusCritical  HealthStatus = "CRITICAL"
	HealthStatusUnknown   HealthStatus = "UNKNOWN"
)

// HealthCheck performs a comprehensive health check on a device
func (e *EtherNetIPHandler) HealthCheck(device *Device) (*HealthCheckResult, error) {
	result := &HealthCheckResult{
		DeviceID:    device.ID,
		Timestamp:   time.Now(),
		Status:      HealthStatusHealthy,
		Checks:      make(map[string]*CheckResult),
	}
	
	// Connectivity check
	if err := e.Ping(device); err != nil {
		result.Checks["connectivity"] = &CheckResult{
			Name:    "Connectivity",
			Status:  HealthStatusCritical,
			Message: err.Error(),
		}
		result.Status = HealthStatusCritical
	} else {
		result.Checks["connectivity"] = &CheckResult{
			Name:    "Connectivity",
			Status:  HealthStatusHealthy,
			Message: "Device is reachable",
		}
	}
	
	// Session check
	if conn, err := e.getConnection(device); err == nil {
		if time.Since(conn.lastUsed) > 5*time.Minute {
			result.Checks["session"] = &CheckResult{
				Name:    "Session Activity",
				Status:  HealthStatusWarning,
				Message: "No recent activity",
			}
			if result.Status == HealthStatusHealthy {
				result.Status = HealthStatusWarning
			}
		} else {
			result.Checks["session"] = &CheckResult{
				Name:    "Session Activity",
				Status:  HealthStatusHealthy,
				Message: "Session is active",
			}
		}
	}
	
	// Performance check
	if diag, err := e.GetEnhancedDiagnostics(device); err == nil {
		if diag.PerformanceInfo.AverageRequestTime > 1*time.Second {
			result.Checks["performance"] = &CheckResult{
				Name:    "Performance",
				Status:  HealthStatusWarning,
				Message: "High response times detected",
			}
			if result.Status == HealthStatusHealthy {
				result.Status = HealthStatusWarning
			}
		} else {
			result.Checks["performance"] = &CheckResult{
				Name:    "Performance",
				Status:  HealthStatusHealthy,
				Message: "Response times normal",
			}
		}
	}
	
	return result, nil
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	DeviceID  string                    `json:"device_id"`
	Timestamp time.Time                 `json:"timestamp"`
	Status    HealthStatus              `json:"status"`
	Checks    map[string]*CheckResult   `json:"checks"`
}

// CheckResult represents the result of an individual check
type CheckResult struct {
	Name     string       `json:"name"`
	Status   HealthStatus `json:"status"`
	Message  string       `json:"message"`
	Duration time.Duration `json:"duration,omitempty"`
}