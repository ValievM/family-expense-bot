// Package llm_test — тесты LLM-клиента через httptest-сервер.
package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/adapters/llm"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

const (
	testAPIPrefix = "/v1"
	testModel     = "test-model"
	testAPIKey    = "secret"
)

func noBackoff(int) time.Duration { return 0 }

func newTestClient(url string) *llm.Client {
	return llm.NewClient(url+testAPIPrefix+"/", testModel, llm.NewAPIKeyAuthorizer(testAPIKey), llm.WithBackoff(noBackoff))
}

// Проверяет: Complete шлёт модель, bearer-ключ и инструменты с tool_choice=auto и разбирает tool_calls ответа.
func TestCompleteSendsToolsAndParsesToolCalls(t *testing.T) {
	var got map[string]any
	var auth, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"add_expense","arguments":"{\"amount\":35000}"}}]}}]}`))
	}))
	defer server.Close()

	reply, err := newTestClient(server.URL).Complete(context.Background(), ports.Request{
		Messages: []ports.Message{{Role: ports.RoleSystem, Content: "rules"}, {Role: ports.RoleUser, Content: "кофе 350"}},
		Tools:    []ports.ToolSpec{{Name: "add_expense", Description: "adds", Parameters: json.RawMessage(`{"type":"object"}`)}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if path != testAPIPrefix+"/chat/completions" || auth != "Bearer "+testAPIKey {
		t.Errorf("path = %q, auth = %q", path, auth)
	}
	if got["model"] != testModel || got["tool_choice"] != "auto" {
		t.Errorf("request = %v", got)
	}
	tools := got["tools"].([]any)
	if function := tools[0].(map[string]any)["function"].(map[string]any); function["name"] != "add_expense" {
		t.Errorf("tool = %v", tools[0])
	}
	if len(reply.ToolCalls) != 1 || reply.ToolCalls[0].ID != "call_1" || string(reply.ToolCalls[0].Arguments) != `{"amount":35000}` {
		t.Errorf("reply = %+v", reply)
	}
}

// Проверяет: ассистентское сообщение с tool_calls уходит с null content, ответ инструмента несёт tool_call_id.
func TestCompleteEncodesToolResultAndAssistantToolCalls(t *testing.T) {
	var got struct {
		Messages []map[string]any `json:"messages"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Готово"}}]}`))
	}))
	defer server.Close()

	reply, err := newTestClient(server.URL).Complete(context.Background(), ports.Request{Messages: []ports.Message{
		{Role: ports.RoleAssistant, ToolCalls: []ports.ToolCall{{ID: "c1", Name: "list_expenses", Arguments: json.RawMessage(`{}`)}}},
		{Role: ports.RoleTool, ToolCallID: "c1", Content: `[]`},
	}})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if reply.Content != "Готово" || len(reply.ToolCalls) != 0 {
		t.Errorf("reply = %+v", reply)
	}
	if got.Messages[0]["content"] != nil || got.Messages[0]["tool_calls"] == nil {
		t.Errorf("assistant message = %v, want null content with tool_calls", got.Messages[0])
	}
	if got.Messages[1]["tool_call_id"] != "c1" || got.Messages[1]["role"] != "tool" {
		t.Errorf("tool message = %v", got.Messages[1])
	}
}

// Проверяет: 429 повторяется, после третьей попытки возвращается успех.
func TestCompleteRetriesTransientFailures(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			http.Error(w, "busy", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	reply, err := newTestClient(server.URL).Complete(context.Background(), ports.Request{})
	if err != nil || reply.Content != "ok" || calls.Load() != 3 {
		t.Errorf("reply = %+v, err = %v, calls = %d, want ok after 3 calls", reply, err, calls.Load())
	}
}

// Проверяет: 401 не повторяется и возвращается как *llm.APIError.
func TestCompleteDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := newTestClient(server.URL).Complete(context.Background(), ports.Request{})

	var apiErr *llm.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized || calls.Load() != 1 {
		t.Errorf("err = %v, calls = %d, want one 401 APIError", err, calls.Load())
	}
}

// Проверяет: пустой список choices даёт llm.ErrEmptyResponse.
func TestCompleteReportsEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	_, err := newTestClient(server.URL).Complete(context.Background(), ports.Request{})

	if !errors.Is(err, llm.ErrEmptyResponse) {
		t.Errorf("err = %v, want ErrEmptyResponse", err)
	}
}

// Проверяет: отмена контекста во время паузы прерывает Complete с context.Canceled.
func TestCompleteStopsRetryingWhenContextIsCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	client := llm.NewClient(server.URL, "m", llm.NewAPIKeyAuthorizer(""), llm.WithBackoff(func(int) time.Duration {
		cancel()
		return time.Hour
	}))

	_, err := client.Complete(ctx, ports.Request{})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
