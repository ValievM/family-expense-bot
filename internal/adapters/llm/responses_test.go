package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ValievM/family-expense-bot/internal/adapters/llm"
	"github.com/ValievM/family-expense-bot/internal/service/agent"
)

// Принимает: w — писатель ответа; events — строки событий (JSON без префикса data).
// Делает: отправляет их как поток server-sent events.
// Возвращает: ничего.
func writeEvents(w http.ResponseWriter, events ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, event := range events {
		_, _ = w.Write([]byte("event: x\ndata: " + event + "\n\n"))
	}
}

// Принимает: url — адрес тестового сервера; model — модель или пустая строка.
// Делает: создаёт клиента Responses API без пауз между повторами.
// Возвращает: *llm.ResponsesClient.
func newResponsesTestClient(url, model string) *llm.ResponsesClient {
	return llm.NewResponsesClient(url+"/v1/", model, llm.NewAPIKeyAuthorizer("token"), llm.WithBackoff(noBackoff))
}

// Принимает: t — тестовый контекст.
// Делает: проверяет запрос: store=false, stream=true, инструменты, а разговор переведён в сообщения, function_call и function_call_output.
// Возвращает: ничего.
func TestResponsesRequestShape(t *testing.T) {
	var body map[string]any
	var auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&body)
		writeEvents(w, `{"type":"response.output_text.delta","delta":"ok"}`, `{"type":"response.completed"}`)
	}))
	defer server.Close()

	_, err := newResponsesTestClient(server.URL, "slug-1").Complete(context.Background(), agent.Request{
		Messages: []agent.Message{
			{Role: agent.RoleSystem, Content: "rules"},
			{Role: agent.RoleUser, Content: "кофе 350"},
			{Role: agent.RoleAssistant, ToolCalls: []agent.ToolCall{{ID: "c1", Name: "add_expense", Arguments: json.RawMessage(`{"amount":350}`)}}},
			{Role: agent.RoleTool, ToolCallID: "c1", Content: `{"id":1}`},
		},
		Tools: []agent.ToolSpec{{Name: "add_expense", Description: "adds", Parameters: json.RawMessage(`{"type":"object"}`)}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if auth != "Bearer token" || body["model"] != "slug-1" || body["store"] != false || body["stream"] != true {
		t.Errorf("auth = %q, body = %v", auth, body)
	}
	input := body["input"].([]any)
	if len(input) != 4 {
		t.Fatalf("input items = %d, want 4: %v", len(input), input)
	}
	call := input[2].(map[string]any)
	output := input[3].(map[string]any)
	if call["type"] != "function_call" || call["call_id"] != "c1" || call["arguments"] != `{"amount":350}` {
		t.Errorf("function call item = %v", call)
	}
	if output["type"] != "function_call_output" || output["call_id"] != "c1" || output["output"] != `{"id":1}` {
		t.Errorf("function output item = %v", output)
	}
	tool := body["tools"].([]any)[0].(map[string]any)
	if tool["type"] != "function" || tool["name"] != "add_expense" {
		t.Errorf("tool = %v", tool)
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет разбор потока: текст из дельт и вызов функции из output_item.done.
// Возвращает: ничего.
func TestResponsesParsesTextAndToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeEvents(w,
			`{"type":"response.output_text.delta","delta":"Запи"}`,
			`{"type":"response.output_text.delta","delta":"сал"}`,
			`{"type":"response.output_item.done","item":{"type":"function_call","call_id":"call_9","name":"list_expenses","arguments":"{\"category\":\"food\"}"}}`,
			`{"type":"response.output_item.done","item":{"type":"message"}}`,
			`{"type":"response.completed"}`,
		)
	}))
	defer server.Close()

	reply, err := newResponsesTestClient(server.URL, "m").Complete(context.Background(), agent.Request{})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if reply.Content != "Записал" || len(reply.ToolCalls) != 1 || reply.ToolCalls[0].ID != "call_9" ||
		reply.ToolCalls[0].Name != "list_expenses" || string(reply.ToolCalls[0].Arguments) != `{"category":"food"}` {
		t.Errorf("reply = %+v", reply)
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет, что без заданной модели берётся первая с visibility=list и выбор запоминается.
// Возвращает: ничего.
func TestResponsesPicksFirstListedModelOnce(t *testing.T) {
	var modelCalls atomic.Int32
	var used atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			modelCalls.Add(1)
			_, _ = w.Write([]byte(`{"models":[{"slug":"hidden-1","visibility":"hide"},{"slug":"gpt-listed","visibility":"list"},{"slug":"gpt-second","visibility":"list"}]}`))
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		used.Store(body["model"])
		writeEvents(w, `{"type":"response.completed"}`)
	}))
	defer server.Close()
	client := newResponsesTestClient(server.URL, "")

	for range 2 {
		if _, err := client.Complete(context.Background(), agent.Request{}); err != nil {
			t.Fatalf("Complete() error = %v", err)
		}
	}

	if used.Load() != "gpt-listed" || modelCalls.Load() != 1 {
		t.Errorf("model = %v, model list calls = %d, want gpt-listed picked once", used.Load(), modelCalls.Load())
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет ошибки потока: сбой с кодом OpenAI, оборванный поток и отсутствие повторов для таких ошибок.
// Возвращает: ничего.
func TestResponsesStreamFailuresAreReportedWithoutRetry(t *testing.T) {
	tests := []struct {
		name   string
		events []string
		want   string
	}{
		{"failed with code", []string{`{"type":"response.failed","response":{"error":{"code":"subscription_sharing_unsupported_capability","message":"tools"}}}`}, "unsupported_capability"},
		{"stream error", []string{`{"type":"error","code":"boom","message":"bad"}`}, "boom"},
		{"incomplete", []string{`{"type":"response.incomplete","response":{"incomplete_details":{"reason":"max_output_tokens"}}}`}, "max_output_tokens"},
		{"cut off", []string{`{"type":"response.output_text.delta","delta":"x"}`}, "without response.completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				writeEvents(w, tt.events...)
			}))
			defer server.Close()

			_, err := newResponsesTestClient(server.URL, "m").Complete(context.Background(), agent.Request{})

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", err, tt.want)
			}
			if calls.Load() != 1 {
				t.Errorf("calls = %d, want 1: stream failures must not be retried", calls.Load())
			}
		})
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет HTTP-ошибки: временный 503 повторяется, исчерпанный лимит подписки (429 с кодом) нет, 401 не повторяется.
// Возвращает: ничего.
func TestResponsesRetryPolicy(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantCalls int32
	}{
		{"unavailable is retried", 503, `{"error":{"code":"subscription_sharing_usage_unavailable","message":"later"}}`, 3},
		{"usage limit is not retried", 429, `{"error":{"code":"subscription_sharing_usage_limit_exceeded","message":"limit"}}`, 1},
		{"unauthorized is not retried", 401, `{"detail":"bad token"}`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			_, err := newResponsesTestClient(server.URL, "m").Complete(context.Background(), agent.Request{})

			var apiErr *llm.APIError
			if !errors.As(err, &apiErr) || apiErr.Status != tt.status {
				t.Fatalf("error = %v, want APIError %d", err, tt.status)
			}
			if calls.Load() != tt.wantCalls {
				t.Errorf("calls = %d, want %d", calls.Load(), tt.wantCalls)
			}
		})
	}
}
