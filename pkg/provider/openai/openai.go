// Package openai implements the Provider interface for OpenAI's API.
package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Chifez/gitai/pkg/prompt"
	"github.com/Chifez/gitai/pkg/provider"
)

var apiURL = "https://api.openai.com/v1/chat/completions"

const (
	requestTimeout = 30 * time.Second
	maxRetries     = 1
)

var (
	retryDelay     = 3 * time.Second
	rateLimitDelay = 10 * time.Second
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
	Stream   bool          `json:"stream,omitempty"`
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

		if statusCode == 429 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(rateLimitDelay):
			}
			continue
		}

		if statusCode >= 500 {
			continue
		}

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

func (o *OpenAI) doStreamRequest(ctx context.Context, reqBody chatRequest) (*http.Response, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonData))

	if err != nil {
		return nil, fmt.Errorf("openai:failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("openai: request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		return nil, &provider.APIError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("OpenAI API returned status %d: %s", resp.StatusCode, string(body)), Provider: "OpenAI"}
	}

	return resp, nil
}

func (o *OpenAI) GenerateMessageStream(ctx context.Context, diff string, opts provider.Options) <-chan provider.StreamEvent {
	ch := make(chan provider.StreamEvent)

	go func() {
		defer close(ch)

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
				{
					Role:    "user",
					Content: promptText,
				},
			},
			Stream: true,
		}
		var lastErr error
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if ctx.Err() != nil {
				ch <- provider.StreamEvent{Err: ctx.Err()}
				return
			}

			if attempt > 0 {
				select {
				case <-ctx.Done():
					ch <- provider.StreamEvent{Err: ctx.Err()}
					return
				case <-time.After(retryDelay):
				}
			}
			resp, err := o.doStreamRequest(ctx, reqBody)
			if err != nil {
				lastErr = err
				if apiErr, ok := err.(*provider.APIError); ok {
					if apiErr.StatusCode == 429 {
						select {
						case <-ctx.Done():
							ch <- provider.StreamEvent{Err: ctx.Err()}
							return
						case <-time.After(rateLimitDelay):
						}
						continue
					}
					if apiErr.StatusCode >= 500 {
						continue
					}
				}
				ch <- provider.StreamEvent{Err: err}
				return
			}

			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				line := scanner.Text()

				if line == "" {
					continue
				}

				if !strings.HasPrefix(line, "data: ") {
					continue
				}

				data := strings.TrimPrefix(line, "data: ")

				if data == "[DONE]" {
					break
				}

				var streamResp struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}

				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					continue
				}
				if len(streamResp.Choices) > 0 {
					content := streamResp.Choices[0].Delta.Content
					if content != "" {
						ch <- provider.StreamEvent{Text: content}
					}
				}
			}
			resp.Body.Close()
			if err := scanner.Err(); err != nil {
				ch <- provider.StreamEvent{Err: fmt.Errorf("openai: stream read error: %w", err)}
			}
			return
		}
		ch <- provider.StreamEvent{Err: fmt.Errorf("openai: stream request failed after retries: %w", lastErr)}
	}()

	return ch
}
