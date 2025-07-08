package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"bifrost-gateway/internal/protocols"
)

// NATSMessaging implements the MessagingLayer interface using NATS
type NATSMessaging struct {
	conn         *nats.Conn
	config       *NATSConfig
	logger       *zap.Logger
	topicBuilder TopicBuilder
	
	// Connection state
	connected      int32 // atomic
	reconnectCount int32 // atomic
	lastError      string
	lastConnected  time.Time
	
	// Metrics
	metrics struct {
		published uint64 // atomic
		received  uint64 // atomic
		pubErrors uint64 // atomic
		recErrors uint64 // atomic
	}
	
	// Subscription management
	subscriptions sync.Map // map[string]*nats.Subscription
	mu           sync.RWMutex
}

// NATSConfig holds NATS-specific configuration
type NATSConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Servers   []string `yaml:"servers"`
	ClusterID string   `yaml:"cluster_id"`
	ClientID  string   `yaml:"client_id"`
	
	// Authentication
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"token"`
	NKeyFile string `yaml:"nkey_file"`
	
	// Subject configuration
	Subjects struct {
		Telemetry   string `yaml:"telemetry"`
		Commands    string `yaml:"commands"`
		Events      string `yaml:"events"`
		Control     string `yaml:"control"`
		Coordination string `yaml:"coordination"`
	} `yaml:"subjects"`
	
	// TLS configuration
	TLS struct {
		Enabled  bool   `yaml:"enabled"`
		CAFile   string `yaml:"ca_file"`
		CertFile string `yaml:"cert_file"`
		KeyFile  string `yaml:"key_file"`
		Insecure bool   `yaml:"insecure"`
	} `yaml:"tls"`
	
	// JetStream configuration
	JetStream struct {
		Enabled   bool   `yaml:"enabled"`
		Storage   string `yaml:"storage"` // file, memory
		Retention string `yaml:"retention"` // e.g., "7d"
	} `yaml:"jetstream"`
	
	// Connection settings
	MaxReconnects   int           `yaml:"max_reconnects"`
	ReconnectWait   time.Duration `yaml:"reconnect_wait"`
	ConnectTimeout  time.Duration `yaml:"connect_timeout"`
	RequestTimeout  time.Duration `yaml:"request_timeout"`
	DrainTimeout    time.Duration `yaml:"drain_timeout"`
}

// NewNATSMessaging creates a new NATS messaging instance
func NewNATSMessaging(config *NATSConfig, logger *zap.Logger) (*NATSMessaging, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("NATS messaging is disabled")
	}
	
	m := &NATSMessaging{
		config:       config,
		logger:       logger,
		topicBuilder: NewNATSTopicBuilder(config),
	}
	
	return m, nil
}

// Connect establishes connection to NATS servers
func (n *NATSMessaging) Connect(ctx context.Context) error {
	n.logger.Info("Connecting to NATS servers", zap.Strings("servers", n.config.Servers))
	
	// Configure NATS options
	opts := []nats.Option{
		nats.Name(n.config.ClientID),
		nats.MaxReconnects(n.config.MaxReconnects),
		nats.ReconnectWait(n.config.ReconnectWait),
		nats.Timeout(n.config.ConnectTimeout),
		nats.DrainTimeout(n.config.DrainTimeout),
	}
	
	// Add authentication if configured
	if n.config.Username != "" {
		opts = append(opts, nats.UserInfo(n.config.Username, n.config.Password))
	} else if n.config.Token != "" {
		opts = append(opts, nats.Token(n.config.Token))
	}
	
	// Add event handlers
	opts = append(opts,
		nats.DisconnectErrHandler(n.onDisconnect),
		nats.ReconnectHandler(n.onReconnect),
		nats.ClosedHandler(n.onClosed),
		nats.ErrorHandler(n.onError),
	)
	
	// Connect to NATS
	conn, err := nats.Connect(
		fmt.Sprintf("nats://%s", n.config.Servers[0]), // Use first server, NATS client handles failover
		opts...,
	)
	if err != nil {
		n.setError(err.Error())
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	
	n.conn = conn
	atomic.StoreInt32(&n.connected, 1)
	n.lastConnected = time.Now()
	n.clearError()
	
	n.logger.Info("Successfully connected to NATS")
	return nil
}

// Disconnect closes the connection to NATS
func (n *NATSMessaging) Disconnect() error {
	if !n.IsConnected() {
		return nil
	}
	
	n.logger.Info("Disconnecting from NATS")
	
	// Drain and close connection
	if err := n.conn.Drain(); err != nil {
		n.logger.Warn("Error draining NATS connection", zap.Error(err))
	}
	
	n.conn.Close()
	atomic.StoreInt32(&n.connected, 0)
	
	return nil
}

// IsConnected returns the current connection status
func (n *NATSMessaging) IsConnected() bool {
	return atomic.LoadInt32(&n.connected) == 1 && n.conn != nil && n.conn.IsConnected()
}

// PublishDeviceData publishes device data to the telemetry subject
func (n *NATSMessaging) PublishDeviceData(deviceID string, data *protocols.Tag) error {
	if !n.IsConnected() {
		return fmt.Errorf("not connected to NATS")
	}
	
	subject := n.topicBuilder.TelemetryTopic("default", deviceID, data.Name)
	payload, err := json.Marshal(data)
	if err != nil {
		atomic.AddUint64(&n.metrics.pubErrors, 1)
		return fmt.Errorf("failed to marshal tag data: %w", err)
	}
	
	if err := n.conn.Publish(subject, payload); err != nil {
		atomic.AddUint64(&n.metrics.pubErrors, 1)
		return fmt.Errorf("failed to publish: %w", err)
	}
	
	atomic.AddUint64(&n.metrics.published, 1)
	return nil
}

// PublishDeviceEvent publishes a device event
func (n *NATSMessaging) PublishDeviceEvent(deviceID string, event *DeviceEvent) error {
	if !n.IsConnected() {
		return fmt.Errorf("not connected to NATS")
	}
	
	subject := n.topicBuilder.EventTopic("default", deviceID, event.EventType)
	payload, err := json.Marshal(event)
	if err != nil {
		atomic.AddUint64(&n.metrics.pubErrors, 1)
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	if err := n.conn.Publish(subject, payload); err != nil {
		atomic.AddUint64(&n.metrics.pubErrors, 1)
		return fmt.Errorf("failed to publish event: %w", err)
	}
	
	atomic.AddUint64(&n.metrics.published, 1)
	return nil
}

// PublishAlarm publishes an alarm event
func (n *NATSMessaging) PublishAlarm(deviceID string, alarm *AlarmEvent) error {
	if !n.IsConnected() {
		return fmt.Errorf("not connected to NATS")
	}
	
	subject := n.topicBuilder.AlarmTopic("default", deviceID, alarm.Severity)
	payload, err := json.Marshal(alarm)
	if err != nil {
		atomic.AddUint64(&n.metrics.pubErrors, 1)
		return fmt.Errorf("failed to marshal alarm: %w", err)
	}
	
	if err := n.conn.Publish(subject, payload); err != nil {
		atomic.AddUint64(&n.metrics.pubErrors, 1)
		return fmt.Errorf("failed to publish alarm: %w", err)
	}
	
	atomic.AddUint64(&n.metrics.published, 1)
	return nil
}

// Subscribe subscribes to a subject with a message handler
func (n *NATSMessaging) Subscribe(subject string, handler MessageHandler) error {
	if !n.IsConnected() {
		return fmt.Errorf("not connected to NATS")
	}
	
	sub, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		if err := handler(msg.Subject, msg.Data); err != nil {
			n.logger.Error("Message handler error", 
				zap.String("subject", msg.Subject),
				zap.Error(err))
			atomic.AddUint64(&n.metrics.recErrors, 1)
		} else {
			atomic.AddUint64(&n.metrics.received, 1)
		}
	})
	
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	
	n.subscriptions.Store(subject, sub)
	return nil
}

// Unsubscribe unsubscribes from a subject
func (n *NATSMessaging) Unsubscribe(subject string) error {
	if sub, ok := n.subscriptions.Load(subject); ok {
		if err := sub.(*nats.Subscription).Unsubscribe(); err != nil {
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}
		n.subscriptions.Delete(subject)
	}
	return nil
}

// RequestReply implements synchronous request-reply pattern
func (n *NATSMessaging) RequestReply(subject string, data []byte, timeout time.Duration) ([]byte, error) {
	if !n.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS")
	}
	
	msg, err := n.conn.Request(subject, data, timeout)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	
	return msg.Data, nil
}

// GetConnectionStatus returns the current connection status
func (n *NATSMessaging) GetConnectionStatus() *ConnectionStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	return &ConnectionStatus{
		Connected:     n.IsConnected(),
		LastConnected: n.lastConnected,
		LastError:     n.lastError,
		ReconnectCount: int(atomic.LoadInt32(&n.reconnectCount)),
	}
}

// GetMetrics returns messaging metrics
func (n *NATSMessaging) GetMetrics() *MessagingMetrics {
	return &MessagingMetrics{
		MessagesPublished: atomic.LoadUint64(&n.metrics.published),
		MessagesReceived:  atomic.LoadUint64(&n.metrics.received),
		PublishErrors:     atomic.LoadUint64(&n.metrics.pubErrors),
		ReceiveErrors:     atomic.LoadUint64(&n.metrics.recErrors),
		LastUpdate:        time.Now(),
	}
}

// Event handlers
func (n *NATSMessaging) onDisconnect(conn *nats.Conn, err error) {
	atomic.StoreInt32(&n.connected, 0)
	if err != nil {
		n.setError(err.Error())
		n.logger.Warn("NATS disconnected", zap.Error(err))
	} else {
		n.logger.Info("NATS disconnected")
	}
}

func (n *NATSMessaging) onReconnect(conn *nats.Conn) {
	atomic.StoreInt32(&n.connected, 1)
	atomic.AddInt32(&n.reconnectCount, 1)
	n.lastConnected = time.Now()
	n.clearError()
	n.logger.Info("NATS reconnected")
}

func (n *NATSMessaging) onClosed(conn *nats.Conn) {
	atomic.StoreInt32(&n.connected, 0)
	n.logger.Info("NATS connection closed")
}

func (n *NATSMessaging) onError(conn *nats.Conn, sub *nats.Subscription, err error) {
	n.setError(err.Error())
	n.logger.Error("NATS error", zap.Error(err))
}

func (n *NATSMessaging) setError(err string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastError = err
}

func (n *NATSMessaging) clearError() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastError = ""
}