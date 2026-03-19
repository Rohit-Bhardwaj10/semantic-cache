package resilience

import (
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu           sync.RWMutex
	state        State
	failures     int
	maxFailures  int
	timeout      time.Duration
	lastFailure  time.Time
}

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:       StateClosed,
		maxFailures: maxFailures,
		timeout:     timeout,
	}
}

// Execute wraps a function call with circuit breaker logic.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
			cb.lastFailure = time.Now()
		}
		return err
	}

	// Success: Reset failures and close circuit
	cb.failures = 0
	cb.state = StateClosed
	return nil
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == StateClosed {
		return true
	}

	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.timeout {
			// Transition to half-open to allow a probe
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	}

	// Half-open: already probe in progress
	return false
}
