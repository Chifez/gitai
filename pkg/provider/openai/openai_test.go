package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Chifez/gitai/pkg/provider"
)

func TestGenerateMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer valid-key" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid key"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"feat: auth system"}}]}`))
	}))
	defer server.Close()

	originalURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalURL }()

	o := New("valid-key", "test-model")
	opts := provider.Options{Style: "conventional"}

	msg, err := o.GenerateMessage(context.Background(), "diff", opts)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if msg != "feat: auth system" {
		t.Errorf("expected 'feat: auth system', got '%s'", msg)
	}
}

func TestGenerateMessage_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid API key"}}`))
	}))
	defer server.Close()

	originalURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalURL }()

	o := New("bad-key", "test-model")
	originalRetry := retryDelay
	defer func() { retryDelay = originalRetry }()

	_, err := o.GenerateMessage(context.Background(), "diff", provider.Options{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*provider.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected 401, got %d", apiErr.StatusCode)
	}
	if apiErr.UserMessage() != "Invalid API key. Check your key at platform.openai.com" {
		t.Errorf("unexpected user message: %s", apiErr.UserMessage())
	}
}

func TestGenerateMessage_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	originalURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalURL }()

	o := New("key", "model")
	o.client.Timeout = 10 * time.Millisecond // force timeout

	_, err := o.GenerateMessage(context.Background(), "diff", provider.Options{})
	if err == nil {
		t.Fatal("expected error due to timeout")
	}
}

func TestGenerateMessage_RateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"feat: retry success"}}]}`))
	}))
	defer server.Close()

	originalURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalURL }()

	origRateDelay := rateLimitDelay
	rateLimitDelay = 10 * time.Millisecond
	defer func() { rateLimitDelay = origRateDelay }()

	origRetryDelay := retryDelay
	retryDelay = 10 * time.Millisecond
	defer func() { retryDelay = origRetryDelay }()

	o := New("key", "model")
	msg, err := o.GenerateMessage(context.Background(), "diff", provider.Options{})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if msg != "feat: retry success" {
		t.Errorf("expected 'feat: retry success', got '%s'", msg)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestGenerateMessage_ServerError_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"server error"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"feat: recovered"}}]}`))
	}))
	defer server.Close()

	originalURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalURL }()

	origRetryDelay := retryDelay
	retryDelay = 10 * time.Millisecond
	defer func() { retryDelay = origRetryDelay }()

	o := New("key", "model")
	msg, err := o.GenerateMessage(context.Background(), "diff", provider.Options{})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if msg != "feat: recovered" {
		t.Errorf("expected 'feat: recovered', got '%s'", msg)
	}
}

func TestGenerateMessage_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	originalURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalURL }()

	o := New("key", "model")
	_, err := o.GenerateMessage(context.Background(), "diff", provider.Options{})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestGenerateMessage_ErrorInBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":{"message":"model not found","type":"invalid_request_error","code":"model_not_found"}}`))
	}))
	defer server.Close()

	originalURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalURL }()

	o := New("key", "model")
	_, err := o.GenerateMessage(context.Background(), "diff", provider.Options{})
	if err == nil {
		t.Fatal("expected error for body error")
	}
}

func TestGenerateMessage_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	originalURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalURL }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	o := New("key", "model")
	_, err := o.GenerateMessage(ctx, "diff", provider.Options{})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestGenerateMessageStream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"feat:\"}}]}\n\n")
		flusher.Flush()
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\" streaming\"}}]}\n\n")
		flusher.Flush()
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	originalURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalURL }()

	o := New("key", "model")
	ch := o.GenerateMessageStream(context.Background(), "diff", provider.Options{})

	var sb strings.Builder
	for event := range ch {
		if event.Err != nil {
			t.Fatalf("unexpected error: %v", event.Err)
		}
		sb.WriteString(event.Text)
	}

	if sb.String() != "feat: streaming" {
		t.Errorf("expected 'feat: streaming', got '%s'", sb.String())
	}
}

func TestGenerateMessageStream_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid key"}}`))
	}))
	defer server.Close()

	originalURL := apiURL
	apiURL = server.URL
	defer func() { apiURL = originalURL }()

	origRetryDelay := retryDelay
	retryDelay = 10 * time.Millisecond
	defer func() { retryDelay = origRetryDelay }()

	o := New("bad-key", "model")
	ch := o.GenerateMessageStream(context.Background(), "diff", provider.Options{})

	event := <-ch
	if event.Err == nil {
		t.Fatal("expected error for auth failure")
	}
}

func TestName(t *testing.T) {
	o := New("key", "model")
	if o.Name() != "openai" {
		t.Errorf("expected 'openai', got '%s'", o.Name())
	}
}

func TestNew(t *testing.T) {
	o := New("sk-test", "gpt-4o")
	if o.apiKey != "sk-test" {
		t.Errorf("expected sk-test, got %s", o.apiKey)
	}
	if o.model != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", o.model)
	}
	if o.client == nil {
		t.Error("expected non-nil client")
	}
}
