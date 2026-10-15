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

// Client — минимальный клиент Bot API на стандартной библиотеке.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// ClientOption настраивает Client.
type ClientOption func(*Client)

// Принимает: baseURL — корень API, например адрес тестового сервера.
// Делает: заменяет адрес Bot API.
// Возвращает: ClientOption.
func WithBaseURL(baseURL string) ClientOption {
	return func(client *Client) { client.baseURL = strings.TrimRight(baseURL, "/") }
}

// Принимает: token — токен бота от BotFather; options — необязательные настройки.
// Делает: создаёт клиент без общего таймаута, потому что сроки задаются контекстом каждого запроса.
// Возвращает: готовый Client.
func NewClient(token string, options ...ClientOption) *Client {
	client := &Client{token: token, baseURL: defaultBaseURL, httpClient: &http.Client{}}
	for _, option := range options {
		option(client)
	}
	return client
}

// apiResponse — общий конверт ответа Bot API.
type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	ErrorCode   int             `json:"error_code"`
}

// Принимает: ctx — контекст остановки; offset — ID первого нужного обновления; timeoutSeconds — сколько секунд Telegram держит запрос открытым.
// Делает: запрашивает новые обновления методом getUpdates.
// Возвращает: список обновлений (возможно пустой) либо ошибку без токена.
func (client *Client) GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error) {
	timeout := time.Duration(timeoutSeconds)*time.Second + pollSlack
	var updates []Update
	payload := map[string]any{"offset": offset, "timeout": timeoutSeconds, "allowed_updates": []string{"message"}}
	if err := client.call(ctx, "getUpdates", timeout, payload, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

// Принимает: ctx — контекст запроса; chatID — чат получателя; text — текст сообщения.
// Делает: отправляет обычное текстовое сообщение методом sendMessage.
// Возвращает: nil либо ошибку без токена.
func (client *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	return client.call(ctx, "sendMessage", requestTimeout, map[string]any{"chat_id": chatID, "text": text}, nil)
}

// Принимает: ctx — контекст запроса; chatID — чат получателя.
// Делает: показывает индикатор «печатает…» методом sendChatAction.
// Возвращает: nil либо ошибку без токена.
func (client *Client) SendTyping(ctx context.Context, chatID int64) error {
	return client.call(ctx, "sendChatAction", requestTimeout, map[string]any{"chat_id": chatID, "action": "typing"}, nil)
}

// Принимает: ctx — контекст запроса; method — имя метода Bot API; timeout — срок вызова; payload — тело запроса; result — куда разобрать result или nil.
// Делает: выполняет POST с JSON, проверяет конверт ответа и убирает токен из любых ошибок.
// Возвращает: nil либо ошибку, не содержащую URL с токеном.
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
// Делает: собирает адрес вида {base}/bot{token}/{method}; токен живёт только здесь.
// Возвращает: полный URL.
func (client *Client) endpoint(method string) string {
	return client.baseURL + "/bot" + client.token + "/" + method
}

// Принимает: err — ошибка транспорта.
// Делает: если это *url.Error, оставляет только внутреннюю причину, потому что URL содержит токен бота.
// Возвращает: ошибку, безопасную для логов.
func redact(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
