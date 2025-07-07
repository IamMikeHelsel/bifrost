package protocols

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
	
	"go.uber.org/zap"
)

// CIP Session Management and Protocol Implementation

// registerSession registers a new CIP session with the device
func (e *EtherNetIPHandler) registerSession(conn *EtherNetIPConnection) (uint32, error) {
	// Build Register Session request
	header := CIPEncapsulationHeader{
		Command:       CIPCommandRegisterSession,
		Length:        4,
		SessionHandle: 0,
		Status:        0,
		Options:       0,
	}
	
	// Register Session data (Protocol Version = 1, Options = 0)
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data[0:2], 1) // Protocol Version
	binary.LittleEndian.PutUint16(data[2:4], 0) // Options
	
	// Send request
	if err := e.sendEncapsulationRequest(conn, &header, data); err != nil {
		return 0, err
	}
	
	// Read response
	respHeader, respData, err := e.readEncapsulationResponse(conn)
	if err != nil {
		return 0, err
	}
	
	if respHeader.Status != 0 {
		return 0, fmt.Errorf("register session failed with status: 0x%08X", respHeader.Status)
	}
	
	if len(respData) < 4 {
		return 0, fmt.Errorf("invalid register session response")
	}
	
	sessionID := respHeader.SessionHandle
	
	e.handler.logger.Debug("CIP session registered",
		zap.String("device_id", conn.deviceID),
		zap.Uint32("session_id", sessionID),
	)
	
	return sessionID, nil
}

// unregisterSession unregisters a CIP session
func (e *EtherNetIPHandler) unregisterSession(conn *EtherNetIPConnection) error {
	header := CIPEncapsulationHeader{
		Command:       CIPCommandUnregisterSession,
		Length:        0,
		SessionHandle: conn.sessionID,
		Status:        0,
		Options:       0,
	}
	
	err := e.sendEncapsulationRequest(conn, &header, nil)
	if err != nil {
		e.logger.Warn("Failed to unregister session", zap.Error(err))
	}
	
	return err
}

// getDeviceIdentity retrieves device identity information
func (e *EtherNetIPHandler) getDeviceIdentity(conn *EtherNetIPConnection) (*CIPIdentityObject, error) {
	// Build request to read Identity Object attributes
	request := &CIPRequest{
		Service:     CIPServiceGetAttributeSingle,
		RequestPath: e.buildInstancePath(CIPClassIdentity, 1, 1), // Identity Object Instance 1, Attribute 1
		RequestData: []byte{},
	}
	
	response, err := e.sendCIPRequest(conn, request)
	if err != nil {
		return nil, err
	}
	
	if response.GeneralStatus != 0 {
		return nil, fmt.Errorf("identity read failed with status: 0x%02X", response.GeneralStatus)
	}
	
	return e.parseIdentityObject(response.ResponseData)
}

// sendCIPRequest sends a CIP request and returns the response
func (e *EtherNetIPHandler) sendCIPRequest(conn *EtherNetIPConnection, request *CIPRequest) (*CIPResponse, error) {
	// Build CIP request packet
	requestData := e.buildCIPRequestData(request)
	
	// Build Common Packet Format
	cpf := &CIPCommonPacketFormat{
		ItemCount: 2,
		TypeID:    0x0000, // NULL Address Type
		Length:    0,
		Data:      requestData,
	}
	
	cpfData := e.buildCPFData(cpf)
	
	// Build encapsulation header
	header := CIPEncapsulationHeader{
		Command:       CIPCommandSendRRData,
		Length:        uint16(len(cpfData)),
		SessionHandle: conn.sessionID,
		Status:        0,
		Options:       0,
	}
	
	// Send request
	if err := e.sendEncapsulationRequest(conn, &header, cpfData); err != nil {
		return nil, err
	}
	
	// Read response
	respHeader, respData, err := e.readEncapsulationResponse(conn)
	if err != nil {
		return nil, err
	}
	
	if respHeader.Status != 0 {
		return nil, fmt.Errorf("CIP request failed with status: 0x%08X", respHeader.Status)
	}
	
	// Parse CIP response
	return e.parseCIPResponse(respData)
}

// sendEncapsulationRequest sends an encapsulation request
func (e *EtherNetIPHandler) sendEncapsulationRequest(conn *EtherNetIPConnection, header *CIPEncapsulationHeader, data []byte) error {
	// Set connection timeout
	conn.tcpConn.SetWriteDeadline(time.Now().Add(e.config.DefaultTimeout))
	
	// Send header
	if err := e.sendEncapsulationHeader(conn, header); err != nil {
		return err
	}
	
	// Send data if present
	if len(data) > 0 {
		_, err := conn.tcpConn.Write(data)
		if err != nil {
			return fmt.Errorf("failed to send encapsulation data: %w", err)
		}
	}
	
	return nil
}

// readEncapsulationResponse reads an encapsulation response
func (e *EtherNetIPHandler) readEncapsulationResponse(conn *EtherNetIPConnection) (*CIPEncapsulationHeader, []byte, error) {
	// Set connection timeout
	conn.tcpConn.SetReadDeadline(time.Now().Add(e.config.DefaultTimeout))
	
	// Read header
	header, err := e.readEncapsulationHeader(conn)
	if err != nil {
		return nil, nil, err
	}
	
	// Read data if present
	var data []byte
	if header.Length > 0 {
		data = make([]byte, header.Length)
		_, err := conn.tcpConn.Read(data)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read encapsulation data: %w", err)
		}
	}
	
	return header, data, nil
}

// sendEncapsulationHeader sends an encapsulation header
func (e *EtherNetIPHandler) sendEncapsulationHeader(conn *EtherNetIPConnection, header *CIPEncapsulationHeader) error {
	buf := make([]byte, 24) // Encapsulation header size
	
	binary.LittleEndian.PutUint16(buf[0:2], header.Command)
	binary.LittleEndian.PutUint16(buf[2:4], header.Length)
	binary.LittleEndian.PutUint32(buf[4:8], header.SessionHandle)
	binary.LittleEndian.PutUint32(buf[8:12], header.Status)
	copy(buf[12:20], header.Context[:])
	binary.LittleEndian.PutUint32(buf[20:24], header.Options)
	
	_, err := conn.tcpConn.Write(buf)
	if err != nil {
		return fmt.Errorf("failed to send encapsulation header: %w", err)
	}
	
	return nil
}

// readEncapsulationHeader reads an encapsulation header
func (e *EtherNetIPHandler) readEncapsulationHeader(conn *EtherNetIPConnection) (*CIPEncapsulationHeader, error) {
	buf := make([]byte, 24)
	_, err := conn.tcpConn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read encapsulation header: %w", err)
	}
	
	header := &CIPEncapsulationHeader{
		Command:       binary.LittleEndian.Uint16(buf[0:2]),
		Length:        binary.LittleEndian.Uint16(buf[2:4]),
		SessionHandle: binary.LittleEndian.Uint32(buf[4:8]),
		Status:        binary.LittleEndian.Uint32(buf[8:12]),
		Options:       binary.LittleEndian.Uint32(buf[20:24]),
	}
	
	copy(header.Context[:], buf[12:20])
	
	return header, nil
}

// Address parsing and path building

// parseAddress parses EtherNet/IP tag addresses
func (e *EtherNetIPHandler) parseAddress(address string) (*EtherNetIPAddress, error) {
	address = strings.TrimSpace(address)
	
	if address == "" {
		return nil, fmt.Errorf("empty address")
	}
	
	addr := &EtherNetIPAddress{
		AttributeID: 1, // Default attribute ID
		DataType:    CIPDataTypeDint, // Default data type
	}
	
	// Check if it's a symbolic address (tag name)
	if !strings.Contains(address, "@") && !strings.Contains(address, ".") {
		// Simple tag name
		addr.TagName = address
		addr.IsSymbolic = true
		return addr, nil
	}
	
	// Parse instance-based addressing: Class@Instance.Attribute
	if strings.Contains(address, "@") {
		parts := strings.Split(address, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid instance address format")
		}
		
		instanceStr := parts[1]
		if strings.Contains(instanceStr, ".") {
			subParts := strings.Split(instanceStr, ".")
			instanceStr = subParts[0]
			
			attrID, err := strconv.ParseUint(subParts[1], 10, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid attribute ID: %w", err)
			}
			addr.AttributeID = uint8(attrID)
		}
		
		instanceID, err := strconv.ParseUint(instanceStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid instance ID: %w", err)
		}
		addr.InstanceID = uint32(instanceID)
		addr.IsSymbolic = false
		return addr, nil
	}
	
	// Parse symbolic addressing with array indices: TagName[index]
	if strings.Contains(address, "[") && strings.Contains(address, "]") {
		openBracket := strings.Index(address, "[")
		closeBracket := strings.Index(address, "]")
		
		if closeBracket <= openBracket {
			return nil, fmt.Errorf("invalid array address format")
		}
		
		addr.TagName = address[:openBracket]
		indexStr := address[openBracket+1:closeBracket]
		
		elementIndex, err := strconv.ParseUint(indexStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid array index: %w", err)
		}
		
		addr.ElementIndex = uint32(elementIndex)
		addr.IsSymbolic = true
		addr.IsArray = true
		return addr, nil
	}
	
	// Default to symbolic addressing
	addr.TagName = address
	addr.IsSymbolic = true
	return addr, nil
}

// buildSymbolicPath builds a CIP request path for symbolic addressing
func (e *EtherNetIPHandler) buildSymbolicPath(tagName string) []byte {
	// Build ANSI Extended Symbolic Segment
	tagBytes := []byte(tagName)
	pathLen := len(tagBytes)
	
	// Calculate padded length (must be even)
	paddedLen := pathLen
	if paddedLen%2 != 0 {
		paddedLen++
	}
	
	path := make([]byte, 2+paddedLen)
	path[0] = 0x91 // ANSI Extended Symbolic Segment
	path[1] = byte(pathLen)
	copy(path[2:], tagBytes)
	
	return path
}

// buildInstancePath builds a CIP request path for instance-based addressing
func (e *EtherNetIPHandler) buildInstancePath(classID uint16, instanceID uint32, attributeID uint8) []byte {
	var path []byte
	
	// Class ID (8-bit or 16-bit)
	if classID <= 0xFF {
		path = append(path, 0x20, byte(classID))
	} else {
		path = append(path, 0x21, 0x00)
		path = append(path, byte(classID&0xFF), byte(classID>>8))
	}
	
	// Instance ID (8-bit or 16-bit)
	if instanceID <= 0xFF {
		path = append(path, 0x24, byte(instanceID))
	} else {
		path = append(path, 0x25, 0x00)
		path = append(path, byte(instanceID&0xFF), byte(instanceID>>8))
	}
	
	// Attribute ID
	path = append(path, 0x30, attributeID)
	
	return path
}

// buildCIPRequestData builds CIP request data
func (e *EtherNetIPHandler) buildCIPRequestData(request *CIPRequest) []byte {
	var data []byte
	
	// Service code
	data = append(data, request.Service)
	
	// Request path size (in words)
	pathSize := len(request.RequestPath) / 2
	if len(request.RequestPath)%2 != 0 {
		pathSize++
	}
	data = append(data, byte(pathSize))
	
	// Request path
	data = append(data, request.RequestPath...)
	
	// Pad to even length
	if len(request.RequestPath)%2 != 0 {
		data = append(data, 0x00)
	}
	
	// Request data
	data = append(data, request.RequestData...)
	
	return data
}

// buildCPFData builds Common Packet Format data
func (e *EtherNetIPHandler) buildCPFData(cpf *CIPCommonPacketFormat) []byte {
	var data []byte
	
	// Item count
	binary.LittleEndian.PutUint16(data[0:2], cpf.ItemCount)
	data = append(data, make([]byte, 2)...)
	binary.LittleEndian.PutUint16(data[0:2], cpf.ItemCount)
	
	// Address item
	data = append(data, make([]byte, 4)...)
	binary.LittleEndian.PutUint16(data[len(data)-4:], 0x0000) // NULL Address Type
	binary.LittleEndian.PutUint16(data[len(data)-2:], 0x0000) // Length = 0
	
	// Data item
	data = append(data, make([]byte, 4)...)
	binary.LittleEndian.PutUint16(data[len(data)-4:], 0x00B2) // Unconnected Data Item
	binary.LittleEndian.PutUint16(data[len(data)-2:], uint16(len(cpf.Data)))
	
	// Data
	data = append(data, cpf.Data...)
	
	return data
}

// parseCIPResponse parses a CIP response
func (e *EtherNetIPHandler) parseCIPResponse(data []byte) (*CIPResponse, error) {
	if len(data) < 6 {
		return nil, fmt.Errorf("response too short")
	}
	
	// Skip CPF header (6 bytes minimum)
	offset := 6
	
	// Find data item
	itemCount := binary.LittleEndian.Uint16(data[0:2])
	offset = 2
	
	for i := 0; i < int(itemCount); i++ {
		if offset+4 > len(data) {
			return nil, fmt.Errorf("invalid CPF format")
		}
		
		typeID := binary.LittleEndian.Uint16(data[offset:offset+2])
		length := binary.LittleEndian.Uint16(data[offset+2:offset+4])
		offset += 4
		
		if typeID == 0x00B2 { // Unconnected Data Item
			if offset+int(length) > len(data) {
				return nil, fmt.Errorf("invalid data item length")
			}
			
			responseData := data[offset:offset+int(length)]
			return e.parseCIPResponseData(responseData)
		}
		
		offset += int(length)
	}
	
	return nil, fmt.Errorf("no data item found in response")
}

// parseCIPResponseData parses CIP response data
func (e *EtherNetIPHandler) parseCIPResponseData(data []byte) (*CIPResponse, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("response data too short")
	}
	
	response := &CIPResponse{
		Service:       data[0],
		GeneralStatus: data[1],
	}
	
	offset := 2
	
	// Check for extended status
	if len(data) > offset {
		extStatusSize := int(data[offset])
		offset++
		
		if len(data) >= offset+extStatusSize*2 {
			response.ExtendedStatus = data[offset:offset+extStatusSize*2]
			offset += extStatusSize * 2
		}
	}
	
	// Remaining data is response data
	if len(data) > offset {
		response.ResponseData = data[offset:]
	}
	
	return response, nil
}

// parseIdentityObject parses Identity Object data
func (e *EtherNetIPHandler) parseIdentityObject(data []byte) (*CIPIdentityObject, error) {
	if len(data) < 24 {
		return nil, fmt.Errorf("identity object data too short")
	}
	
	identity := &CIPIdentityObject{
		VendorID:     binary.LittleEndian.Uint16(data[0:2]),
		DeviceType:   binary.LittleEndian.Uint16(data[2:4]),
		ProductCode:  binary.LittleEndian.Uint16(data[4:6]),
		Revision:     binary.LittleEndian.Uint16(data[6:8]),
		Status:       binary.LittleEndian.Uint16(data[8:10]),
		SerialNumber: binary.LittleEndian.Uint32(data[10:14]),
		State:        data[23],
	}
	
	// Parse product name (starts at offset 14)
	if len(data) > 14 {
		nameLen := int(data[14])
		if len(data) >= 15+nameLen {
			identity.ProductName = string(data[15:15+nameLen])
		}
	}
	
	return identity, nil
}

// Data conversion methods

// convertFromCIP converts CIP data to Go types
func (e *EtherNetIPHandler) convertFromCIP(data []byte, dataType uint8) (interface{}, error) {
	switch dataType {
	case CIPDataTypeBool:
		if len(data) < 1 {
			return nil, fmt.Errorf("insufficient data for bool")
		}
		return data[0] != 0, nil
		
	case CIPDataTypeSint:
		if len(data) < 1 {
			return nil, fmt.Errorf("insufficient data for sint")
		}
		return int8(data[0]), nil
		
	case CIPDataTypeInt:
		if len(data) < 2 {
			return nil, fmt.Errorf("insufficient data for int")
		}
		return int16(binary.LittleEndian.Uint16(data)), nil
		
	case CIPDataTypeDint:
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for dint")
		}
		return int32(binary.LittleEndian.Uint32(data)), nil
		
	case CIPDataTypeLint:
		if len(data) < 8 {
			return nil, fmt.Errorf("insufficient data for lint")
		}
		return int64(binary.LittleEndian.Uint64(data)), nil
		
	case CIPDataTypeUsint:
		if len(data) < 1 {
			return nil, fmt.Errorf("insufficient data for usint")
		}
		return uint8(data[0]), nil
		
	case CIPDataTypeUint:
		if len(data) < 2 {
			return nil, fmt.Errorf("insufficient data for uint")
		}
		return binary.LittleEndian.Uint16(data), nil
		
	case CIPDataTypeUdint:
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for udint")
		}
		return binary.LittleEndian.Uint32(data), nil
		
	case CIPDataTypeUlint:
		if len(data) < 8 {
			return nil, fmt.Errorf("insufficient data for ulint")
		}
		return binary.LittleEndian.Uint64(data), nil
		
	case CIPDataTypeReal:
		if len(data) < 4 {
			return nil, fmt.Errorf("insufficient data for real")
		}
		bits := binary.LittleEndian.Uint32(data)
		return float32(bits), nil
		
	case CIPDataTypeLreal:
		if len(data) < 8 {
			return nil, fmt.Errorf("insufficient data for lreal")
		}
		bits := binary.LittleEndian.Uint64(data)
		return float64(bits), nil
		
	case CIPDataTypeString:
		if len(data) < 2 {
			return nil, fmt.Errorf("insufficient data for string")
		}
		strLen := binary.LittleEndian.Uint16(data[0:2])
		if len(data) < int(strLen)+2 {
			return nil, fmt.Errorf("insufficient data for string content")
		}
		return string(data[2:2+strLen]), nil
		
	default:
		return nil, fmt.Errorf("unsupported CIP data type: 0x%02X", dataType)
	}
}

// convertToCIP converts Go types to CIP data
func (e *EtherNetIPHandler) convertToCIP(value interface{}, dataType uint8) ([]byte, error) {
	switch dataType {
	case CIPDataTypeBool:
		if boolVal, ok := value.(bool); ok {
			if boolVal {
				return []byte{0x01}, nil
			}
			return []byte{0x00}, nil
		}
		return nil, fmt.Errorf("expected boolean value")
		
	case CIPDataTypeDint:
		var intVal int32
		switch v := value.(type) {
		case int:
			intVal = int32(v)
		case int32:
			intVal = v
		case int64:
			intVal = int32(v)
		default:
			return nil, fmt.Errorf("expected integer value")
		}
		
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(intVal))
		return buf, nil
		
	case CIPDataTypeReal:
		var floatVal float32
		switch v := value.(type) {
		case float32:
			floatVal = v
		case float64:
			floatVal = float32(v)
		default:
			return nil, fmt.Errorf("expected float value")
		}
		
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(floatVal))
		return buf, nil
		
	case CIPDataTypeString:
		strVal, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string value")
		}
		
		strBytes := []byte(strVal)
		buf := make([]byte, 2+len(strBytes))
		binary.LittleEndian.PutUint16(buf[0:2], uint16(len(strBytes)))
		copy(buf[2:], strBytes)
		return buf, nil
		
	default:
		return nil, fmt.Errorf("unsupported CIP data type for conversion: 0x%02X", dataType)
	}
}

// getVendorName returns vendor name from vendor ID
func (e *EtherNetIPHandler) getVendorName(vendorID uint16) string {
	switch vendorID {
	case 0x0001:
		return "Rockwell Automation/Allen-Bradley"
	case 0x0002:
		return "Schneider Electric"
	case 0x0003:
		return "Siemens"
	case 0x0004:
		return "GE Fanuc"
	case 0x0005:
		return "Omron"
	case 0x0006:
		return "Mitsubishi Electric"
	case 0x0007:
		return "Honeywell"
	case 0x0008:
		return "Yokogawa"
	case 0x0009:
		return "Emerson"
	case 0x000A:
		return "ABB"
	default:
		return fmt.Sprintf("Unknown (0x%04X)", vendorID)
	}
}

// Additional helper methods for batch operations and device discovery

// groupTagsForBatchRead groups tags for efficient batch reading
func (e *EtherNetIPHandler) groupTagsForBatchRead(tags []*Tag, maxBatchSize int) [][]*Tag {
	var batches [][]*Tag
	
	for i := 0; i < len(tags); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(tags) {
			end = len(tags)
		}
		batches = append(batches, tags[i:end])
	}
	
	return batches
}

// readTagBatch reads a batch of tags using Multiple Service Packet
func (e *EtherNetIPHandler) readTagBatch(conn *EtherNetIPConnection, tags []*Tag) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	
	// For now, fall back to individual reads
	// Full Multiple Service Packet implementation would be more complex
	for _, tag := range tags {
		if value, err := e.readSingleTag(conn, tag); err == nil {
			results[tag.ID] = value
		}
	}
	
	return results, nil
}

// readSingleTag reads a single tag (helper method)
func (e *EtherNetIPHandler) readSingleTag(conn *EtherNetIPConnection, tag *Tag) (interface{}, error) {
	addr, err := e.parseAddress(tag.Address)
	if err != nil {
		return nil, err
	}
	
	var request *CIPRequest
	if addr.IsSymbolic {
		request = &CIPRequest{
			Service:     CIPServiceGetAttributeSingle,
			RequestPath: e.buildSymbolicPath(addr.TagName),
			RequestData: []byte{},
		}
	} else {
		request = &CIPRequest{
			Service:     CIPServiceGetAttributeSingle,
			RequestPath: e.buildInstancePath(CIPClassSymbol, addr.InstanceID, addr.AttributeID),
			RequestData: []byte{},
		}
	}
	
	response, err := e.sendCIPRequest(conn, request)
	if err != nil {
		return nil, err
	}
	
	if response.GeneralStatus != 0 {
		return nil, fmt.Errorf("CIP error: status 0x%02X", response.GeneralStatus)
	}
	
	return e.convertFromCIP(response.ResponseData, addr.DataType)
}

// probeEtherNetIPDevice probes for EtherNet/IP devices
func (e *EtherNetIPHandler) probeEtherNetIPDevice(ctx context.Context, ip string, port int) *Device {
	timeout := 2 * time.Second
	
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), timeout)
	if err != nil {
		return nil
	}
	defer conn.Close()
	
	// Try to establish a basic EtherNet/IP connection
	tempConn := &EtherNetIPConnection{
		tcpConn:  conn,
		deviceID: fmt.Sprintf("temp-%s-%d", ip, port),
	}
	
	// Try to register a session
	sessionID, err := e.registerSession(tempConn)
	if err != nil {
		return nil
	}
	
	tempConn.sessionID = sessionID
	
	// Try to get device identity
	identity, err := e.getDeviceIdentity(tempConn)
	if err != nil {
		// Still create device even if identity fails
		identity = &CIPIdentityObject{
			ProductName: "EtherNet/IP Device",
		}
	}
	
	// Unregister session
	e.unregisterSession(tempConn)
	
	return &Device{
		ID:       fmt.Sprintf("ethernetip-%s-%d", ip, port),
		Name:     fmt.Sprintf("%s at %s:%d", identity.ProductName, ip, port),
		Protocol: "ethernet-ip",
		Address:  ip,
		Port:     port,
		Config:   make(map[string]interface{}),
	}
}