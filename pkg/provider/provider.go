// Package provider defines the Provider interface for LLM backends.
package provider

import "context"

// Options configures the LLM message generation.
type Options struct {
	Style       string // "conventional" | "simple" | "emoji"
	MaxLength   int    // max chars in subject line
	Lang        string // language for commit message
	Context     string // optional user-supplied hint
	IncludeBody bool   // include multi-line body
}

type StreamEvent struct {
	Text string
	Err  error
}

// Provider is the interface that any LLM backend must implement.
type Provider interface {
	// GenerateMessage sends the diff to the LLM and returns a commit message.
	// Implementations must respect ctx for cancellation.
	GenerateMessage(ctx context.Context, diff string, opts Options) (string, error)

	GenerateMessageStream(ctx context.Context, diff string, opts Options) <-chan StreamEvent
	// Name returns a human-readable identifier used in error messages and config.
	Name() string
}

// APIError represents an error from an LLM provider with a user-facing message.
type APIError struct {
	StatusCode int
	Message    string
	Provider   string
}

func (e *APIError) Error() string {
	return e.Message
}

// UserMessage returns a clean, user-facing error message.
func (e *APIError) UserMessage() string {
	switch e.StatusCode {
	case 401:
		return "Invalid API key. Check your key at platform.openai.com"
	case 429:
		return "Rate limited by " + e.Provider + ". Please wait and try again."
	case 404:
		return "Model not found. Check available models or update: gitai config set model gpt-4o-mini"
	default:
		return e.Message
	}
}
