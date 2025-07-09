package protocols

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockConnection represents a mock connection for testing
type MockConnection struct {
	id         int
	connected  bool
	closeCalls int32
	mu         sync.Mutex
}

func (c *MockConnection) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = true
	return nil
}

func (c *MockConnection) Close() error {
	atomic.AddInt32(&c.closeCalls, 1)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	return nil
}

func (c *MockConnection) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *MockConnection) Read(data []byte) (int, error) {
	if !c.IsConnected() {
		return 0, fmt.Errorf("not connected")
	}
	return len(data), nil
}

func (c *MockConnection) Write(data []byte) (int, error) {
	if !c.IsConnected() {
		return 0, fmt.Errorf("not connected")
	}
	return len(data), nil
}

// TestConnectionPool tests connection pool functionality
func TestConnectionPool(t *testing.T) {
	t.Run("Create and destroy pool", func(t *testing.T) {
		pool := NewConnectionPool(5, func() (interface{}, error) {
			return &MockConnection{}, nil
		})
		assert.NotNil(t, pool)
		
		err := pool.Close()
		assert.NoError(t, err)
	})

	t.Run("Get and put connections", func(t *testing.T) {
		connID := 0
		pool := NewConnectionPool(3, func() (interface{}, error) {
			conn := &MockConnection{id: connID}
			connID++
			return conn, nil
		})
		defer pool.Close()

		// Get a connection
		connInterface, err := pool.Get(context.Background())
		require.NoError(t, err)
		conn := connInterface.(*MockConnection)
		assert.NotNil(t, conn)

		// Put it back
		err = pool.Put(conn)
		assert.NoError(t, err)

		// Get it again - should be the same connection
		connInterface2, err := pool.Get(context.Background())
		require.NoError(t, err)
		conn2 := connInterface2.(*MockConnection)
		assert.Equal(t, conn.id, conn2.id)
	})

	t.Run("Max connections enforcement", func(t *testing.T) {
		maxConns := 2
		pool := NewConnectionPool(maxConns, func() (interface{}, error) {
			conn := &MockConnection{}
			conn.Connect()
			return conn, nil
		})
		defer pool.Close()

		// Get max connections
		conns := make([]*MockConnection, maxConns)
		for i := 0; i < maxConns; i++ {
			connInterface, err := pool.Get(context.Background())
			require.NoError(t, err)
			conns[i] = connInterface.(*MockConnection)
		}

		// Try to get one more with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := pool.Get(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")

		// Put one back
		err = pool.Put(conns[0])
		assert.NoError(t, err)

		// Now we should be able to get one
		connInterface, err := pool.Get(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, connInterface)
	})

	t.Run("Connection validation", func(t *testing.T) {
		pool := NewConnectionPool(2, func() (interface{}, error) {
			conn := &MockConnection{}
			conn.Connect()
			return conn, nil
		})
		defer pool.Close()

		// Set validation function
		pool.SetValidator(func(conn interface{}) bool {
			mockConn := conn.(*MockConnection)
			return mockConn.IsConnected()
		})

		// Get a connection
		connInterface, err := pool.Get(context.Background())
		require.NoError(t, err)
		conn := connInterface.(*MockConnection)

		// Disconnect it
		conn.Close()

		// Put it back - should be rejected
		err = pool.Put(conn)
		assert.NoError(t, err)

		// Get a new connection - should create a new one
		connInterface2, err := pool.Get(context.Background())
		require.NoError(t, err)
		conn2 := connInterface2.(*MockConnection)
		assert.True(t, conn2.IsConnected())
	})

	t.Run("Concurrent access", func(t *testing.T) {
		pool := NewConnectionPool(5, func() (interface{}, error) {
			conn := &MockConnection{}
			conn.Connect()
			time.Sleep(10 * time.Millisecond) // Simulate connection time
			return conn, nil
		})
		defer pool.Close()

		var wg sync.WaitGroup
		errors := make(chan error, 100)

		// Spawn multiple goroutines
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				for j := 0; j < 10; j++ {
					// Get connection
					connInterface, err := pool.Get(context.Background())
					if err != nil {
						errors <- err
						return
					}

					// Simulate work
					time.Sleep(5 * time.Millisecond)

					// Put back
					if err := pool.Put(connInterface); err != nil {
						errors <- err
						return
					}
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent access error: %v", err)
		}
	})

	t.Run("Connection factory error", func(t *testing.T) {
		errorCount := 0
		pool := NewConnectionPool(2, func() (interface{}, error) {
			errorCount++
			if errorCount < 3 {
				return nil, fmt.Errorf("connection failed")
			}
			conn := &MockConnection{}
			conn.Connect()
			return conn, nil
		})
		defer pool.Close()

		// First two attempts should fail
		ctx1, cancel1 := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel1()
		_, err := pool.Get(ctx1)
		assert.Error(t, err)

		ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel2()
		_, err = pool.Get(ctx2)
		assert.Error(t, err)

		// Third attempt should succeed
		connInterface, err := pool.Get(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, connInterface)
	})

	t.Run("Pool statistics", func(t *testing.T) {
		pool := NewConnectionPool(3, func() (interface{}, error) {
			conn := &MockConnection{}
			conn.Connect()
			return conn, nil
		})
		defer pool.Close()

		// Get stats before any operations
		stats := pool.Stats()
		assert.Equal(t, 0, stats.ActiveConnections)
		assert.Equal(t, 0, stats.IdleConnections)
		assert.Equal(t, 0, stats.TotalCreated)

		// Get a connection
		conn1, err := pool.Get(context.Background())
		require.NoError(t, err)

		stats = pool.Stats()
		assert.Equal(t, 1, stats.ActiveConnections)
		assert.Equal(t, 0, stats.IdleConnections)
		assert.Equal(t, 1, stats.TotalCreated)

		// Put it back
		err = pool.Put(conn1)
		assert.NoError(t, err)

		stats = pool.Stats()
		assert.Equal(t, 0, stats.ActiveConnections)
		assert.Equal(t, 1, stats.IdleConnections)
		assert.Equal(t, 1, stats.TotalCreated)
	})

	t.Run("Connection timeout", func(t *testing.T) {
		pool := NewConnectionPool(1, func() (interface{}, error) {
			// Simulate slow connection
			time.Sleep(200 * time.Millisecond)
			conn := &MockConnection{}
			conn.Connect()
			return conn, nil
		})
		defer pool.Close()

		// Try to get with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := pool.Get(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("Pool close with active connections", func(t *testing.T) {
		closeCount := int32(0)
		pool := NewConnectionPool(3, func() (interface{}, error) {
			conn := &MockConnection{}
			conn.Connect()
			return conn, nil
		})

		// Get some connections
		conns := make([]interface{}, 2)
		for i := 0; i < 2; i++ {
			conn, err := pool.Get(context.Background())
			require.NoError(t, err)
			conns[i] = conn
		}

		// Close pool - should close all connections
		err := pool.Close()
		assert.NoError(t, err)

		// Verify connections were closed
		for _, conn := range conns {
			mockConn := conn.(*MockConnection)
			assert.False(t, mockConn.IsConnected())
		}
	})

	t.Run("Health check", func(t *testing.T) {
		unhealthyConns := make(map[int]bool)
		var mu sync.Mutex

		pool := NewConnectionPool(3, func() (interface{}, error) {
			conn := &MockConnection{}
			conn.Connect()
			return conn, nil
		})
		defer pool.Close()

		// Set health check
		pool.SetHealthCheck(func(conn interface{}) bool {
			mockConn := conn.(*MockConnection)
			mu.Lock()
			defer mu.Unlock()
			return !unhealthyConns[mockConn.id]
		})

		// Get connections
		conn1Interface, _ := pool.Get(context.Background())
		conn1 := conn1Interface.(*MockConnection)
		conn2Interface, _ := pool.Get(context.Background())
		conn2 := conn2Interface.(*MockConnection)

		// Put them back
		pool.Put(conn1)
		pool.Put(conn2)

		// Mark conn1 as unhealthy
		mu.Lock()
		unhealthyConns[conn1.id] = true
		mu.Unlock()

		// Run health check
		pool.HealthCheck()

		// Get a connection - should get conn2
		connInterface, err := pool.Get(context.Background())
		require.NoError(t, err)
		conn := connInterface.(*MockConnection)
		assert.Equal(t, conn2.id, conn.id)
	})
}

// BenchmarkConnectionPool benchmarks connection pool operations
func BenchmarkConnectionPool(b *testing.B) {
	b.Run("Get and Put", func(b *testing.B) {
		pool := NewConnectionPool(10, func() (interface{}, error) {
			conn := &MockConnection{}
			conn.Connect()
			return conn, nil
		})
		defer pool.Close()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				conn, err := pool.Get(context.Background())
				if err != nil {
					b.Fatal(err)
				}
				pool.Put(conn)
			}
		})
	})

	b.Run("Concurrent access", func(b *testing.B) {
		pool := NewConnectionPool(50, func() (interface{}, error) {
			conn := &MockConnection{}
			conn.Connect()
			return conn, nil
		})
		defer pool.Close()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				conn, err := pool.Get(context.Background())
				if err != nil {
					b.Fatal(err)
				}
				// Simulate some work
				time.Sleep(time.Microsecond)
				pool.Put(conn)
			}
		})
	})
}

// TestConnectionPoolMetrics tests connection pool metrics
func TestConnectionPoolMetrics(t *testing.T) {
	t.Run("Request metrics", func(t *testing.T) {
		pool := NewConnectionPool(2, func() (interface{}, error) {
			conn := &MockConnection{}
			conn.Connect()
			return conn, nil
		})
		defer pool.Close()

		// Enable metrics
		pool.EnableMetrics(true)

		// Perform operations
		for i := 0; i < 10; i++ {
			conn, err := pool.Get(context.Background())
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
			pool.Put(conn)
		}

		metrics := pool.GetMetrics()
		assert.Equal(t, int64(10), metrics.TotalRequests)
		assert.Equal(t, int64(10), metrics.SuccessfulRequests)
		assert.Equal(t, int64(0), metrics.FailedRequests)
		assert.Greater(t, metrics.AverageWaitTime, time.Duration(0))
	})
}