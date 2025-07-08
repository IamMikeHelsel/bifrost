package cloud

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Buffer provides smart buffering with disk persistence
type Buffer interface {
	Add(message *CloudMessage) error
	Get(count int) ([]*CloudMessage, error)
	Remove(messageIDs []string) error
	Size() int
	Close() error
}

// DiskBuffer implements Buffer with disk persistence
type DiskBuffer struct {
	logger      *zap.Logger
	config      *BufferConfig
	messages    map[string]*CloudMessage
	queue       []*CloudMessage
	mutex       sync.RWMutex
	filePath    string
	flushTimer  *time.Timer
	shouldFlush bool
}

// BufferConfig holds configuration for the buffer
type BufferConfig struct {
	MaxSize         int           `yaml:"max_size"`
	FlushInterval   time.Duration `yaml:"flush_interval"`
	PersistentPath  string        `yaml:"persistent_path"`
	CompressionType string        `yaml:"compression_type"`
	MaxFileSize     int64         `yaml:"max_file_size"`
	RetentionTime   time.Duration `yaml:"retention_time"`
}

// NewDiskBuffer creates a new disk-backed buffer
func NewDiskBuffer(logger *zap.Logger, config *BufferConfig) (*DiskBuffer, error) {
	if config.PersistentPath == "" {
		config.PersistentPath = "/tmp/bifrost-buffer"
	}
	
	// Ensure directory exists
	if err := os.MkdirAll(config.PersistentPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create buffer directory: %w", err)
	}
	
	buffer := &DiskBuffer{
		logger:   logger,
		config:   config,
		messages: make(map[string]*CloudMessage),
		queue:    make([]*CloudMessage, 0, config.MaxSize),
		filePath: filepath.Join(config.PersistentPath, "buffer.json"),
	}
	
	// Load existing data
	if err := buffer.loadFromDisk(); err != nil {
		logger.Warn("Failed to load buffer from disk", zap.Error(err))
	}
	
	// Start flush timer
	buffer.startFlushTimer()
	
	return buffer, nil
}

// Add adds a message to the buffer
func (b *DiskBuffer) Add(message *CloudMessage) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	// Check if buffer is full
	if len(b.queue) >= b.config.MaxSize {
		// Remove oldest message
		oldest := b.queue[0]
		delete(b.messages, oldest.ID)
		b.queue = b.queue[1:]
		b.logger.Warn("Buffer full, dropping oldest message", zap.String("messageID", oldest.ID))
	}
	
	// Add new message
	b.messages[message.ID] = message
	b.queue = append(b.queue, message)
	b.shouldFlush = true
	
	b.logger.Debug("Added message to buffer", 
		zap.String("messageID", message.ID),
		zap.Int("bufferSize", len(b.queue)))
	
	return nil
}

// Get retrieves up to count messages from the buffer
func (b *DiskBuffer) Get(count int) ([]*CloudMessage, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	
	if count <= 0 || len(b.queue) == 0 {
		return nil, nil
	}
	
	if count > len(b.queue) {
		count = len(b.queue)
	}
	
	// Return copy of messages to avoid race conditions
	result := make([]*CloudMessage, count)
	for i := 0; i < count; i++ {
		result[i] = b.queue[i]
	}
	
	return result, nil
}

// Remove removes messages from the buffer by ID
func (b *DiskBuffer) Remove(messageIDs []string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	removed := 0
	for _, id := range messageIDs {
		if _, exists := b.messages[id]; exists {
			delete(b.messages, id)
			removed++
		}
	}
	
	// Rebuild queue without removed messages
	newQueue := make([]*CloudMessage, 0, len(b.queue))
	for _, msg := range b.queue {
		if _, exists := b.messages[msg.ID]; exists {
			newQueue = append(newQueue, msg)
		}
	}
	b.queue = newQueue
	
	if removed > 0 {
		b.shouldFlush = true
		b.logger.Debug("Removed messages from buffer", 
			zap.Int("removedCount", removed),
			zap.Int("remainingSize", len(b.queue)))
	}
	
	return nil
}

// Size returns the current buffer size
func (b *DiskBuffer) Size() int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return len(b.queue)
}

// Close closes the buffer and saves to disk
func (b *DiskBuffer) Close() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	if b.flushTimer != nil {
		b.flushTimer.Stop()
	}
	
	return b.saveToDisk()
}

// loadFromDisk loads messages from disk
func (b *DiskBuffer) loadFromDisk() error {
	file, err := os.Open(b.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's OK
		}
		return err
	}
	defer file.Close()
	
	var savedMessages []*CloudMessage
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&savedMessages); err != nil {
		return err
	}
	
	// Filter out expired messages
	now := time.Now()
	for _, msg := range savedMessages {
		if !msg.Expires.IsZero() && msg.Expires.Before(now) {
			continue // Skip expired messages
		}
		
		b.messages[msg.ID] = msg
		b.queue = append(b.queue, msg)
	}
	
	b.logger.Info("Loaded messages from disk", 
		zap.Int("loadedCount", len(b.queue)),
		zap.Int("totalCount", len(savedMessages)))
	
	return nil
}

// saveToDisk saves current messages to disk
func (b *DiskBuffer) saveToDisk() error {
	if !b.shouldFlush {
		return nil
	}
	
	file, err := os.Create(b.filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(b.queue); err != nil {
		return err
	}
	
	b.shouldFlush = false
	b.logger.Debug("Saved buffer to disk", zap.Int("messageCount", len(b.queue)))
	
	return nil
}

// startFlushTimer starts the periodic flush timer
func (b *DiskBuffer) startFlushTimer() {
	if b.config.FlushInterval <= 0 {
		return
	}
	
	b.flushTimer = time.AfterFunc(b.config.FlushInterval, func() {
		b.mutex.Lock()
		if err := b.saveToDisk(); err != nil {
			b.logger.Error("Failed to flush buffer to disk", zap.Error(err))
		}
		b.mutex.Unlock()
		
		// Schedule next flush
		b.startFlushTimer()
	})
}

// MemoryBuffer implements Buffer with in-memory storage only
type MemoryBuffer struct {
	logger   *zap.Logger
	config   *BufferConfig
	messages map[string]*CloudMessage
	queue    []*CloudMessage
	mutex    sync.RWMutex
}

// NewMemoryBuffer creates a new memory-only buffer
func NewMemoryBuffer(logger *zap.Logger, config *BufferConfig) *MemoryBuffer {
	return &MemoryBuffer{
		logger:   logger,
		config:   config,
		messages: make(map[string]*CloudMessage),
		queue:    make([]*CloudMessage, 0, config.MaxSize),
	}
}

// Add adds a message to the memory buffer
func (b *MemoryBuffer) Add(message *CloudMessage) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	// Check if buffer is full
	if len(b.queue) >= b.config.MaxSize {
		// Remove oldest message
		oldest := b.queue[0]
		delete(b.messages, oldest.ID)
		b.queue = b.queue[1:]
	}
	
	// Add new message
	b.messages[message.ID] = message
	b.queue = append(b.queue, message)
	
	return nil
}

// Get retrieves up to count messages from the memory buffer
func (b *MemoryBuffer) Get(count int) ([]*CloudMessage, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	
	if count <= 0 || len(b.queue) == 0 {
		return nil, nil
	}
	
	if count > len(b.queue) {
		count = len(b.queue)
	}
	
	result := make([]*CloudMessage, count)
	copy(result, b.queue[:count])
	
	return result, nil
}

// Remove removes messages from the memory buffer by ID
func (b *MemoryBuffer) Remove(messageIDs []string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	for _, id := range messageIDs {
		delete(b.messages, id)
	}
	
	// Rebuild queue
	newQueue := make([]*CloudMessage, 0, len(b.queue))
	for _, msg := range b.queue {
		if _, exists := b.messages[msg.ID]; exists {
			newQueue = append(newQueue, msg)
		}
	}
	b.queue = newQueue
	
	return nil
}

// Size returns the current memory buffer size
func (b *MemoryBuffer) Size() int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return len(b.queue)
}

// Close closes the memory buffer (no-op)
func (b *MemoryBuffer) Close() error {
	return nil
}