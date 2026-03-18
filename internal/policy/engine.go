package policy

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"gopkg.in/yaml.v3"
)

// Engine manages the loading and hot-reloading of domain policies.
type Engine struct {
	mu       sync.RWMutex
	policies PolicyConfig
	path     string
}

// NewEngine creates a new policy engine and loads the initial configuration.
func NewEngine(configPath string) (*Engine, error) {
	e := &Engine{
		path: configPath,
	}

	if err := e.Reload(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	return e, nil
}

// Reload reads the policy configuration file from disk.
func (e *Engine) Reload() error {
	data, err := os.ReadFile(e.path)
	if err != nil {
		return err
	}

	var config PolicyConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	e.mu.Lock()
	e.policies = config
	e.mu.Unlock()

	return nil
}

// GetPolicy returns the policy for a given domain, or the "general" policy as fallback.
func (e *Engine) GetPolicy(domain string) Policy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	p, ok := e.policies[domain]
	if !ok {
		// Fallback to "general"
		return e.policies["general"]
	}
	return p
}

// WatchSIGHUP starts a goroutine to listen for SIGHUP signals and trigger a reload.
// This allows updating policies without stopping the service.
func (e *Engine) WatchSIGHUP() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)

	go func() {
		for range sigs {
			if err := e.Reload(); err != nil {
				fmt.Fprintf(os.Stderr, "Error reloading policies on SIGHUP: %v\n", err)
			} else {
				fmt.Println("Policies reloaded successfully via SIGHUP")
			}
		}
	}()
}
