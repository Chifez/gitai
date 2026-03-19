// Package openai implements the Provider interface for OpenAI's API.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Chifez/gitai/pkg/prompt"
	"github.com/Chifez/gitai/pkg/provider"
)

const (
	apiURL         = "https://api.openai.com/v1/chat/completions"
	requestTimeout = 30 * time.Second
	retryDelay     = 3 * time.Second
	rateLimitDelay = 10 * time.Second
	maxRetries     = 1
)

// OpenAI implements the Provider interface using OpenAI's Chat Completions API.
type OpenAI struct {
	apiKey string
	model  string
	client *http.Client
}

// New creates a new OpenAI provider.
func New(apiKey, model string) *OpenAI {
	return &OpenAI{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: requestTimeout},
	}
}

// Name returns the provider identifier.
func (o *OpenAI) Name() string {
	return "openai"
}

// chatRequest is the OpenAI API request body.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

// chatMessage is a single message in the OpenAI API request.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the OpenAI API response body.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// GenerateMessage sends the diff to OpenAI and returns a commit message.
func (o *OpenAI) GenerateMessage(ctx context.Context, diff string, opts provider.Options) (string, error) {
	promptOpts := prompt.Options{
		Style:       opts.Style,
		MaxLength:   opts.MaxLength,
		Lang:        opts.Lang,
		Context:     opts.Context,
		IncludeBody: opts.IncludeBody,
	}
	promptText := prompt.BuildPrompt(diff, promptOpts)

	reqBody := chatRequest{
		Model: o.model,
		Messages: []chatMessage{
			{Role: "user", Content: promptText},
		},
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		result, statusCode, err := o.doRequest(ctx, reqBody)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Handle rate limiting
		if statusCode == 429 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(rateLimitDelay):
			}
			continue
		}

		// Retry on 5xx server errors
		if statusCode >= 500 {
			continue
		}

		// Non-retryable errors
		return "", err
	}

	return "", fmt.Errorf("openai: request failed after retries: %w", lastErr)
}

// doRequest makes a single HTTP request to the OpenAI API.
func (o *OpenAI) doRequest(ctx context.Context, reqBody chatRequest) (string, int, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("openai: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", 0, fmt.Errorf("openai: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("openai: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("openai: failed to read response: %w", err)
	}

	// Handle HTTP error codes
	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode, &provider.APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("OpenAI API returned status %d: %s", resp.StatusCode, string(body)),
			Provider:   "OpenAI",
		}
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", resp.StatusCode, fmt.Errorf("openai: failed to parse response: %w", err)
	}

	if chatResp.Error != nil {
		return "", resp.StatusCode, &provider.APIError{
			StatusCode: resp.StatusCode,
			Message:    chatResp.Error.Message,
			Provider:   "OpenAI",
		}
	}

	if len(chatResp.Choices) == 0 {
		return "", resp.StatusCode, fmt.Errorf("openai: no choices in response")
	}

	return chatResp.Choices[0].Message.Content, resp.StatusCode, nil
}
