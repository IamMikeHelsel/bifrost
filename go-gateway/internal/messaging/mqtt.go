package messaging

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"bifrost-gateway/internal/protocols"
)

// MQTTMessaging implements the MessagingLayer interface using MQTT
type MQTTMessaging struct {
	client       mqtt.Client
	config       *MQTTConfig
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
	subscriptions sync.Map // map[string]MessageHandler
	mu           sync.RWMutex
}

// MQTTConfig holds MQTT-specific configuration
type MQTTConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Broker     string `yaml:"broker"`
	ClientID   string `yaml:"client_id"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	QoS        byte   `yaml:"qos"`
	Retain     bool   `yaml:"retain"`
	
	// Topic configuration
	Topics struct {
		Telemetry   string `yaml:"telemetry"`
		Commands    string `yaml:"commands"`
		Alarms      string `yaml:"alarms"`
		Events      string `yaml:"events"`
		Diagnostics string `yaml:"diagnostics"`
	} `yaml:"topics"`
	
	// TLS configuration
	TLS struct {
		Enabled  bool   `yaml:"enabled"`
		CAFile   string `yaml:"ca_file"`
		CertFile string `yaml:"cert_file"`
		KeyFile  string `yaml:"key_file"`
		Insecure bool   `yaml:"insecure"`
	} `yaml:"tls"`
	
	// Connection settings
	KeepAlive        time.Duration `yaml:"keep_alive"`
	ConnectTimeout   time.Duration `yaml:"connect_timeout"`
	WriteTimeout     time.Duration `yaml:"write_timeout"`
	MaxReconnectWait time.Duration `yaml:"max_reconnect_wait"`
	AutoReconnect    bool          `yaml:"auto_reconnect"`
	CleanSession     bool          `yaml:"clean_session"`
}

// NewMQTTMessaging creates a new MQTT messaging instance
func NewMQTTMessaging(config *MQTTConfig, logger *zap.Logger) (*MQTTMessaging, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("MQTT messaging is disabled")
	}
	
	m := &MQTTMessaging{
		config:       config,
		logger:       logger,
		topicBuilder: NewMQTTTopicBuilder(config),
	}
	
	// Configure MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Broker)
	opts.SetClientID(config.ClientID)
	opts.SetKeepAlive(config.KeepAlive)
	opts.SetConnectTimeout(config.ConnectTimeout)
	opts.SetWriteTimeout(config.WriteTimeout)
	opts.SetAutoReconnect(config.AutoReconnect)
	opts.SetCleanSession(config.CleanSession)
	
	if config.Username != "" {
		opts.SetUsername(config.Username)
		opts.SetPassword(config.Password)
	}
	
	// Configure TLS if enabled
	if config.TLS.Enabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.TLS.Insecure,
		}
		opts.SetTLSConfig(tlsConfig)
	}
	
	// Set connection callbacks
	opts.SetConnectionLostHandler(m.onConnectionLost)
	opts.SetOnConnectHandler(m.onConnect)
	opts.SetDefaultPublishHandler(m.onMessage)
	
	m.client = mqtt.NewClient(opts)
	
	return m, nil
}

// Connect establishes connection to the MQTT broker
func (m *MQTTMessaging) Connect(ctx context.Context) error {
	m.logger.Info("Connecting to MQTT broker", zap.String("broker", m.config.Broker))
	
	token := m.client.Connect()
	if !token.WaitTimeout(m.config.ConnectTimeout) {
		return fmt.Errorf("connection timeout")
	}
	
	if err := token.Error(); err != nil {
		m.setError(err.Error())
		return fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}
	
	atomic.StoreInt32(&m.connected, 1)
	m.lastConnected = time.Now()
	m.clearError()
	
	m.logger.Info("Successfully connected to MQTT broker")
	return nil
}

// Disconnect closes the connection to the MQTT broker
func (m *MQTTMessaging) Disconnect() error {
	if !m.IsConnected() {
		return nil
	}
	
	m.logger.Info("Disconnecting from MQTT broker")
	m.client.Disconnect(250) // 250ms quiesce time
	atomic.StoreInt32(&m.connected, 0)
	
	return nil
}

// IsConnected returns the current connection status
func (m *MQTTMessaging) IsConnected() bool {
	return atomic.LoadInt32(&m.connected) == 1 && m.client.IsConnected()
}

// PublishDeviceData publishes device data to the telemetry topic
func (m *MQTTMessaging) PublishDeviceData(deviceID string, data *protocols.Tag) error {
	if !m.IsConnected() {
		return fmt.Errorf("not connected to MQTT broker")
	}
	
	topic := m.topicBuilder.TelemetryTopic("default", deviceID, data.Name)
	payload, err := json.Marshal(data)
	if err != nil {
		atomic.AddUint64(&m.metrics.pubErrors, 1)
		return fmt.Errorf("failed to marshal tag data: %w", err)
	}
	
	token := m.client.Publish(topic, m.config.QoS, m.config.Retain, payload)
	if !token.WaitTimeout(m.config.WriteTimeout) {
		atomic.AddUint64(&m.metrics.pubErrors, 1)
		return fmt.Errorf("publish timeout")
	}
	
	if err := token.Error(); err != nil {
		atomic.AddUint64(&m.metrics.pubErrors, 1)
		return fmt.Errorf("failed to publish: %w", err)
	}
	
	atomic.AddUint64(&m.metrics.published, 1)
	return nil
}

// PublishDeviceEvent publishes a device event
func (m *MQTTMessaging) PublishDeviceEvent(deviceID string, event *DeviceEvent) error {
	if !m.IsConnected() {
		return fmt.Errorf("not connected to MQTT broker")
	}
	
	topic := m.topicBuilder.EventTopic("default", deviceID, event.EventType)
	payload, err := json.Marshal(event)
	if err != nil {
		atomic.AddUint64(&m.metrics.pubErrors, 1)
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	token := m.client.Publish(topic, m.config.QoS, m.config.Retain, payload)
	if !token.WaitTimeout(m.config.WriteTimeout) {
		atomic.AddUint64(&m.metrics.pubErrors, 1)
		return fmt.Errorf("publish timeout")
	}
	
	if err := token.Error(); err != nil {
		atomic.AddUint64(&m.metrics.pubErrors, 1)
		return fmt.Errorf("failed to publish event: %w", err)
	}
	
	atomic.AddUint64(&m.metrics.published, 1)
	return nil
}

// PublishAlarm publishes an alarm event
func (m *MQTTMessaging) PublishAlarm(deviceID string, alarm *AlarmEvent) error {
	if !m.IsConnected() {
		return fmt.Errorf("not connected to MQTT broker")
	}
	
	topic := m.topicBuilder.AlarmTopic("default", deviceID, alarm.Severity)
	payload, err := json.Marshal(alarm)
	if err != nil {
		atomic.AddUint64(&m.metrics.pubErrors, 1)
		return fmt.Errorf("failed to marshal alarm: %w", err)
	}
	
	token := m.client.Publish(topic, m.config.QoS, m.config.Retain, payload)
	if !token.WaitTimeout(m.config.WriteTimeout) {
		atomic.AddUint64(&m.metrics.pubErrors, 1)
		return fmt.Errorf("publish timeout")
	}
	
	if err := token.Error(); err != nil {
		atomic.AddUint64(&m.metrics.pubErrors, 1)
		return fmt.Errorf("failed to publish alarm: %w", err)
	}
	
	atomic.AddUint64(&m.metrics.published, 1)
	return nil
}

// Subscribe subscribes to a topic with a message handler
func (m *MQTTMessaging) Subscribe(subject string, handler MessageHandler) error {
	if !m.IsConnected() {
		return fmt.Errorf("not connected to MQTT broker")
	}
	
	token := m.client.Subscribe(subject, m.config.QoS, func(client mqtt.Client, msg mqtt.Message) {
		if err := handler(msg.Topic(), msg.Payload()); err != nil {
			m.logger.Error("Message handler error", 
				zap.String("topic", msg.Topic()),
				zap.Error(err))
			atomic.AddUint64(&m.metrics.recErrors, 1)
		} else {
			atomic.AddUint64(&m.metrics.received, 1)
		}
	})
	
	if !token.WaitTimeout(m.config.WriteTimeout) {
		return fmt.Errorf("subscribe timeout")
	}
	
	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	
	m.subscriptions.Store(subject, handler)
	return nil
}

// Unsubscribe unsubscribes from a topic
func (m *MQTTMessaging) Unsubscribe(subject string) error {
	if !m.IsConnected() {
		return fmt.Errorf("not connected to MQTT broker")
	}
	
	token := m.client.Unsubscribe(subject)
	if !token.WaitTimeout(m.config.WriteTimeout) {
		return fmt.Errorf("unsubscribe timeout")
	}
	
	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}
	
	m.subscriptions.Delete(subject)
	return nil
}

// RequestReply implements request-reply pattern (not native to MQTT)
func (m *MQTTMessaging) RequestReply(subject string, data []byte, timeout time.Duration) ([]byte, error) {
	return nil, fmt.Errorf("request-reply not supported by MQTT")
}

// GetConnectionStatus returns the current connection status
func (m *MQTTMessaging) GetConnectionStatus() *ConnectionStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return &ConnectionStatus{
		Connected:     m.IsConnected(),
		LastConnected: m.lastConnected,
		LastError:     m.lastError,
		ReconnectCount: int(atomic.LoadInt32(&m.reconnectCount)),
	}
}

// GetMetrics returns messaging metrics
func (m *MQTTMessaging) GetMetrics() *MessagingMetrics {
	return &MessagingMetrics{
		MessagesPublished: atomic.LoadUint64(&m.metrics.published),
		MessagesReceived:  atomic.LoadUint64(&m.metrics.received),
		PublishErrors:     atomic.LoadUint64(&m.metrics.pubErrors),
		ReceiveErrors:     atomic.LoadUint64(&m.metrics.recErrors),
		LastUpdate:        time.Now(),
	}
}

// Connection event handlers
func (m *MQTTMessaging) onConnectionLost(client mqtt.Client, err error) {
	atomic.StoreInt32(&m.connected, 0)
	atomic.AddInt32(&m.reconnectCount, 1)
	m.setError(err.Error())
	m.logger.Warn("MQTT connection lost", zap.Error(err))
}

func (m *MQTTMessaging) onConnect(client mqtt.Client) {
	atomic.StoreInt32(&m.connected, 1)
	m.lastConnected = time.Now()
	m.clearError()
	m.logger.Info("MQTT connection established")
}

func (m *MQTTMessaging) onMessage(client mqtt.Client, msg mqtt.Message) {
	// Default message handler for unsubscribed messages
	m.logger.Debug("Received unhandled message", 
		zap.String("topic", msg.Topic()),
		zap.Int("payload_size", len(msg.Payload())))
}

func (m *MQTTMessaging) setError(err string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastError = err
}

func (m *MQTTMessaging) clearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastError = ""
}