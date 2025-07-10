package protocols

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Custom error types for testing
type ProtocolError struct {
	Code    int
	Message string
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("Protocol error %d: %s", e.Code, e.Message)
}

type ConnectionError struct {
	Host   string
	Port   int
	Reason string
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("Connection to %s:%d failed: %s", e.Host, e.Port, e.Reason)
}

// TestErrorHandling tests comprehensive error handling scenarios
func TestErrorHandling(t *testing.T) {
	logger := zap.NewNop()

	t.Run("Network errors", func(t *testing.T) {
		tests := []struct {
			name          string
			err           error
			shouldRetry   bool
			expectedType  string
		}{
			{
				name:          "Timeout error",
				err:           &net.OpError{Op: "dial", Err: errors.New("i/o timeout")},
				shouldRetry:   true,
				expectedType:  "timeout",
			},
			{
				name:          "Connection refused",
				err:           &net.OpError{Op: "dial", Err: errors.New("connection refused")},
				shouldRetry:   true,
				expectedType:  "connection",
			},
			{
				name:          "Network unreachable",
				err:           &net.OpError{Op: "dial", Err: errors.New("network is unreachable")},
				shouldRetry:   false,
				expectedType:  "network",
			},
			{
				name:          "DNS error",
				err:           &net.DNSError{Err: "no such host", Name: "invalid.host"},
				shouldRetry:   false,
				expectedType:  "dns",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				handler := &ErrorHandler{logger: logger}
				
				result := handler.HandleError(tt.err)
				assert.Equal(t, tt.shouldRetry, result.ShouldRetry)
				assert.Equal(t, tt.expectedType, result.ErrorType)
			})
		}
	})

	t.Run("Protocol errors", func(t *testing.T) {
		handler := &ErrorHandler{logger: logger}

		// Test Modbus exception codes
		modbusErrors := []struct {
			code     int
			message  string
			severity string
		}{
			{1, "Illegal function", "error"},
			{2, "Illegal data address", "error"},
			{3, "Illegal data value", "error"},
			{4, "Slave device failure", "critical"},
			{6, "Slave device busy", "warning"},
		}

		for _, me := range modbusErrors {
			err := &ProtocolError{Code: me.code, Message: me.message}
			result := handler.HandleError(err)
			assert.Equal(t, me.severity, result.Severity)
		}
	})

	t.Run("Context errors", func(t *testing.T) {
		handler := &ErrorHandler{logger: logger}

		// Test context cancellation
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		
		result := handler.HandleError(ctx.Err())
		assert.False(t, result.ShouldRetry)
		assert.Equal(t, "cancelled", result.ErrorType)

		// Test context timeout
		ctx, cancel = context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		time.Sleep(time.Millisecond)
		
		result = handler.HandleError(ctx.Err())
		assert.True(t, result.ShouldRetry)
		assert.Equal(t, "timeout", result.ErrorType)
	})

	t.Run("Error wrapping and unwrapping", func(t *testing.T) {
		baseErr := &ConnectionError{
			Host:   "192.168.1.100",
			Port:   502,
			Reason: "connection timeout",
		}

		// Wrap error multiple times
		wrapped := fmt.Errorf("failed to connect: %w", baseErr)
		wrapped = fmt.Errorf("device communication error: %w", wrapped)

		// Test unwrapping
		var connErr *ConnectionError
		assert.True(t, errors.As(wrapped, &connErr))
		assert.Equal(t, baseErr.Host, connErr.Host)
		assert.Equal(t, baseErr.Port, connErr.Port)
	})

	t.Run("Retry logic", func(t *testing.T) {
		retrier := &RetryHandler{
			MaxRetries:     3,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     1 * time.Second,
			BackoffFactor:  2,
		}

		attemptCount := 0
		operation := func() error {
			attemptCount++
			if attemptCount < 3 {
				return &net.OpError{Op: "dial", Err: errors.New("connection refused")}
			}
			return nil
		}

		err := retrier.RetryWithBackoff(context.Background(), operation)
		assert.NoError(t, err)
		assert.Equal(t, 3, attemptCount)
	})

	t.Run("Retry with non-retryable error", func(t *testing.T) {
		retrier := &RetryHandler{
			MaxRetries:     3,
			InitialBackoff: 10 * time.Millisecond,
		}

		attemptCount := 0
		operation := func() error {
			attemptCount++
			return &ProtocolError{Code: 2, Message: "Illegal data address"}
		}

		err := retrier.RetryWithBackoff(context.Background(), operation)
		assert.Error(t, err)
		assert.Equal(t, 1, attemptCount) // Should not retry
	})

	t.Run("Circuit breaker", func(t *testing.T) {
		breaker := &CircuitBreaker{
			FailureThreshold: 3,
			ResetTimeout:     100 * time.Millisecond,
			HalfOpenRequests: 1,
		}

		// Cause failures to open circuit
		for i := 0; i < 3; i++ {
			err := breaker.Call(func() error {
				return errors.New("service error")
			})
			assert.Error(t, err)
		}

		// Circuit should be open
		assert.Equal(t, "open", breaker.State())

		// Calls should fail immediately
		err := breaker.Call(func() error {
			return nil
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker is open")

		// Wait for reset timeout
		time.Sleep(150 * time.Millisecond)

		// Circuit should be half-open, one successful call should close it
		err = breaker.Call(func() error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, "closed", breaker.State())
	})

	t.Run("Error aggregation", func(t *testing.T) {
		aggregator := &ErrorAggregator{}

		// Add various errors
		errors := []error{
			&ConnectionError{Host: "192.168.1.100", Port: 502, Reason: "timeout"},
			&ConnectionError{Host: "192.168.1.101", Port: 502, Reason: "refused"},
			&ProtocolError{Code: 3, Message: "Illegal data value"},
			&ConnectionError{Host: "192.168.1.100", Port: 502, Reason: "timeout"},
		}

		for _, err := range errors {
			aggregator.Add(err)
		}

		// Get aggregated report
		report := aggregator.Report()
		assert.Equal(t, 4, report.TotalErrors)
		assert.Equal(t, 2, report.UniqueErrors)
		assert.Equal(t, 2, report.ErrorCounts["ConnectionError: timeout"])
	})

	t.Run("Panic recovery", func(t *testing.T) {
		handler := &PanicHandler{logger: logger}

		// Function that panics
		riskyOperation := func() (err error) {
			defer handler.Recover(&err)
			panic("unexpected error")
		}

		err := riskyOperation()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "panic recovered")
	})

	t.Run("Error rate limiting", func(t *testing.T) {
		limiter := &ErrorRateLimiter{
			MaxErrorsPerMinute: 10,
			Window:             time.Minute,
		}

		// Generate errors quickly
		for i := 0; i < 15; i++ {
			shouldLog := limiter.ShouldLog(errors.New("test error"))
			if i < 10 {
				assert.True(t, shouldLog)
			} else {
				assert.False(t, shouldLog)
			}
		}
	})

	t.Run("Structured error logging", func(t *testing.T) {
		// Create a test logger that captures output
		logger, _ := zap.NewDevelopment()
		handler := &ErrorHandler{logger: logger}

		err := &ConnectionError{
			Host:   "192.168.1.100",
			Port:   502,
			Reason: "connection timeout",
		}

		// Log structured error
		handler.LogError("failed to read tag", err, map[string]interface{}{
			"device_id": "modbus-001",
			"tag_name":  "temperature",
			"attempt":   3,
		})

		// In real test, we would capture and verify log output
	})

	t.Run("Error metrics", func(t *testing.T) {
		metrics := &ErrorMetrics{}

		// Track various errors
		errors := []error{
			&ConnectionError{Host: "192.168.1.100", Port: 502, Reason: "timeout"},
			&ProtocolError{Code: 3, Message: "Illegal data value"},
			context.DeadlineExceeded,
			&ConnectionError{Host: "192.168.1.101", Port: 502, Reason: "refused"},
		}

		for _, err := range errors {
			metrics.Record(err)
		}

		stats := metrics.GetStats()
		assert.Equal(t, int64(4), stats.TotalErrors)
		assert.Equal(t, int64(2), stats.ConnectionErrors)
		assert.Equal(t, int64(1), stats.ProtocolErrors)
		assert.Equal(t, int64(1), stats.TimeoutErrors)
	})
}

// Error handling helper structures

type ErrorHandler struct {
	logger *zap.Logger
}

type ErrorResult struct {
	ShouldRetry bool
	ErrorType   string
	Severity    string
}

func (h *ErrorHandler) HandleError(err error) ErrorResult {
	result := ErrorResult{
		ShouldRetry: false,
		ErrorType:   "unknown",
		Severity:    "error",
	}

	switch e := err.(type) {
	case *net.OpError:
		result.ErrorType = "network"
		if e.Timeout() {
			result.ErrorType = "timeout"
			result.ShouldRetry = true
		} else if e.Err != nil && e.Err.Error() == "connection refused" {
			result.ErrorType = "connection"
			result.ShouldRetry = true
		}
	case *net.DNSError:
		result.ErrorType = "dns"
		result.Severity = "critical"
	case *ProtocolError:
		result.ErrorType = "protocol"
		if e.Code == 6 { // Device busy
			result.Severity = "warning"
			result.ShouldRetry = true
		} else if e.Code == 4 { // Device failure
			result.Severity = "critical"
		}
	default:
		if errors.Is(err, context.Canceled) {
			result.ErrorType = "cancelled"
		} else if errors.Is(err, context.DeadlineExceeded) {
			result.ErrorType = "timeout"
			result.ShouldRetry = true
		}
	}

	return result
}

func (h *ErrorHandler) LogError(message string, err error, fields map[string]interface{}) {
	zapFields := make([]zap.Field, 0, len(fields)+1)
	zapFields = append(zapFields, zap.Error(err))
	
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	
	h.logger.Error(message, zapFields...)
}

type RetryHandler struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

func (r *RetryHandler) RetryWithBackoff(ctx context.Context, operation func() error) error {
	backoff := r.InitialBackoff
	
	for attempt := 0; attempt < r.MaxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		// Check if error is retryable
		handler := &ErrorHandler{}
		result := handler.HandleError(err)
		if !result.ShouldRetry {
			return err
		}

		if attempt < r.MaxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff = time.Duration(float64(backoff) * r.BackoffFactor)
				if backoff > r.MaxBackoff {
					backoff = r.MaxBackoff
				}
			}
		}
	}

	return fmt.Errorf("max retries exceeded")
}

type CircuitBreaker struct {
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenRequests int
	
	failures         int
	lastFailureTime  time.Time
	state            string
	halfOpenAttempts int
}

func (cb *CircuitBreaker) Call(operation func() error) error {
	if cb.state == "open" {
		if time.Since(cb.lastFailureTime) > cb.ResetTimeout {
			cb.state = "half-open"
			cb.halfOpenAttempts = 0
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	err := operation()
	
	if err != nil {
		cb.failures++
		cb.lastFailureTime = time.Now()
		
		if cb.failures >= cb.FailureThreshold {
			cb.state = "open"
		}
		return err
	}

	// Success
	if cb.state == "half-open" {
		cb.halfOpenAttempts++
		if cb.halfOpenAttempts >= cb.HalfOpenRequests {
			cb.state = "closed"
			cb.failures = 0
		}
	} else {
		cb.failures = 0
	}

	return nil
}

func (cb *CircuitBreaker) State() string {
	if cb.state == "" {
		return "closed"
	}
	return cb.state
}

type ErrorAggregator struct {
	errors map[string]int
}

func (ea *ErrorAggregator) Add(err error) {
	if ea.errors == nil {
		ea.errors = make(map[string]int)
	}
	
	key := fmt.Sprintf("%T", err)
	if connErr, ok := err.(*ConnectionError); ok {
		key = fmt.Sprintf("ConnectionError: %s", connErr.Reason)
	}
	
	ea.errors[key]++
}

type ErrorReport struct {
	TotalErrors  int
	UniqueErrors int
	ErrorCounts  map[string]int
}

func (ea *ErrorAggregator) Report() ErrorReport {
	report := ErrorReport{
		ErrorCounts: make(map[string]int),
	}
	
	for errType, count := range ea.errors {
		report.TotalErrors += count
		report.ErrorCounts[errType] = count
	}
	
	report.UniqueErrors = len(ea.errors)
	return report
}

type PanicHandler struct {
	logger *zap.Logger
}

func (ph *PanicHandler) Recover(err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("panic recovered: %v", r)
		ph.logger.Error("Panic recovered", zap.Any("panic", r))
	}
}

type ErrorRateLimiter struct {
	MaxErrorsPerMinute int
	Window             time.Duration
	errorTimes         []time.Time
}

func (erl *ErrorRateLimiter) ShouldLog(err error) bool {
	now := time.Now()
	
	// Remove old entries
	cutoff := now.Add(-erl.Window)
	newTimes := []time.Time{}
	for _, t := range erl.errorTimes {
		if t.After(cutoff) {
			newTimes = append(newTimes, t)
		}
	}
	erl.errorTimes = newTimes
	
	// Check if we can log
	if len(erl.errorTimes) >= erl.MaxErrorsPerMinute {
		return false
	}
	
	erl.errorTimes = append(erl.errorTimes, now)
	return true
}

type ErrorMetrics struct {
	totalErrors      int64
	connectionErrors int64
	protocolErrors   int64
	timeoutErrors    int64
}

func (em *ErrorMetrics) Record(err error) {
	em.totalErrors++
	
	switch err.(type) {
	case *ConnectionError:
		em.connectionErrors++
	case *ProtocolError:
		em.protocolErrors++
	default:
		if errors.Is(err, context.DeadlineExceeded) {
			em.timeoutErrors++
		}
	}
}

type ErrorStats struct {
	TotalErrors      int64
	ConnectionErrors int64
	ProtocolErrors   int64
	TimeoutErrors    int64
}

func (em *ErrorMetrics) GetStats() ErrorStats {
	return ErrorStats{
		TotalErrors:      em.totalErrors,
		ConnectionErrors: em.connectionErrors,
		ProtocolErrors:   em.protocolErrors,
		TimeoutErrors:    em.timeoutErrors,
	}
}