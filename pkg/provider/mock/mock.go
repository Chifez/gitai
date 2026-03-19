// Package mock provides a mock Provider for testing.
package mock

import (
	"context"

	"github.com/Chifez/gitai/pkg/provider"
)

// MockProvider is a configurable mock that implements the Provider interface.
type MockProvider struct {
	// Message is the commit message to return.
	Message string
	// Err is the error to return (if any).
	Err error
	// CallCount tracks how many times GenerateMessage was called.
	CallCount int
}

// GenerateMessage returns the configured message or error.
func (m *MockProvider) GenerateMessage(ctx context.Context, diff string, opts provider.Options) (string, error) {
	m.CallCount++
	if m.Err != nil {
		return "", m.Err
	}
	return m.Message, nil
}

// Name returns the mock provider name.
func (m *MockProvider) Name() string {
	return "mock"
}
