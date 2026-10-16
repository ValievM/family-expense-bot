package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.telegram.org"
	// requestTimeout ограничивает короткие вызовы; long polling получает свой срок поверх интервала ожидания.
	requestTimeout = 15 * time.Second
	pollSlack      = 10 * time.Second
)

type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

type ClientOption func(*Client)

func WithBaseURL(baseURL string) ClientOption {
	return func(client *Client) { client.baseURL = strings.TrimRight(baseURL, "/") }
}

// Принимает: token — токен бота; options — настройки.
// Делает: создаёт клиент без общего таймаута: сроки задаются контекстом каждого запроса.
// Возвращает: *Client.
func NewClient(token string, options ...ClientOption) *Client {
	client := &Client{token: token, baseURL: defaultBaseURL, httpClient: &http.Client{}}
	for _, option := range options {
		option(client)
	}
	return client
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	ErrorCode   int             `json:"error_code"`
}

// Принимает: ctx; offset — ID первого нужного обновления; timeoutSeconds — сколько Telegram держит запрос.
// Делает: запрашивает новые обновления методом getUpdates.
// Возвращает: обновления (возможно пустые) или ошибку без токена.
func (client *Client) GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error) {
	timeout := time.Duration(timeoutSeconds)*time.Second + pollSlack
	var updates []Update
	payload := map[string]any{"offset": offset, "timeout": timeoutSeconds, "allowed_updates": []string{"message"}}
	if err := client.call(ctx, "getUpdates", timeout, payload, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

func (client *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	return client.call(ctx, "sendMessage", requestTimeout, map[string]any{"chat_id": chatID, "text": text}, nil)
}

func (client *Client) SendHTML(ctx context.Context, chatID int64, html string) error {
	return client.call(ctx, "sendMessage", requestTimeout, map[string]any{
		"chat_id": chatID, "text": html, "parse_mode": "HTML", "link_preview_options": map[string]any{"is_disabled": true},
	}, nil)
}

// Принимает: ctx; profile — описание и меню команд.
// Делает: обновляет карточку бота: setMyDescription, setMyShortDescription и setMyCommands.
// Возвращает: nil либо первую ошибку Bot API.
func (client *Client) SetProfile(ctx context.Context, profile Profile) error {
	commands := make([]map[string]string, 0, len(profile.Commands))
	for _, command := range profile.Commands {
		commands = append(commands, map[string]string{"command": command.Name, "description": command.Description})
	}
	steps := []struct {
		method  string
		payload map[string]any
	}{
		{"setMyDescription", map[string]any{"description": profile.Description}},
		{"setMyShortDescription", map[string]any{"short_description": profile.ShortDescription}},
		{"setMyCommands", map[string]any{"commands": commands}},
	}
	for _, step := range steps {
		if err := client.call(ctx, step.method, requestTimeout, step.payload, nil); err != nil {
			return err
		}
	}
	return nil
}

func (client *Client) SendTyping(ctx context.Context, chatID int64) error {
	return client.call(ctx, "sendChatAction", requestTimeout, map[string]any{"chat_id": chatID, "action": "typing"}, nil)
}

// Принимает: ctx; method — метод Bot API; timeout — срок вызова; payload — тело; result — куда разобрать result или nil.
// Делает: POST с JSON, проверка конверта ответа, удаление токена из любых ошибок.
// Возвращает: nil или ошибку без URL с токеном.
func (client *Client) call(ctx context.Context, method string, timeout time.Duration, payload any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode telegram %s request: %w", method, err)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint(method), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build telegram %s request: %w", method, redact(err))
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call telegram %s: %w", method, redact(err))
	}
	defer func() { _ = response.Body.Close() }()

	var envelope apiResponse
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode telegram %s response (HTTP %d): %w", method, response.StatusCode, err)
	}
	if !envelope.OK {
		return fmt.Errorf("telegram %s failed: %d %s", method, envelope.ErrorCode, envelope.Description)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("decode telegram %s result: %w", method, err)
	}
	return nil
}

// Принимает: method — имя метода Bot API.
// Делает: собирает {base}/bot{token}/{method}; токен живёт только здесь.
// Возвращает: полный URL.
func (client *Client) endpoint(method string) string {
	return client.baseURL + "/bot" + client.token + "/" + method
}

// Принимает: err — ошибка транспорта.
// Делает: для *url.Error оставляет только внутреннюю причину, потому что URL содержит токен.
// Возвращает: ошибку, безопасную для логов.
func redact(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
