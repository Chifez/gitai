package mock

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Chifez/gitai/pkg/provider"
)

func TestGenerateMessage_Success(t *testing.T) {
	m := &MockProvider{Message: "feat: test commit"}
	msg, err := m.GenerateMessage(context.Background(), "diff", provider.Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "feat: test commit" {
		t.Errorf("expected 'feat: test commit', got '%s'", msg)
	}
	if m.CallCount != 1 {
		t.Errorf("expected CallCount 1, got %d", m.CallCount)
	}
}

func TestGenerateMessage_Error(t *testing.T) {
	m := &MockProvider{Err: errors.New("test error")}
	_, err := m.GenerateMessage(context.Background(), "diff", provider.Options{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got '%s'", err.Error())
	}
}

func TestGenerateMessageStream_Success(t *testing.T) {
	m := &MockProvider{
		Message:    "feat: add streaming",
		ChunkDelay: 1 * time.Millisecond,
	}
	ch := m.GenerateMessageStream(context.Background(), "diff", provider.Options{})

	var sb strings.Builder
	for event := range ch {
		if event.Err != nil {
			t.Fatalf("unexpected error: %v", event.Err)
		}
		sb.WriteString(event.Text)
	}

	if sb.String() != "feat: add streaming" {
		t.Errorf("expected 'feat: add streaming', got '%s'", sb.String())
	}
	if m.CallCount != 1 {
		t.Errorf("expected CallCount 1, got %d", m.CallCount)
	}
}

func TestGenerateMessageStream_MultiLine(t *testing.T) {
	m := &MockProvider{
		Message:    "feat: add feature\n\nThis is the body.",
		ChunkDelay: 1 * time.Millisecond,
	}
	ch := m.GenerateMessageStream(context.Background(), "diff", provider.Options{})

	var sb strings.Builder
	for event := range ch {
		if event.Err != nil {
			t.Fatalf("unexpected error: %v", event.Err)
		}
		sb.WriteString(event.Text)
	}

	if sb.String() != "feat: add feature\n\nThis is the body." {
		t.Errorf("expected multiline message, got '%s'", sb.String())
	}
}

func TestGenerateMessageStream_Error(t *testing.T) {
	m := &MockProvider{
		Err:        errors.New("stream error"),
		ChunkDelay: 1 * time.Millisecond,
	}
	ch := m.GenerateMessageStream(context.Background(), "diff", provider.Options{})

	event := <-ch
	if event.Err == nil {
		t.Fatal("expected error, got nil")
	}
	if event.Err.Error() != "stream error" {
		t.Errorf("expected 'stream error', got '%s'", event.Err.Error())
	}
}

func TestGenerateMessageStream_ContextCancel(t *testing.T) {
	m := &MockProvider{
		Message:    "feat: this is a long message that should be cancelled",
		ChunkDelay: 100 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	ch := m.GenerateMessageStream(ctx, "diff", provider.Options{})

	// Read one chunk then cancel
	<-ch
	cancel()

	// Drain remaining — should stop quickly
	for range ch {
	}
}

func TestName(t *testing.T) {
	m := &MockProvider{}
	if m.Name() != "mock" {
		t.Errorf("expected 'mock', got '%s'", m.Name())
	}
}
