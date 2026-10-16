// Package llm — адаптер LLM-порта агента для OpenAI-совместимых chat completion API.
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

	"github.com/ValievM/family-expense-bot/internal/ports"
)

const (
	defaultTimeout = 60 * time.Second
	maxRetries     = 2
	maxErrorBody   = 512
)

type APIError struct {
	Status  int
	Message string
}

func (err *APIError) Error() string {
	return fmt.Sprintf("llm provider returned %d: %s", err.Status, err.Message)
}

type Client struct {
	baseURL    string
	model      string
	authorizer Authorizer
	httpClient *http.Client
	backoff    func(attempt int) time.Duration
}

type Option func(*Client)

func WithBackoff(backoff func(attempt int) time.Duration) Option {
	return func(client *Client) { client.backoff = backoff }
}

// Принимает: корень API, имя модели, способ авторизации и дополнительные опции.
// Делает: обрезает завершающие слеши baseURL, ставит HTTP-таймаут 60 с и линейный backoff.
// Возвращает: готовый *Client.
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

// Принимает: ctx — контекст; request — сообщения и описания инструментов.
// Делает: отправляет диалог провайдеру; при 429 и 5xx повторяет до maxRetries раз с паузой.
// Возвращает: ответ ассистента или последнюю ошибку; ошибку контекста при отмене ожидания.
func (client *Client) Complete(ctx context.Context, request ports.Request) (ports.Message, error) {
	payload, err := json.Marshal(newChatRequest(client.model, request))
	if err != nil {
		return ports.Message{}, fmt.Errorf("encode llm request: %w", err)
	}

	return withRetries(ctx, client.backoff, retryable, func() (ports.Message, error) {
		return client.send(ctx, payload)
	})
}

// Принимает: ctx — контекст; payload — JSON тела запроса.
// Делает: один POST на /chat/completions с авторизацией; при ошибке читает до maxErrorBody байт тела.
// Возвращает: сообщение ассистента, *APIError при неуспешном статусе или ошибку вызова и разбора.
func (client *Client) send(ctx context.Context, payload []byte) (ports.Message, error) {
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return ports.Message{}, fmt.Errorf("build llm request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if err := client.authorizer.Authorize(ctx, httpRequest); err != nil {
		return ports.Message{}, fmt.Errorf("authorize llm request: %w", err)
	}

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return ports.Message{}, fmt.Errorf("call llm provider: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBody))
		return ports.Message{}, &APIError{Status: response.StatusCode, Message: strings.TrimSpace(string(body))}
	}

	var decoded chatResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return ports.Message{}, fmt.Errorf("decode llm response: %w", err)
	}
	return decoded.assistantMessage()
}

// Принимает: err — ошибка попытки.
// Делает: разрешает повтор только для *APIError со статусом 429 или 5xx.
// Возвращает: true, если повтор допустим.
func retryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if isOutOfFunds(apiErr.Message) {
			return false
		}
		return apiErr.Status == http.StatusTooManyRequests || apiErr.Status >= http.StatusInternalServerError
	}
	return false
}

// Принимает: message — текст ошибки провайдера.
// Делает: узнаёт окончательные отказы «нет денег или квоты», которые провайдеры отдают с кодом 429; повтор их не лечит.
// Возвращает: true, если ждать и повторять бессмысленно.
func isOutOfFunds(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "insufficient") || strings.Contains(lower, "quota") || strings.Contains(lower, "recharge")
}

// Принимает: ctx — контекст; duration — длительность паузы.
// Делает: ждёт указанное время или отмену контекста.
// Возвращает: nil после паузы или ctx.Err() при отмене.
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

var _ ports.LLM = (*Client)(nil)
