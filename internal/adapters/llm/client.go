// Package llm implements the agent's LLM port for OpenAI-compatible chat completion APIs.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ValievM/family-expense-bot/internal/service/agent"
)

const (
	defaultTimeout = 60 * time.Second
	maxRetries     = 2
	maxErrorBody   = 512
)

// APIError describes a non-success response from the provider.
type APIError struct {
	Status  int
	Message string
}

func (err *APIError) Error() string {
	return fmt.Sprintf("llm provider returned %d: %s", err.Status, err.Message)
}

// Client talks to any server exposing POST {base}/chat/completions with tool calling.
type Client struct {
	baseURL    string
	model      string
	authorizer Authorizer
	httpClient *http.Client
	backoff    func(attempt int) time.Duration
}

var _ agent.LLM = (*Client)(nil)

// Option customizes a Client.
type Option func(*Client)

// WithHTTPClient replaces the default HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) { client.httpClient = httpClient }
}

// WithBackoff overrides the delay between retries; tests use it to avoid sleeping.
func WithBackoff(backoff func(attempt int) time.Duration) Option {
	return func(client *Client) { client.backoff = backoff }
}

// NewClient builds a client for the given API root, for example https://api.openai.com/v1.
func NewClient(baseURL, model string, authorizer Authorizer, options ...Option) *Client {
	client := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		authorizer: authorizer,
		httpClient: &http.Client{Timeout: defaultTimeout},
		backoff:    func(attempt int) time.Duration { return time.Duration(attempt) * time.Second },
	}
	for _, option := range options {
		option(client)
	}
	return client
}

// Complete sends the conversation and returns the assistant reply, retrying rate limits and server errors.
func (client *Client) Complete(ctx context.Context, request agent.Request) (agent.Message, error) {
	payload, err := json.Marshal(newChatRequest(client.model, request))
	if err != nil {
		return agent.Message{}, fmt.Errorf("encode llm request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleep(ctx, client.backoff(attempt)); err != nil {
				return agent.Message{}, err
			}
		}
		reply, err := client.send(ctx, payload)
		if err == nil {
			return reply, nil
		}
		lastErr = err
		if !retryable(err) {
			break
		}
	}
	return agent.Message{}, lastErr
}

func (client *Client) send(ctx context.Context, payload []byte) (agent.Message, error) {
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return agent.Message{}, fmt.Errorf("build llm request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if err := client.authorizer.Authorize(ctx, httpRequest); err != nil {
		return agent.Message{}, fmt.Errorf("authorize llm request: %w", err)
	}

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return agent.Message{}, fmt.Errorf("call llm provider: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBody))
		return agent.Message{}, &APIError{Status: response.StatusCode, Message: strings.TrimSpace(string(body))}
	}

	var decoded chatResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return agent.Message{}, fmt.Errorf("decode llm response: %w", err)
	}
	return decoded.assistantMessage()
}

func retryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status == http.StatusTooManyRequests || apiErr.Status >= http.StatusInternalServerError
	}
	return false
}

func sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
