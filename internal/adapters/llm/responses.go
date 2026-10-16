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

// ResponsesClient — потоковый клиент Responses API; так работает оплата подпиской ChatGPT.
type ResponsesClient struct {
	baseURL    string
	authorizer Authorizer
	httpClient *http.Client
	backoff    func(attempt int) time.Duration

	modelMutex sync.Mutex
	model      string
}

// Принимает: корень API, slug модели (пусто — первая доступная аккаунту), авторизатор и опции.
// Делает: создаёт клиента с теми же настройками, что и у Client.
// Возвращает: *ResponsesClient.
func NewResponsesClient(baseURL, model string, authorizer Authorizer, options ...Option) *ResponsesClient {
	base := NewClient(baseURL, model, authorizer, options...)
	return &ResponsesClient{
		baseURL: base.baseURL, model: model, authorizer: authorizer,
		httpClient: base.httpClient, backoff: base.backoff,
	}
}

// Принимает: ctx — контекст; request — разговор и инструменты.
// Делает: выбирает модель, отправляет запрос потоком и собирает ответ; временные сбои повторяет.
// Возвращает: сообщение ассистента (текст и/или вызовы инструментов) или ошибку.
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
// Делает: берёт заданную модель или один раз запрашивает список аккаунта и берёт первую с visibility=list.
// Возвращает: slug модели, ErrNoModels или ошибку запроса.
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
// Возвращает: собранное сообщение, *APIError при HTTP-отказе или ошибку потока.
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
// Возвращает: *APIError со статусом и кодом, безопасным для логов (без токенов).
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
// Делает: разрешает повтор для 5xx и 429, кроме исчерпанного лимита подписки.
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
// Делает: собирает ответ ассистента из событий: текст из дельт, вызовы функций, признак завершения.
// Возвращает: сообщение либо ошибку сбоя, неполного ответа, обрыва потока или разбора.
func readEventStream(body io.Reader) (ports.Message, error) {
	reply := replyBuilder{message: ports.Message{Role: ports.RoleAssistant}}
	if err := eachEvent(body, reply.apply); err != nil {
		return ports.Message{}, err
	}
	return reply.finish()
}

type replyBuilder struct {
	message   ports.Message
	text      strings.Builder
	completed bool
}

// Принимает: event — очередное событие потока.
// Делает: накапливает текст и вызовы функций, запоминает завершение.
// Возвращает: ошибку для response.failed, error и response.incomplete, иначе nil.
func (reply *replyBuilder) apply(event streamEvent) error {
	switch event.Type {
	case "response.output_text.delta":
		reply.text.WriteString(event.Delta)
	case "response.output_item.done":
		if event.Item.Type == "function_call" {
			reply.message.ToolCalls = append(reply.message.ToolCalls, ports.ToolCall{
				ID: event.Item.CallID, Name: event.Item.Name, Arguments: json.RawMessage(event.Item.Arguments),
			})
		}
	case "response.completed":
		reply.completed = true
	case "response.failed", "error", "response.incomplete":
		return event.failure()
	}
	return nil
}

func (reply *replyBuilder) finish() (ports.Message, error) {
	if !reply.completed {
		return ports.Message{}, errors.New("responses stream ended without response.completed")
	}
	reply.message.Content = reply.text.String()
	return reply.message, nil
}

// Принимает: body — поток events; handle — обработчик события.
// Делает: читает строки потока, разбирает строки data: и передаёт события обработчику до конца потока или первой ошибки.
// Возвращает: nil на конце потока либо ошибку чтения, разбора или обработчика.
func eachEvent(body io.Reader, handle func(streamEvent) error) error {
	reader := bufio.NewReaderSize(body, 64*1024)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > maxStreamLine {
			return errors.New("responses stream line is too long")
		}
		if data, found := strings.CutPrefix(strings.TrimRight(line, "\r\n"), "data:"); found {
			event, parseErr := parseEvent(strings.TrimSpace(data))
			if parseErr != nil {
				return parseErr
			}
			if handleErr := handle(event); handleErr != nil {
				return handleErr
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read responses stream: %w", err)
		}
	}
}

var _ ports.LLM = (*ResponsesClient)(nil)
