package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ValievM/family-expense-bot/internal/ports"
)

const maxStreamLine = 1 << 20

// ErrNoModels означает, что аккаунту не доступна ни одна модель из списка.
var ErrNoModels = errors.New("no models are available for this account")

// ResponsesClient общается с Responses API (POST {base}/responses) в потоковом режиме; так работает оплата подпиской ChatGPT.
type ResponsesClient struct {
	baseURL    string
	authorizer Authorizer
	httpClient *http.Client
	backoff    func(attempt int) time.Duration

	modelMutex sync.Mutex
	model      string
}

// Проверка на этапе компиляции: ResponsesClient реализует порт LLM.
var _ ports.LLM = (*ResponsesClient)(nil)

// Принимает: baseURL — корень API; model — slug модели, пустая строка означает «взять первую доступную аккаунту»; authorizer — выдаёт заголовки авторизации; options — настройки.
// Делает: создаёт клиента с тем же набором настроек, что и у Client.
// Возвращает: ResponsesClient.
func NewResponsesClient(baseURL, model string, authorizer Authorizer, options ...Option) *ResponsesClient {
	base := NewClient(baseURL, model, authorizer, options...)
	return &ResponsesClient{
		baseURL: base.baseURL, model: model, authorizer: authorizer,
		httpClient: base.httpClient, backoff: base.backoff,
	}
}

// Принимает: ctx — контекст; request — разговор и инструменты.
// Делает: выбирает модель, отправляет запрос потоком и собирает ответ ассистента с повтором при временных сбоях.
// Возвращает: сообщение ассистента (текст и/или вызовы инструментов) либо ошибку.
func (client *ResponsesClient) Complete(ctx context.Context, request ports.Request) (ports.Message, error) {
	model, err := client.resolveModel(ctx)
	if err != nil {
		return ports.Message{}, err
	}
	payload, err := json.Marshal(newResponsesRequest(model, request))
	if err != nil {
		return ports.Message{}, fmt.Errorf("encode responses request: %w", err)
	}

	return withRetries(ctx, client.backoff, retryableResponse, func() (ports.Message, error) {
		return client.stream(ctx, payload)
	})
}

// Принимает: ctx — контекст.
// Делает: возвращает заданную модель или один раз запрашивает список моделей аккаунта и берёт первую с visibility=list.
// Возвращает: slug модели либо ErrNoModels и ошибки запроса.
func (client *ResponsesClient) resolveModel(ctx context.Context) (string, error) {
	client.modelMutex.Lock()
	defer client.modelMutex.Unlock()
	if client.model != "" {
		return client.model, nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/models", nil)
	if err != nil {
		return "", fmt.Errorf("build models request: %w", err)
	}
	if err := client.authorizer.Authorize(ctx, request); err != nil {
		return "", fmt.Errorf("authorize models request: %w", err)
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("list models: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", readAPIError(response)
	}

	var listing struct {
		Models []struct {
			Slug       string `json:"slug"`
			Visibility string `json:"visibility"`
		} `json:"models"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listing); err != nil {
		return "", fmt.Errorf("decode models: %w", err)
	}
	for _, model := range listing.Models {
		if model.Visibility == "list" && model.Slug != "" {
			client.model = model.Slug
			return client.model, nil
		}
	}
	return "", ErrNoModels
}

// Принимает: ctx — контекст; payload — тело запроса.
// Делает: отправляет запрос и читает поток событий до response.completed.
// Возвращает: собранное сообщение либо *APIError при HTTP-отказе или ошибку потока.
func (client *ResponsesClient) stream(ctx context.Context, payload []byte) (ports.Message, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/responses", bytes.NewReader(payload))
	if err != nil {
		return ports.Message{}, fmt.Errorf("build responses request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	if err := client.authorizer.Authorize(ctx, request); err != nil {
		return ports.Message{}, fmt.Errorf("authorize responses request: %w", err)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return ports.Message{}, fmt.Errorf("call responses API: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return ports.Message{}, readAPIError(response)
	}
	return readEventStream(response.Body)
}

// Принимает: response — ответ с ошибочным статусом.
// Делает: читает короткий фрагмент тела и достаёт error.code, error.message или detail.
// Возвращает: *APIError со статусом и кодом, пригодным для логов (без токенов).
func readAPIError(response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBody))
	var parsed struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Detail string `json:"detail"`
	}
	message := strings.TrimSpace(string(body))
	if json.Unmarshal(body, &parsed) == nil {
		switch {
		case parsed.Error.Code != "":
			message = parsed.Error.Code + ": " + parsed.Error.Message
		case parsed.Error.Message != "":
			message = parsed.Error.Message
		case parsed.Detail != "":
			message = parsed.Detail
		}
	}
	return &APIError{Status: response.StatusCode, Message: message}
}

// Принимает: err — ошибка попытки.
// Делает: решает, стоит ли повторять: временные сбои 5xx и 429, кроме исчерпанного лимита подписки.
// Возвращает: true, если повтор имеет смысл.
func retryableResponse(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if strings.Contains(apiErr.Message, "usage_limit_exceeded") {
		return false
	}
	return retryable(err)
}

// Принимает: body — поток server-sent events.
// Делает: разбирает события: накапливает текст из output_text.delta, вызовы функций из output_item.done и ждёт response.completed.
// Возвращает: сообщение ассистента либо ошибку для response.failed, response.incomplete, error и оборванного потока.
func readEventStream(body io.Reader) (ports.Message, error) {
	reader := bufio.NewReaderSize(body, 64*1024)
	message := ports.Message{Role: ports.RoleAssistant}
	var text strings.Builder
	completed := false

	for {
		line, err := reader.ReadString('\n')
		if len(line) > maxStreamLine {
			return ports.Message{}, errors.New("responses stream line is too long")
		}
		if data, found := strings.CutPrefix(strings.TrimRight(line, "\r\n"), "data:"); found {
			event, parseErr := parseEvent(strings.TrimSpace(data))
			if parseErr != nil {
				return ports.Message{}, parseErr
			}
			switch event.Type {
			case "response.output_text.delta":
				text.WriteString(event.Delta)
			case "response.output_item.done":
				if event.Item.Type == "function_call" {
					message.ToolCalls = append(message.ToolCalls, ports.ToolCall{
						ID: event.Item.CallID, Name: event.Item.Name, Arguments: json.RawMessage(event.Item.Arguments),
					})
				}
			case "response.completed":
				completed = true
			case "response.failed", "error", "response.incomplete":
				return ports.Message{}, event.failure()
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return ports.Message{}, fmt.Errorf("read responses stream: %w", err)
		}
	}
	if !completed {
		return ports.Message{}, errors.New("responses stream ended without response.completed")
	}
	message.Content = text.String()
	return message, nil
}
