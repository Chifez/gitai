package editor

import (
	"context"
	"strings"
	"testing"
)

func TestReviewMessage_Accept(t *testing.T) {
	input := strings.NewReader("y\n")
	msg, action, err := ReviewMessageWithReader(context.Background(), "feat: test", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionCommit {
		t.Errorf("expected ActionCommit, got %d", action)
	}
	if msg != "feat: test" {
		t.Errorf("expected 'feat: test', got '%s'", msg)
	}
}

func TestReviewMessage_Cancel(t *testing.T) {
	input := strings.NewReader("n\n")
	_, action, err := ReviewMessageWithReader(context.Background(), "feat: test", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionCancel {
		t.Errorf("expected ActionCancel, got %d", action)
	}
}

func TestReviewMessage_Regenerate(t *testing.T) {
	input := strings.NewReader("r\n")
	_, action, err := ReviewMessageWithReader(context.Background(), "feat: test", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionRegenerate {
		t.Errorf("expected ActionRegenerate, got %d", action)
	}
}

func TestReviewMessage_InvalidThenAccept(t *testing.T) {
	input := strings.NewReader("x\ny\n")
	msg, action, err := ReviewMessageWithReader(context.Background(), "feat: test", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionCommit {
		t.Errorf("expected ActionCommit, got %d", action)
	}
	if msg != "feat: test" {
		t.Errorf("expected 'feat: test', got '%s'", msg)
	}
}

func TestReviewMessage_CaseInsensitive(t *testing.T) {
	input := strings.NewReader("Y\n")
	_, action, err := ReviewMessageWithReader(context.Background(), "feat: test", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != ActionCommit {
		t.Errorf("expected ActionCommit, got %d", action)
	}
}

func TestReviewMessage_EOF(t *testing.T) {
	input := strings.NewReader("")
	_, action, err := ReviewMessageWithReader(context.Background(), "feat: test", input)
	if err == nil {
		t.Fatal("expected error on EOF, got nil")
	}
	if action != ActionCancel {
		t.Errorf("expected ActionCancel on error, got %d", action)
	}
}
