
package protocols

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
)

// OPCUAHandler handles OPC-UA communication with high performance and security support.
type OPCUAHandler struct {
	logger      *zap.Logger
	clients     map[string]*opcua.Client
	connections map[string]*OPCUAConnection
	mutex       sync.RWMutex

	// Performance optimization
	maxConcurrentReads int
	batchSize          int
	readTimeout        time.Duration
}

// OPCUAConnection holds connection state and configuration
type OPCUAConnection struct {
	client      *opcua.Client
	endpoint    string
	securityPolicy string
	securityMode ua.MessageSecurityMode
	authPolicy  ua.UserTokenType
	username    string
	password    string
	certificate []byte
	privateKey  *rsa.PrivateKey
	connected   bool
	lastSeen    time.Time
	
	// Subscription management
	subscriptions map[string]*OPCUASubscription
	subMutex      sync.RWMutex
}

// OPCUASubscription manages real-time data subscriptions
type OPCUASubscription struct {
	id            uint32
	publishingInterval time.Duration
	monitoredItems map[string]*ua.MonitoredItemCreateRequest
	callback      func(values map[string]interface{})
	active        bool
}

// OPCUAConfig holds OPC UA specific configuration
type OPCUAConfig struct {
	SecurityPolicy      string        `json:"security_policy"`
	SecurityMode        string        `json:"security_mode"`
	AuthPolicy          string        `json:"auth_policy"`
	Username            string        `json:"username"`
	Password            string        `json:"password"`
	CertificatePath     string        `json:"certificate_path"`
	PrivateKeyPath      string        `json:"private_key_path"`
	SessionTimeout      time.Duration `json:"session_timeout"`
	MaxConcurrentReads  int           `json:"max_concurrent_reads"`
	BatchSize           int           `json:"batch_size"`
	ReadTimeout         time.Duration `json:"read_timeout"`
}

// NewOPCUAHandler creates a new high-performance OPC-UA handler.
func NewOPCUAHandler(logger *zap.Logger) *OPCUAHandler {
	return &OPCUAHandler{
		logger:      logger,
		clients:     make(map[string]*opcua.Client),
		connections: make(map[string]*OPCUAConnection),
		
		// Performance defaults optimized for industrial use
		maxConcurrentReads: 100,
		batchSize:          1000, // Read up to 1000 values in one batch
		readTimeout:        time.Second * 5,
	}
}

// Connect establishes a secure connection to an OPC-UA server with all security policies support.
func (h *OPCUAHandler) Connect(device *Device) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.logger.Info("Connecting to OPC-UA server", 
		zap.String("device_id", device.ID),
		zap.String("address", device.Address),
		zap.Int("port", device.Port))

	// Parse OPC UA configuration from device config
	config := h.parseOPCUAConfig(device.Config)
	
	// Build endpoint URL
	endpoint := h.buildEndpointURL(device.Address, device.Port)
	
	// Create client with security configuration
	client, err := h.createSecureClient(endpoint, config)
	if err != nil {
		return fmt.Errorf("failed to create OPC UA client: %w", err)
	}

	// Connect to server
	if err := client.Connect(context.Background()); err != nil {
		return fmt.Errorf("failed to connect to OPC UA server: %w", err)
	}

	// Store connection state
	connection := &OPCUAConnection{
		client:        client,
		endpoint:      endpoint,
		securityPolicy: config.SecurityPolicy,
		securityMode:  h.parseSecurityMode(config.SecurityMode),
		authPolicy:    h.parseAuthPolicy(config.AuthPolicy),
		username:      config.Username,
		password:      config.Password,
		connected:     true,
		lastSeen:      time.Now(),
		subscriptions: make(map[string]*OPCUASubscription),
	}

	h.connections[device.ID] = connection
	h.clients[device.ID] = client

	h.logger.Info("Successfully connected to OPC-UA server", 
		zap.String("device_id", device.ID),
		zap.String("endpoint", endpoint))

	return nil
}

// Disconnect closes the connection to the OPC-UA server.
func (h *OPCUAHandler) Disconnect(device *Device) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.logger.Info("Disconnecting from OPC-UA server", zap.String("device_id", device.ID))

	connection, exists := h.connections[device.ID]
	if !exists {
		return fmt.Errorf("device %s not connected", device.ID)
	}

	// Close all subscriptions first
	for subID := range connection.subscriptions {
		if err := h.closeSubscription(device.ID, subID); err != nil {
			h.logger.Warn("Failed to close subscription", 
				zap.String("device_id", device.ID),
				zap.String("subscription_id", subID),
				zap.Error(err))
		}
	}

	// Close client connection
	if err := connection.client.Close(context.Background()); err != nil {
		h.logger.Warn("Error closing OPC UA connection", 
			zap.String("device_id", device.ID),
			zap.Error(err))
	}

	// Clean up connection state
	delete(h.connections, device.ID)
	delete(h.clients, device.ID)

	h.logger.Info("Disconnected from OPC-UA server", zap.String("device_id", device.ID))
	return nil
}

// IsConnected checks if the device is connected.
func (h *OPCUAHandler) IsConnected(device *Device) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	connection, exists := h.connections[device.ID]
	return exists && connection.connected
}

// ReadTag reads a single tag from an OPC-UA device with optimized performance.
func (h *OPCUAHandler) ReadTag(device *Device, tag *Tag) (interface{}, error) {
	connection, err := h.getConnection(device.ID)
	if err != nil {
		return nil, err
	}

	// Parse node ID from tag address
	nodeID, err := ua.ParseNodeID(tag.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid node ID %s: %w", tag.Address, err)
	}

	h.logger.Debug("Reading OPC-UA node", 
		zap.String("device_id", device.ID),
		zap.String("node_id", tag.Address))

	// Create read request
	request := &ua.ReadRequest{
		MaxAge: 0, // Get fresh values
		NodesToRead: []*ua.ReadValueID{
			{
				NodeID:      nodeID,
				AttributeID: ua.AttributeIDValue,
			},
		},
	}

	// Perform read with timeout
	ctx, cancel := context.WithTimeout(context.Background(), h.readTimeout)
	defer cancel()

	response, err := connection.client.Read(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to read node %s: %w", tag.Address, err)
	}

	if len(response.Results) == 0 {
		return nil, fmt.Errorf("no results returned for node %s", tag.Address)
	}

	result := response.Results[0]
	if result.Status != ua.StatusOK {
		return nil, fmt.Errorf("bad status for node %s: %s", tag.Address, result.Status)
	}

	// Update tag metadata
	tag.Quality = h.mapStatusToQuality(result.Status)
	tag.Timestamp = time.Now()

	// Convert value to appropriate Go type
	value := h.convertVariantValue(result.Value)
	tag.Value = value

	connection.lastSeen = time.Now()

	h.logger.Debug("Successfully read OPC-UA node", 
		zap.String("device_id", device.ID),
		zap.String("node_id", tag.Address),
		zap.Any("value", value))

	return value, nil
}

// WriteTag writes a single tag to an OPC-UA device.
func (h *OPCUAHandler) WriteTag(device *Device, tag *Tag, value interface{}) error {
	connection, err := h.getConnection(device.ID)
	if err != nil {
		return err
	}

	// Parse node ID from tag address
	nodeID, err := ua.ParseNodeID(tag.Address)
	if err != nil {
		return fmt.Errorf("invalid node ID %s: %w", tag.Address, err)
	}

	h.logger.Debug("Writing to OPC-UA node", 
		zap.String("device_id", device.ID),
		zap.String("node_id", tag.Address),
		zap.Any("value", value))

	// Convert value to variant
	variant, err := h.convertToVariant(value, tag.DataType)
	if err != nil {
		return fmt.Errorf("failed to convert value: %w", err)
	}

	// Create write request
	request := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      nodeID,
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					Value: variant,
				},
			},
		},
	}

	// Perform write with timeout
	ctx, cancel := context.WithTimeout(context.Background(), h.readTimeout)
	defer cancel()

	response, err := connection.client.Write(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to write node %s: %w", tag.Address, err)
	}

	if len(response.Results) == 0 {
		return fmt.Errorf("no results returned for write to node %s", tag.Address)
	}

	if response.Results[0] != ua.StatusOK {
		return fmt.Errorf("bad status for write to node %s: %s", tag.Address, response.Results[0])
	}

	connection.lastSeen = time.Now()

	h.logger.Debug("Successfully wrote to OPC-UA node", 
		zap.String("device_id", device.ID),
		zap.String("node_id", tag.Address),
		zap.Any("value", value))

	return nil
}

// ReadMultipleTags reads multiple tags from an OPC-UA device with bulk optimizations.
// Targets: Read 1,000 values in < 100ms
func (h *OPCUAHandler) ReadMultipleTags(device *Device, tags []*Tag) (map[string]interface{}, error) {
	connection, err := h.getConnection(device.ID)
	if err != nil {
		return nil, err
	}

	h.logger.Debug("Reading multiple tags from OPC-UA device", 
		zap.String("device_id", device.ID),
		zap.Int("tag_count", len(tags)))

	startTime := time.Now()
	results := make(map[string]interface{})

	// Process tags in batches for optimal performance
	for i := 0; i < len(tags); i += h.batchSize {
		end := i + h.batchSize
		if end > len(tags) {
			end = len(tags)
		}

		batchTags := tags[i:end]
		batchResults, err := h.readTagBatch(connection, batchTags)
		if err != nil {
			return nil, fmt.Errorf("failed to read batch: %w", err)
		}

		// Merge batch results
		for addr, value := range batchResults {
			results[addr] = value
		}
	}

	duration := time.Since(startTime)
	connection.lastSeen = time.Now()

	h.logger.Info("Completed bulk read operation", 
		zap.String("device_id", device.ID),
		zap.Int("tag_count", len(tags)),
		zap.Duration("duration", duration),
		zap.Float64("tags_per_second", float64(len(tags))/duration.Seconds()))

	return results, nil
}

// BrowseNodes browses OPC UA nodes with high performance for large namespaces.
// Target: Browse 10,000 nodes in < 1 second
func (h *OPCUAHandler) BrowseNodes(device *Device, nodeID string, maxDepth int) ([]*BrowseResult, error) {
	connection, err := h.getConnection(device.ID)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	
	h.logger.Info("Starting OPC-UA browse operation", 
		zap.String("device_id", device.ID),
		zap.String("root_node", nodeID),
		zap.Int("max_depth", maxDepth))

	var rootNodeID *ua.NodeID
	if nodeID == "" {
		rootNodeID = ua.NewNumericNodeID(0, id.ObjectsFolder)
	} else {
		rootNodeID, err = ua.ParseNodeID(nodeID)
		if err != nil {
			return nil, fmt.Errorf("invalid root node ID %s: %w", nodeID, err)
		}
	}

	results, err := h.browseRecursive(connection, rootNodeID, 0, maxDepth)
	if err != nil {
		return nil, fmt.Errorf("browse failed: %w", err)
	}

	duration := time.Since(startTime)
	h.logger.Info("Completed browse operation", 
		zap.String("device_id", device.ID),
		zap.Int("nodes_found", len(results)),
		zap.Duration("duration", duration),
		zap.Float64("nodes_per_second", float64(len(results))/duration.Seconds()))

	return results, nil
}

// CreateSubscription creates a real-time subscription for monitoring data changes.
// Target: Subscription updates in < 10ms latency
func (h *OPCUAHandler) CreateSubscription(device *Device, tags []*Tag, publishingInterval time.Duration, callback func(map[string]interface{})) (string, error) {
	connection, err := h.getConnection(device.ID)
	if err != nil {
		return "", err
	}

	h.logger.Info("Creating OPC-UA subscription", 
		zap.String("device_id", device.ID),
		zap.Int("tag_count", len(tags)),
		zap.Duration("interval", publishingInterval))

	// Create notification channel
	notifCh := make(chan *opcua.PublishNotificationData, 100)

	// Create subscription parameters
	params := &opcua.SubscriptionParameters{
		Interval: publishingInterval,
		Priority: 1,
		MaxNotificationsPerPublish: 1000,
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.readTimeout)
	defer cancel()

	// Create subscription using the new API
	sub, err := connection.client.Subscribe(ctx, params, notifCh)
	if err != nil {
		return "", fmt.Errorf("failed to create subscription: %w", err)
	}

	subscriptionID := fmt.Sprintf("sub_%d", sub.SubscriptionID)

	// Create monitored items for all tags
	var monitoredItems []*ua.MonitoredItemCreateRequest
	for i, tag := range tags {
		nodeID, err := ua.ParseNodeID(tag.Address)
		if err != nil {
			h.logger.Warn("Invalid node ID for subscription", 
				zap.String("node_id", tag.Address),
				zap.Error(err))
			continue
		}

		monitoredItems = append(monitoredItems, &ua.MonitoredItemCreateRequest{
			ItemToMonitor: &ua.ReadValueID{
				NodeID:      nodeID,
				AttributeID: ua.AttributeIDValue,
			},
			MonitoringMode: ua.MonitoringModeReporting,
			RequestedParameters: &ua.MonitoringParameters{
				ClientHandle:     uint32(i),
				SamplingInterval: float64(publishingInterval.Milliseconds()) / 2, // Sample at 2x publish rate
				QueueSize:        10,
				DiscardOldest:    true,
			},
		})
	}

	if len(monitoredItems) == 0 {
		return "", fmt.Errorf("no valid monitored items created")
	}

	// Monitor the items
	_, err = sub.Monitor(ctx, ua.TimestampsToReturnBoth, monitoredItems...)
	if err != nil {
		return "", fmt.Errorf("failed to create monitored items: %w", err)
	}

	// Store subscription state
	connection.subMutex.Lock()
	subscription := &OPCUASubscription{
		id:                sub.SubscriptionID,
		publishingInterval: publishingInterval,
		monitoredItems:    make(map[string]*ua.MonitoredItemCreateRequest),
		callback:          callback,
		active:            true,
	}

	for i, item := range monitoredItems {
		if i < len(tags) {
			subscription.monitoredItems[tags[i].Address] = item
		}
	}

	connection.subscriptions[subscriptionID] = subscription
	connection.subMutex.Unlock()

	// Start subscription handler in goroutine
	go h.handleSubscriptionData(connection, subscription, notifCh)

	h.logger.Info("Successfully created OPC-UA subscription", 
		zap.String("device_id", device.ID),
		zap.String("subscription_id", subscriptionID),
		zap.Int("monitored_items", len(monitoredItems)))

	return subscriptionID, nil
}

// DiscoverDevices discovers OPC-UA devices on the network using FindServers.
func (h *OPCUAHandler) DiscoverDevices(ctx context.Context, networkRange string) ([]*Device, error) {
	h.logger.Info("Discovering OPC-UA devices", zap.String("networkRange", networkRange))

	// Parse network range and scan for OPC UA servers
	// This is a simplified implementation - in production you'd want more sophisticated discovery
	devices := make([]*Device, 0)

	// For demo purposes, try a few common endpoints
	commonEndpoints := []string{
		"opc.tcp://localhost:4840",
		"opc.tcp://127.0.0.1:4840",
	}

	for _, endpoint := range commonEndpoints {
		if device := h.tryDiscoverDevice(ctx, endpoint); device != nil {
			devices = append(devices, device)
		}
	}

	h.logger.Info("Discovery completed", 
		zap.String("networkRange", networkRange),
		zap.Int("devices_found", len(devices)))

	return devices, nil
}

// GetDeviceInfo returns detailed information about an OPC-UA device.
func (h *OPCUAHandler) GetDeviceInfo(device *Device) (*DeviceInfo, error) {
	connection, err := h.getConnection(device.ID)
	if err != nil {
		return nil, err
	}

	h.logger.Debug("Getting device info for OPC-UA device", zap.String("device_id", device.ID))

	// Read server status and information
	serverArray, err := h.readServerArray(connection)
	if err != nil {
		h.logger.Warn("Failed to read server array", zap.Error(err))
	}

	namespaceArray, err := h.readNamespaceArray(connection)
	if err != nil {
		h.logger.Warn("Failed to read namespace array", zap.Error(err))
	}

	info := &DeviceInfo{
		Vendor:          "Unknown",
		Model:           "OPC UA Server",
		SerialNumber:    "N/A",
		FirmwareVersion: "Unknown",
		Capabilities:    []string{"OPC-UA"},
		MaxConnections:  100, // Default assumption
		SupportedRates:  []int{100, 250, 500, 1000, 2000, 5000}, // Common sampling rates in ms
		CustomInfo: map[string]string{
			"endpoint":        connection.endpoint,
			"security_policy": connection.securityPolicy,
			"server_array":    strings.Join(serverArray, ", "),
			"namespaces":      strings.Join(namespaceArray, ", "),
		},
	}

	return info, nil
}

// GetDiagnostics returns diagnostic information for an OPC-UA device.
func (h *OPCUAHandler) GetDiagnostics(device *Device) (*Diagnostics, error) {
	connection, err := h.getConnection(device.ID)
	if err != nil {
		return nil, err
	}

	h.logger.Debug("Getting diagnostics for OPC-UA device", zap.String("device_id", device.ID))

	// Test connectivity with a simple read
	startTime := time.Now()
	_, err = h.readServerStatus(connection)
	responseTime := time.Since(startTime)

	isHealthy := err == nil
	
	diagnostics := &Diagnostics{
		IsHealthy:         isHealthy,
		LastCommunication: connection.lastSeen,
		ResponseTime:      responseTime,
		ErrorCount:        0, // Would track in production
		SuccessRate:       1.0, // Would calculate in production
		ConnectionUptime:  time.Since(connection.lastSeen), // Approximate
		Errors:            []DiagnosticError{},
		ProtocolDiagnostics: map[string]interface{}{
			"endpoint":        connection.endpoint,
			"security_policy": connection.securityPolicy,
			"security_mode":   connection.securityMode.String(),
			"subscriptions":   len(connection.subscriptions),
		},
	}

	if err != nil {
		diagnostics.Errors = append(diagnostics.Errors, DiagnosticError{
			Timestamp:   time.Now(),
			ErrorCode:   "CONNECTIVITY_ERROR",
			Description: err.Error(),
			Operation:   "server_status_read",
		})
	}

	return diagnostics, nil
}

// GetSupportedDataTypes returns a list of data types supported by the OPC-UA protocol.
func (h *OPCUAHandler) GetSupportedDataTypes() []string {
	return []string{
		"bool", "sbyte", "byte", "int16", "uint16", "int32", "uint32", 
		"int64", "uint64", "float", "double", "string", "datetime", 
		"guid", "bytestring", "xmlelement", "nodeid", "expandednodeid",
		"statuscode", "qualifiedname", "localizedtext", "extensionobject",
		"datavalue", "variant", "diagnosticinfo",
	}
}

// ValidateTagAddress validates an OPC-UA tag address (NodeID).
func (h *OPCUAHandler) ValidateTagAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	// Try to parse as NodeID
	_, err := ua.ParseNodeID(address)
	if err != nil {
		return fmt.Errorf("invalid NodeID format: %w", err)
	}

	return nil
}

// Ping checks if the OPC-UA device is reachable by reading server status.
func (h *OPCUAHandler) Ping(device *Device) error {
	connection, err := h.getConnection(device.ID)
	if err != nil {
		return err
	}

	h.logger.Debug("Pinging OPC-UA device", zap.String("device_id", device.ID))

	// Read server status as a connectivity test
	_, err = h.readServerStatus(connection)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	connection.lastSeen = time.Now()
	return nil
}

// Helper functions for OPC UA implementation

func (h *OPCUAHandler) parseOPCUAConfig(config map[string]interface{}) *OPCUAConfig {
	opcuaConfig := &OPCUAConfig{
		SecurityPolicy:     "None",
		SecurityMode:       "None",
		AuthPolicy:         "Anonymous",
		SessionTimeout:     time.Minute * 30,
		MaxConcurrentReads: h.maxConcurrentReads,
		BatchSize:          h.batchSize,
		ReadTimeout:        h.readTimeout,
	}

	if val, ok := config["security_policy"].(string); ok {
		opcuaConfig.SecurityPolicy = val
	}
	if val, ok := config["security_mode"].(string); ok {
		opcuaConfig.SecurityMode = val
	}
	if val, ok := config["auth_policy"].(string); ok {
		opcuaConfig.AuthPolicy = val
	}
	if val, ok := config["username"].(string); ok {
		opcuaConfig.Username = val
	}
	if val, ok := config["password"].(string); ok {
		opcuaConfig.Password = val
	}

	return opcuaConfig
}

func (h *OPCUAHandler) buildEndpointURL(address string, port int) string {
	if !strings.HasPrefix(address, "opc.tcp://") {
		if port == 0 {
			port = 4840 // Default OPC UA port
		}
		return fmt.Sprintf("opc.tcp://%s:%d", address, port)
	}
	return address
}

func (h *OPCUAHandler) createSecureClient(endpoint string, config *OPCUAConfig) (*opcua.Client, error) {
	opts := []opcua.Option{
		opcua.SecurityPolicy(config.SecurityPolicy),
		opcua.SecurityModeString(config.SecurityMode),
	}

	// Add authentication if configured
	if config.AuthPolicy != "Anonymous" && config.Username != "" {
		opts = append(opts, opcua.AuthUsername(config.Username, config.Password))
	}

	// Add certificate-based authentication if configured
	if config.CertificatePath != "" && config.PrivateKeyPath != "" {
		opts = append(opts, 
			opcua.CertificateFile(config.CertificatePath),
			opcua.PrivateKeyFile(config.PrivateKeyPath))
	}

	// Session configuration
	opts = append(opts, 
		opcua.SessionTimeout(config.SessionTimeout),
		opcua.RequestTimeout(config.ReadTimeout))

	return opcua.NewClient(endpoint, opts...)
}

func (h *OPCUAHandler) parseSecurityMode(mode string) ua.MessageSecurityMode {
	switch strings.ToLower(mode) {
	case "sign":
		return ua.MessageSecurityModeSign
	case "signandencrypt":
		return ua.MessageSecurityModeSignAndEncrypt
	default:
		return ua.MessageSecurityModeNone
	}
}

func (h *OPCUAHandler) parseAuthPolicy(policy string) ua.UserTokenType {
	switch strings.ToLower(policy) {
	case "username":
		return ua.UserTokenTypeUserName
	case "certificate":
		return ua.UserTokenTypeCertificate
	case "issuedtoken":
		return ua.UserTokenTypeIssuedToken
	default:
		return ua.UserTokenTypeAnonymous
	}
}

func (h *OPCUAHandler) getConnection(deviceID string) (*OPCUAConnection, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	connection, exists := h.connections[deviceID]
	if !exists {
		return nil, fmt.Errorf("device %s not connected", deviceID)
	}

	if !connection.connected {
		return nil, fmt.Errorf("device %s connection is not active", deviceID)
	}

	return connection, nil
}

func (h *OPCUAHandler) readTagBatch(connection *OPCUAConnection, tags []*Tag) (map[string]interface{}, error) {
	// Create read request for batch
	var nodesToRead []*ua.ReadValueID
	for _, tag := range tags {
		nodeID, err := ua.ParseNodeID(tag.Address)
		if err != nil {
			h.logger.Warn("Invalid node ID in batch", 
				zap.String("node_id", tag.Address),
				zap.Error(err))
			continue
		}

		nodesToRead = append(nodesToRead, &ua.ReadValueID{
			NodeID:      nodeID,
			AttributeID: ua.AttributeIDValue,
		})
	}

	if len(nodesToRead) == 0 {
		return make(map[string]interface{}), nil
	}

	request := &ua.ReadRequest{
		MaxAge:      0,
		NodesToRead: nodesToRead,
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.readTimeout)
	defer cancel()

	response, err := connection.client.Read(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("batch read failed: %w", err)
	}

	// Process results
	results := make(map[string]interface{})
	validTagIndex := 0

	for _, tag := range tags {
		// Skip tags that were invalid during parsing
		if validTagIndex >= len(response.Results) {
			break
		}

		result := response.Results[validTagIndex]
		validTagIndex++

		if result.Status == ua.StatusOK {
			value := h.convertVariantValue(result.Value)
			results[tag.Address] = value
			
			// Update tag metadata
			tag.Value = value
			tag.Quality = h.mapStatusToQuality(result.Status)
			tag.Timestamp = time.Now()
		} else {
			h.logger.Debug("Bad status for node in batch", 
				zap.String("node_id", tag.Address),
				zap.String("status", result.Status.Error()))
		}
	}

	return results, nil
}

func (h *OPCUAHandler) convertVariantValue(variant *ua.Variant) interface{} {
	if variant == nil {
		return nil
	}

	switch variant.Type() {
	case ua.TypeIDBoolean:
		return variant.Bool()
	case ua.TypeIDSByte:
		return int8(variant.Int())
	case ua.TypeIDByte:
		return uint8(variant.Uint())
	case ua.TypeIDInt16:
		return int16(variant.Int())
	case ua.TypeIDUint16:
		return uint16(variant.Uint())
	case ua.TypeIDInt32:
		return int32(variant.Int())
	case ua.TypeIDUint32:
		return uint32(variant.Uint())
	case ua.TypeIDInt64:
		return variant.Int()
	case ua.TypeIDUint64:
		return variant.Uint()
	case ua.TypeIDFloat:
		return float32(variant.Float())
	case ua.TypeIDDouble:
		return variant.Float()
	case ua.TypeIDString:
		return variant.String()
	case ua.TypeIDDateTime:
		return variant.Time()
	default:
		return variant.Value()
	}
}

func (h *OPCUAHandler) convertToVariant(value interface{}, dataType string) (*ua.Variant, error) {
	switch strings.ToLower(dataType) {
	case "bool", "boolean":
		if val, ok := value.(bool); ok {
			return ua.MustVariant(val), nil
		}
	case "int16":
		if val, ok := value.(int16); ok {
			return ua.MustVariant(val), nil
		}
		if val, ok := value.(int); ok {
			return ua.MustVariant(int16(val)), nil
		}
	case "int32":
		if val, ok := value.(int32); ok {
			return ua.MustVariant(val), nil
		}
		if val, ok := value.(int); ok {
			return ua.MustVariant(int32(val)), nil
		}
	case "float32", "float":
		if val, ok := value.(float32); ok {
			return ua.MustVariant(val), nil
		}
		if val, ok := value.(float64); ok {
			return ua.MustVariant(float32(val)), nil
		}
	case "float64", "double":
		if val, ok := value.(float64); ok {
			return ua.MustVariant(val), nil
		}
	case "string":
		if val, ok := value.(string); ok {
			return ua.MustVariant(val), nil
		}
	}

	// Try to auto-detect type
	return ua.MustVariant(value), nil
}

func (h *OPCUAHandler) mapStatusToQuality(status ua.StatusCode) Quality {
	switch status {
	case ua.StatusOK:
		return QualityGood
	default:
		// Check if status indicates bad or uncertain quality
		if status != ua.StatusOK {
			return QualityBad
		}
		return QualityStale
	}
}

// Browse operations
type BrowseResult struct {
	NodeID      string `json:"node_id"`
	DisplayName string `json:"display_name"`
	NodeClass   string `json:"node_class"`
	DataType    string `json:"data_type"`
	Writable    bool   `json:"writable"`
	Children    []*BrowseResult `json:"children,omitempty"`
}

func (h *OPCUAHandler) browseRecursive(connection *OPCUAConnection, nodeID *ua.NodeID, currentDepth, maxDepth int) ([]*BrowseResult, error) {
	if currentDepth >= maxDepth {
		return nil, nil
	}

	request := &ua.BrowseRequest{
		NodesToBrowse: []*ua.BrowseDescription{
			{
				NodeID:          nodeID,
				BrowseDirection: ua.BrowseDirectionForward,
				ReferenceTypeID: ua.NewNumericNodeID(0, id.HierarchicalReferences),
				IncludeSubtypes: true,
				NodeClassMask:   uint32(ua.NodeClassAll),
				ResultMask:      uint32(ua.BrowseResultMaskAll),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.readTimeout)
	defer cancel()

	response, err := connection.client.Browse(ctx, request)
	if err != nil {
		return nil, err
	}

	var results []*BrowseResult
	if len(response.Results) > 0 {
		for _, ref := range response.Results[0].References {
			result := &BrowseResult{
				NodeID:      ref.NodeID.String(),
				DisplayName: ref.DisplayName.Text,
				NodeClass:   ref.NodeClass.String(),
			}

			// Browse children recursively
			if currentDepth < maxDepth-1 {
				// Convert ExpandedNodeID to NodeID for recursion
				nodeID := ua.NewNodeIDFromExpandedNodeID(ref.NodeID)
				children, err := h.browseRecursive(connection, nodeID, currentDepth+1, maxDepth)
				if err == nil {
					result.Children = children
				}
			}

			results = append(results, result)
		}
	}

	return results, nil
}

// Subscription handling
func (h *OPCUAHandler) handleSubscriptionData(connection *OPCUAConnection, subscription *OPCUASubscription, notifCh chan *opcua.PublishNotificationData) {
	h.logger.Debug("Starting subscription data handler", 
		zap.Uint32("subscription_id", subscription.id))

	for subscription.active {
		select {
		case notif := <-notifCh:
			if notif != nil {
				h.processSubscriptionNotification(subscription, notif)
			}
		case <-time.After(subscription.publishingInterval * 2):
			// Timeout - check if subscription is still active
			if !subscription.active {
				break
			}
		}
	}

	h.logger.Debug("Subscription data handler stopped", 
		zap.Uint32("subscription_id", subscription.id))
}

func (h *OPCUAHandler) processSubscriptionNotification(subscription *OPCUASubscription, notif *opcua.PublishNotificationData) {
	// Process notification data and call callback
	if subscription.callback != nil {
		values := make(map[string]interface{})
		// In a real implementation, you would parse the notification data
		// and extract the actual values for the monitored items
		subscription.callback(values)
	}
}

func (h *OPCUAHandler) closeSubscription(deviceID, subscriptionID string) error {
	connection := h.connections[deviceID]
	if connection == nil {
		return fmt.Errorf("device not connected")
	}

	connection.subMutex.Lock()
	defer connection.subMutex.Unlock()

	subscription, exists := connection.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription not found")
	}

	subscription.active = false

	// In the new API, subscriptions are managed differently
	// The subscription would be cancelled through the context or Cancel method
	
	delete(connection.subscriptions, subscriptionID)
	return nil
}

// Helper functions for device info
func (h *OPCUAHandler) readServerArray(connection *OPCUAConnection) ([]string, error) {
	nodeID := ua.NewNumericNodeID(0, id.Server_ServerArray)
	request := &ua.ReadRequest{
		MaxAge: 0,
		NodesToRead: []*ua.ReadValueID{
			{
				NodeID:      nodeID,
				AttributeID: ua.AttributeIDValue,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.readTimeout)
	defer cancel()

	response, err := connection.client.Read(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(response.Results) > 0 && response.Results[0].Status == ua.StatusOK {
		if array, ok := response.Results[0].Value.Value().([]string); ok {
			return array, nil
		}
	}

	return []string{}, nil
}

func (h *OPCUAHandler) readNamespaceArray(connection *OPCUAConnection) ([]string, error) {
	nodeID := ua.NewNumericNodeID(0, id.Server_NamespaceArray)
	request := &ua.ReadRequest{
		MaxAge: 0,
		NodesToRead: []*ua.ReadValueID{
			{
				NodeID:      nodeID,
				AttributeID: ua.AttributeIDValue,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.readTimeout)
	defer cancel()

	response, err := connection.client.Read(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(response.Results) > 0 && response.Results[0].Status == ua.StatusOK {
		if array, ok := response.Results[0].Value.Value().([]string); ok {
			return array, nil
		}
	}

	return []string{}, nil
}

func (h *OPCUAHandler) readServerStatus(connection *OPCUAConnection) (interface{}, error) {
	nodeID := ua.NewNumericNodeID(0, id.Server_ServerStatus)
	request := &ua.ReadRequest{
		MaxAge: 0,
		NodesToRead: []*ua.ReadValueID{
			{
				NodeID:      nodeID,
				AttributeID: ua.AttributeIDValue,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.readTimeout)
	defer cancel()

	response, err := connection.client.Read(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(response.Results) > 0 && response.Results[0].Status == ua.StatusOK {
		return response.Results[0].Value.Value(), nil
	}

	return nil, fmt.Errorf("failed to read server status")
}

func (h *OPCUAHandler) tryDiscoverDevice(ctx context.Context, endpoint string) *Device {
	// Parse endpoint URL
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil
	}

	client, err := opcua.NewClient(endpoint)
	if err != nil {
		return nil
	}

	// Try to connect briefly for discovery
	connectCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if err := client.Connect(connectCtx); err != nil {
		return nil
	}

	// Disconnect immediately after discovery
	defer client.Close(context.Background())

	// Create device info
	device := &Device{
		ID:       fmt.Sprintf("opcua_%s_%s", u.Hostname(), u.Port()),
		Name:     fmt.Sprintf("OPC UA Server (%s)", u.Host),
		Protocol: "opcua",
		Address:  u.Hostname(),
		Config:   make(map[string]interface{}),
	}

	if u.Port() != "" {
		if port, err := strconv.Atoi(u.Port()); err == nil {
			device.Port = port
		}
	}

	return device
}
