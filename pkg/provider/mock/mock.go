// Package mock provides a mock Provider for testing.
package mock

import (
	"context"
	"strings"
	"time"

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
	// ChunkDelay is the delay between streamed chunks. Defaults to 20ms.
	ChunkDelay time.Duration
}

// GenerateMessage returns the configured message or error.
func (m *MockProvider) GenerateMessage(ctx context.Context, diff string, opts provider.Options) (string, error) {
	m.CallCount++
	if m.Err != nil {
		return "", m.Err
	}
	return m.Message, nil
}

// GenerateMessageStream streams the message in small chunks with a configurable delay,
// simulating realistic LLM streaming behavior.
func (m *MockProvider) GenerateMessageStream(ctx context.Context, diff string, opts provider.Options) <-chan provider.StreamEvent {
	m.CallCount++

	ch := make(chan provider.StreamEvent)
	delay := m.ChunkDelay
	if delay == 0 {
		delay = 20 * time.Millisecond
	}

	go func() {
		defer close(ch)

		if m.Err != nil {
			select {
			case <-ctx.Done():
			case ch <- provider.StreamEvent{Err: m.Err}:
			}
			return
		}

		// Stream line-by-line, then word-by-word within each line
		lines := strings.Split(m.Message, "\n")
		for li, line := range lines {
			if li > 0 {
				// Send the newline separator
				if !sendChunk(ctx, ch, "\n", delay) {
					return
				}
			}
			words := strings.Fields(line)
			for wi, word := range words {
				if wi > 0 {
					word = " " + word
				}
				if !sendChunk(ctx, ch, word, delay) {
					return
				}
			}
		}
	}()
	return ch
}

// sendChunk sends a text chunk on the channel after a delay. Returns false if ctx was cancelled.
func sendChunk(ctx context.Context, ch chan<- provider.StreamEvent, text string, delay time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(delay):
	}
	select {
	case <-ctx.Done():
		return false
	case ch <- provider.StreamEvent{Text: text}:
		return true
	}
}

// Name returns the mock provider name.
func (m *MockProvider) Name() string {
	return "mock"
}
