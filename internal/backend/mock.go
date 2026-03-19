package backend

import (
	"context"
	"time"
)

// MockBackend simulating an LLM service with artificial delay.
type MockBackend struct {
	Response string
	Model    string
	Delay    time.Duration
}

func NewMockBackend(response, model string) *MockBackend {
	return &MockBackend{
		Response: response,
		Model:    model,
		Delay:    0,
	}
}

func (m *MockBackend) Query(ctx context.Context, query string) (*Response, error) {
	time.Sleep(m.Delay)
	return &Response{
		Answer:  m.Response,
		Model:   m.Model,
		Latency: m.Delay.Milliseconds(),
	}, nil
}
