package telegram_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ValievM/family-expense-bot/internal/adapters/telegram"
)

// Проверяет: адрес вызова с токеном, тело запроса и разбор списка обновлений.
func TestGetUpdatesCallsBotAPI(t *testing.T) {
	var path string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":7,"message":{"message_id":1,"from":{"id":42},"chat":{"id":42,"type":"private"},"text":"кофе 350"}}]}`))
	}))
	defer server.Close()

	updates, err := telegram.NewClient("TOKEN", telegram.WithBaseURL(server.URL)).GetUpdates(context.Background(), 5, 30)
	if err != nil {
		t.Fatalf("GetUpdates() error = %v", err)
	}

	if path != "/botTOKEN/getUpdates" || body["offset"].(float64) != 5 || body["timeout"].(float64) != 30 {
		t.Errorf("path = %q, body = %v", path, body)
	}
	if len(updates) != 1 || updates[0].UpdateID != 7 || updates[0].Message.Text != "кофе 350" || updates[0].Message.From.ID != 42 {
		t.Errorf("updates = %+v", updates)
	}
}

// Проверяет: отказ Bot API превращается в ошибку с кодом и описанием.
func TestSendMessageReportsAPIFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`))
	}))
	defer server.Close()

	err := telegram.NewClient("TOKEN", telegram.WithBaseURL(server.URL)).SendMessage(context.Background(), 42, "привет")

	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error = %v, want code and description", err)
	}
}

// Проверяет: при сетевой ошибке токен бота не попадает в текст ошибки.
func TestTransportErrorsNeverLeakTheToken(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()

	err := telegram.NewClient("SECRET-TOKEN-123", telegram.WithBaseURL(url)).SendTyping(context.Background(), 42)

	if err == nil {
		t.Fatal("SendTyping() error = nil, want a connection error")
	}
	if strings.Contains(err.Error(), "SECRET-TOKEN-123") {
		t.Errorf("error leaks the bot token: %v", err)
	}
}
