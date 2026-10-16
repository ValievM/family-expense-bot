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
	"github.com/ValievM/family-expense-bot/internal/ports"
)

func writeEvents(w http.ResponseWriter, events ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, event := range events {
		_, _ = w.Write([]byte("event: x\ndata: " + event + "\n\n"))
	}
}

func newResponsesTestClient(url, model string) *llm.ResponsesClient {
	return llm.NewResponsesClient(url+"/v1/", model, llm.NewAPIKeyAuthorizer("token"), llm.WithBackoff(noBackoff))
}

// Проверяет: store=false, stream=true, инструменты; разговор переведён в message, function_call и function_call_output.
func TestResponsesRequestShape(t *testing.T) {
	var body map[string]any
	var auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&body)
		writeEvents(w, `{"type":"response.output_text.delta","delta":"ok"}`, `{"type":"response.completed"}`)
	}))
	defer server.Close()

	_, err := newResponsesTestClient(server.URL, "slug-1").Complete(context.Background(), ports.Request{
		Messages: []ports.Message{
			{Role: ports.RoleSystem, Content: "rules"},
			{Role: ports.RoleUser, Content: "кофе 350"},
			{Role: ports.RoleAssistant, ToolCalls: []ports.ToolCall{{ID: "c1", Name: "add_expense", Arguments: json.RawMessage(`{"amount":350}`)}}},
			{Role: ports.RoleTool, ToolCallID: "c1", Content: `{"id":1}`},
		},
		Tools: []ports.ToolSpec{{Name: "add_expense", Description: "adds", Parameters: json.RawMessage(`{"type":"object"}`)}},
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

// Проверяет: разбор потока: текст из дельт и вызов функции из output_item.done.
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

	reply, err := newResponsesTestClient(server.URL, "m").Complete(context.Background(), ports.Request{})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if reply.Content != "Записал" || len(reply.ToolCalls) != 1 || reply.ToolCalls[0].ID != "call_9" ||
		reply.ToolCalls[0].Name != "list_expenses" || string(reply.ToolCalls[0].Arguments) != `{"category":"food"}` {
		t.Errorf("reply = %+v", reply)
	}
}

// Проверяет: без заданной модели берётся первая с visibility=list, выбор запоминается.
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
		if _, err := client.Complete(context.Background(), ports.Request{}); err != nil {
			t.Fatalf("Complete() error = %v", err)
		}
	}

	if used.Load() != "gpt-listed" || modelCalls.Load() != 1 {
		t.Errorf("model = %v, model list calls = %d, want gpt-listed picked once", used.Load(), modelCalls.Load())
	}
}

// Проверяет: сбой потока с кодом OpenAI и оборванный поток сообщаются без повторов.
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

			_, err := newResponsesTestClient(server.URL, "m").Complete(context.Background(), ports.Request{})

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", err, tt.want)
			}
			if calls.Load() != 1 {
				t.Errorf("calls = %d, want 1: stream failures must not be retried", calls.Load())
			}
		})
	}
}

// Проверяет: 503 повторяется, исчерпанный лимит подписки (429 с кодом) и 401 нет.
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

			_, err := newResponsesTestClient(server.URL, "m").Complete(context.Background(), ports.Request{})

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
