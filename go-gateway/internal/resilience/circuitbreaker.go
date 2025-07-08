package resilience

import (
	"errors"
	"sync/atomic"
	"time"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
	
	// ErrMaxConcurrency is returned when maximum concurrent requests is exceeded
	ErrMaxConcurrency = errors.New("maximum concurrent requests exceeded")
)

// State represents the current state of the circuit breaker
type State int32

const (
	// StateClosed allows requests to pass through
	StateClosed State = iota
	// StateHalfOpen allows limited requests to test if service is recovered
	StateHalfOpen
	// StateOpen blocks all requests
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// CircuitBreaker provides circuit breaker functionality for industrial protocols
type CircuitBreaker struct {
	// Configuration
	failureThreshold int64         // Number of failures to open circuit
	timeout          time.Duration // Time to wait before transitioning to half-open
	maxConcurrency   int64         // Maximum concurrent requests
	
	// State tracking
	state           int32 // State (atomic)
	failures        int64 // Failure count (atomic)
	successes       int64 // Success count (atomic)
	lastFailureTime int64 // Unix timestamp of last failure (atomic)
	concurrentReqs  int64 // Current concurrent requests (atomic)
	
	// Statistics
	totalRequests int64 // Total requests processed (atomic)
	totalFailures int64 // Total failures (atomic)
}

// NewCircuitBreaker creates a new circuit breaker with specified parameters
func NewCircuitBreaker(failureThreshold int64, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		timeout:          timeout,
		maxConcurrency:   100, // Default max concurrency
		state:           int32(StateClosed),
	}
}

// NewCircuitBreakerWithConcurrency creates a circuit breaker with concurrency limit
func NewCircuitBreakerWithConcurrency(failureThreshold int64, timeout time.Duration, maxConcurrency int64) *CircuitBreaker {
	cb := NewCircuitBreaker(failureThreshold, timeout)
	cb.maxConcurrency = maxConcurrency
	return cb
}

// Call executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	// Check circuit state
	if !cb.canExecute() {
		return ErrCircuitOpen
	}
	
	// Check concurrency limit
	concurrent := atomic.AddInt64(&cb.concurrentReqs, 1)
	if concurrent > cb.maxConcurrency {
		atomic.AddInt64(&cb.concurrentReqs, -1)
		return ErrMaxConcurrency
	}
	
	defer atomic.AddInt64(&cb.concurrentReqs, -1)
	
	// Track total requests
	atomic.AddInt64(&cb.totalRequests, 1)
	
	// Execute function
	err := fn()
	
	// Record result
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	
	return err
}

// canExecute determines if a request can be executed based on circuit state
func (cb *CircuitBreaker) canExecute() bool {
	state := State(atomic.LoadInt32(&cb.state))
	
	switch state {
	case StateClosed:
		return true
		
	case StateOpen:
		// Check if timeout has elapsed to transition to half-open
		lastFailure := atomic.LoadInt64(&cb.lastFailureTime)
		if time.Since(time.Unix(lastFailure, 0)) >= cb.timeout {
			// Try to transition to half-open
			if atomic.CompareAndSwapInt32(&cb.state, int32(StateOpen), int32(StateHalfOpen)) {
				// Reset failure count for half-open state
				atomic.StoreInt64(&cb.failures, 0)
			}
			return true
		}
		return false
		
	case StateHalfOpen:
		// In half-open state, allow limited requests
		return atomic.LoadInt64(&cb.failures) == 0
		
	default:
		return false
	}
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	failures := atomic.AddInt64(&cb.failures, 1)
	atomic.AddInt64(&cb.totalFailures, 1)
	atomic.StoreInt64(&cb.lastFailureTime, time.Now().Unix())
	
	state := State(atomic.LoadInt32(&cb.state))
	
	// Check if we should open the circuit
	if (state == StateClosed || state == StateHalfOpen) && failures >= cb.failureThreshold {
		atomic.StoreInt32(&cb.state, int32(StateOpen))
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	state := State(atomic.LoadInt32(&cb.state))
	
	if state == StateHalfOpen {
		// Transition back to closed on success in half-open state
		atomic.StoreInt32(&cb.state, int32(StateClosed))
		atomic.StoreInt64(&cb.failures, 0)
	} else if state == StateClosed {
		// Reset failure count on success in closed state
		atomic.StoreInt64(&cb.failures, 0)
	}
	
	atomic.AddInt64(&cb.successes, 1)
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() State {
	return State(atomic.LoadInt32(&cb.state))
}

// GetStatistics returns current statistics
func (cb *CircuitBreaker) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"state":              cb.GetState().String(),
		"total_requests":     atomic.LoadInt64(&cb.totalRequests),
		"total_failures":     atomic.LoadInt64(&cb.totalFailures),
		"current_failures":   atomic.LoadInt64(&cb.failures),
		"successes":         atomic.LoadInt64(&cb.successes),
		"concurrent_requests": atomic.LoadInt64(&cb.concurrentReqs),
		"failure_threshold":  cb.failureThreshold,
		"timeout_seconds":    cb.timeout.Seconds(),
		"max_concurrency":    cb.maxConcurrency,
	}
}

// Reset resets the circuit breaker to initial state
func (cb *CircuitBreaker) Reset() {
	atomic.StoreInt32(&cb.state, int32(StateClosed))
	atomic.StoreInt64(&cb.failures, 0)
	atomic.StoreInt64(&cb.successes, 0)
	atomic.StoreInt64(&cb.lastFailureTime, 0)
	atomic.StoreInt64(&cb.totalRequests, 0)
	atomic.StoreInt64(&cb.totalFailures, 0)
	atomic.StoreInt64(&cb.concurrentReqs, 0)
}

// ForceOpen manually opens the circuit breaker
func (cb *CircuitBreaker) ForceOpen() {
	atomic.StoreInt32(&cb.state, int32(StateOpen))
	atomic.StoreInt64(&cb.lastFailureTime, time.Now().Unix())
}

// ForceClose manually closes the circuit breaker
func (cb *CircuitBreaker) ForceClose() {
	atomic.StoreInt32(&cb.state, int32(StateClosed))
	atomic.StoreInt64(&cb.failures, 0)
}