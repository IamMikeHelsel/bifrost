package cloud

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"go.uber.org/zap"
)

// RetryStrategy defines different retry strategies
type RetryStrategy string

const (
	RetryStrategyFixed      RetryStrategy = "fixed"
	RetryStrategyLinear     RetryStrategy = "linear"
	RetryStrategyExponential RetryStrategy = "exponential"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxRetries    int           `yaml:"max_retries"`
	InitialDelay  time.Duration `yaml:"initial_delay"`
	MaxDelay      time.Duration `yaml:"max_delay"`
	Strategy      RetryStrategy `yaml:"strategy"`
	Jitter        bool          `yaml:"jitter"`
	JitterPercent float64       `yaml:"jitter_percent"`
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    5,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		Strategy:      RetryStrategyExponential,
		Jitter:        true,
		JitterPercent: 0.1,
	}
}

// RetryManager handles retry logic for cloud operations
type RetryManager struct {
	logger *zap.Logger
	config *RetryConfig
}

// NewRetryManager creates a new retry manager
func NewRetryManager(logger *zap.Logger, config *RetryConfig) *RetryManager {
	if config == nil {
		config = DefaultRetryConfig()
	}
	
	return &RetryManager{
		logger: logger,
		config: config,
	}
}

// RetryOperation represents an operation that can be retried
type RetryOperation func(ctx context.Context) error

// Execute executes an operation with retry logic
func (r *RetryManager) Execute(ctx context.Context, operation RetryOperation, operationName string) error {
	var lastErr error
	
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Execute the operation
		err := operation(ctx)
		if err == nil {
			// Success
			if attempt > 0 {
				r.logger.Info("Operation succeeded after retries",
					zap.String("operation", operationName),
					zap.Int("attempts", attempt+1))
			}
			return nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if cloudErr, ok := err.(*CloudError); ok && !cloudErr.Retryable {
			r.logger.Error("Non-retryable error, giving up",
				zap.String("operation", operationName),
				zap.Error(err),
				zap.Int("attempt", attempt+1))
			return err
		}
		
		// If this was the last attempt, return the error
		if attempt == r.config.MaxRetries {
			r.logger.Error("Max retries exceeded",
				zap.String("operation", operationName),
				zap.Error(err),
				zap.Int("totalAttempts", attempt+1))
			return fmt.Errorf("max retries (%d) exceeded for operation %s: %w", 
				r.config.MaxRetries, operationName, err)
		}
		
		// Calculate delay for next attempt
		delay := r.calculateDelay(attempt)
		
		r.logger.Warn("Operation failed, retrying",
			zap.String("operation", operationName),
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.Duration("nextRetryIn", delay))
		
		// Wait before retry
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Continue with retry
		}
	}
	
	return lastErr
}

// calculateDelay calculates the delay for the next retry attempt
func (r *RetryManager) calculateDelay(attempt int) time.Duration {
	var delay time.Duration
	
	switch r.config.Strategy {
	case RetryStrategyFixed:
		delay = r.config.InitialDelay
		
	case RetryStrategyLinear:
		delay = r.config.InitialDelay * time.Duration(attempt+1)
		
	case RetryStrategyExponential:
		// Exponential backoff: delay = initial_delay * (2^attempt)
		multiplier := math.Pow(2, float64(attempt))
		delay = time.Duration(float64(r.config.InitialDelay) * multiplier)
		
	default:
		delay = r.config.InitialDelay
	}
	
	// Apply maximum delay cap
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}
	
	// Apply jitter if enabled
	if r.config.Jitter && r.config.JitterPercent > 0 {
		jitterRange := float64(delay) * r.config.JitterPercent
		jitter := rand.Float64()*jitterRange*2 - jitterRange // Random value between -jitterRange and +jitterRange
		delay = time.Duration(float64(delay) + jitter)
		
		// Ensure delay is not negative
		if delay < 0 {
			delay = r.config.InitialDelay
		}
	}
	
	return delay
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	logger           *zap.Logger
	config           *CircuitBreakerConfig
	state            CircuitState
	consecutiveFailures int
	lastFailureTime     time.Time
	halfOpenAttempts    int
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitStateClosed CircuitState = iota
	CircuitStateOpen
	CircuitStateHalfOpen
)

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	MaxFailures     int           `yaml:"max_failures"`
	ResetTimeout    time.Duration `yaml:"reset_timeout"`
	MaxHalfOpenCalls int          `yaml:"max_half_open_calls"`
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		MaxFailures:      5,
		ResetTimeout:     30 * time.Second,
		MaxHalfOpenCalls: 3,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(logger *zap.Logger, config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}
	
	return &CircuitBreaker{
		logger: logger,
		config: config,
		state:  CircuitStateClosed,
	}
}

// Execute executes an operation through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, operation RetryOperation, operationName string) error {
	// Check if circuit is open
	if cb.state == CircuitStateOpen {
		if time.Since(cb.lastFailureTime) < cb.config.ResetTimeout {
			return NewCloudError("circuit_open", 
				fmt.Sprintf("Circuit breaker is open for operation %s", operationName),
				"circuit_breaker", operationName, false)
		}
		
		// Transition to half-open
		cb.state = CircuitStateHalfOpen
		cb.halfOpenAttempts = 0
		cb.logger.Info("Circuit breaker transitioning to half-open",
			zap.String("operation", operationName))
	}
	
	// Check half-open limits
	if cb.state == CircuitStateHalfOpen {
		if cb.halfOpenAttempts >= cb.config.MaxHalfOpenCalls {
			return NewCloudError("circuit_half_open_limit", 
				fmt.Sprintf("Circuit breaker half-open call limit exceeded for operation %s", operationName),
				"circuit_breaker", operationName, true)
		}
		cb.halfOpenAttempts++
	}
	
	// Execute operation
	err := operation(ctx)
	
	if err != nil {
		cb.recordFailure(operationName)
		return err
	}
	
	// Success
	cb.recordSuccess(operationName)
	return nil
}

// recordFailure records a failure
func (cb *CircuitBreaker) recordFailure(operationName string) {
	cb.consecutiveFailures++
	cb.lastFailureTime = time.Now()
	
	if cb.state == CircuitStateHalfOpen {
		// Go back to open
		cb.state = CircuitStateOpen
		cb.logger.Warn("Circuit breaker opening from half-open state",
			zap.String("operation", operationName),
			zap.Int("consecutiveFailures", cb.consecutiveFailures))
		return
	}
	
	if cb.consecutiveFailures >= cb.config.MaxFailures {
		cb.state = CircuitStateOpen
		cb.logger.Warn("Circuit breaker opening",
			zap.String("operation", operationName),
			zap.Int("consecutiveFailures", cb.consecutiveFailures))
	}
}

// recordSuccess records a success
func (cb *CircuitBreaker) recordSuccess(operationName string) {
	cb.consecutiveFailures = 0
	
	if cb.state == CircuitStateHalfOpen {
		// Transition back to closed
		cb.state = CircuitStateClosed
		cb.logger.Info("Circuit breaker closing from half-open state",
			zap.String("operation", operationName))
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	return cb.state
}

// ResilienceManager combines retry logic and circuit breaker
type ResilienceManager struct {
	retryManager   *RetryManager
	circuitBreaker *CircuitBreaker
	logger         *zap.Logger
}

// NewResilienceManager creates a new resilience manager
func NewResilienceManager(logger *zap.Logger, retryConfig *RetryConfig, cbConfig *CircuitBreakerConfig) *ResilienceManager {
	return &ResilienceManager{
		retryManager:   NewRetryManager(logger, retryConfig),
		circuitBreaker: NewCircuitBreaker(logger, cbConfig),
		logger:         logger,
	}
}

// Execute executes an operation with both retry and circuit breaker protection
func (rm *ResilienceManager) Execute(ctx context.Context, operation RetryOperation, operationName string) error {
	return rm.retryManager.Execute(ctx, func(ctx context.Context) error {
		return rm.circuitBreaker.Execute(ctx, operation, operationName)
	}, operationName)
}