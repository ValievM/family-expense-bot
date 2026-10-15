package chatgpt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultAuthorizeURL = "https://auth.openai.com/api/accounts/authorize"
	defaultTokenURL     = "https://auth.openai.com/api/accounts/oauth/token" //nolint:gosec // публичный адрес точки выдачи токенов, не секрет
	maxTokenBody        = 1 << 16
)

// OAuth общается с сервером авторизации OpenAI; адреса можно подменить в тестах.
type OAuth struct {
	AuthorizeURL string
	TokenURL     string
	HTTPClient   *http.Client
	Now          func() time.Time
}

// Принимает: ничего.
// Делает: создаёт клиента с боевыми адресами и таймаутом 30 секунд.
// Возвращает: готовый OAuth.
func NewOAuth() *OAuth {
	return &OAuth{
		AuthorizeURL: defaultAuthorizeURL,
		TokenURL:     defaultTokenURL,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		Now:          time.Now,
	}
}

// TokenError — отказ сервера авторизации с кодом из OAuth.
type TokenError struct {
	Status      int
	Code        string
	Description string
}

// Принимает: ничего.
// Делает: формирует текст ошибки без токенов.
// Возвращает: строку с HTTP-статусом и кодом.
func (err *TokenError) Error() string {
	return fmt.Sprintf("token endpoint returned %d: %s %s", err.Status, err.Code, err.Description)
}

// tokenResponse — ответ точки выдачи токенов.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

// Принимает: ctx — контекст; clientID — выданный идентификатор клиента; code — код из callback; verifier — PKCE-секрет; redirectURI — тот же адрес, что в запросе входа.
// Делает: обменивает код авторизации на токены.
// Возвращает: токены либо *TokenError или ошибку сети.
func (oauth *OAuth) exchangeCode(ctx context.Context, clientID, code, verifier, redirectURI string) (Credentials, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {redirectURI},
		"resource":      {Resource},
	}
	return oauth.requestTokens(ctx, form, Credentials{ClientID: clientID})
}

// Принимает: ctx — контекст; current — сохранённые данные с токеном обновления.
// Делает: меняет токен обновления на новую пару токенов; область (scope) не передаётся, чтобы сохранить выданные права.
// Возвращает: обновлённые данные либо *TokenError или ошибку сети.
func (oauth *OAuth) Refresh(ctx context.Context, current Credentials) (Credentials, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {current.ClientID},
		"refresh_token": {current.RefreshToken},
		"resource":      {Resource},
	}
	return oauth.requestTokens(ctx, form, current)
}

// Принимает: ctx — контекст; form — параметры запроса; base — данные, поля которых сохраняются, если сервер их не вернул.
// Делает: отправляет POST и разбирает токены, подставляя время истечения.
// Возвращает: новые данные либо ошибку; тело ответа в ошибки не попадает.
func (oauth *OAuth) requestTokens(ctx context.Context, form url.Values, base Credentials) (Credentials, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, oauth.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Credentials{}, fmt.Errorf("build token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := oauth.HTTPClient.Do(request)
	if err != nil {
		return Credentials{}, fmt.Errorf("call token endpoint: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxTokenBody))
	if err != nil {
		return Credentials{}, fmt.Errorf("read token response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		var failure struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		_ = json.Unmarshal(body, &failure)
		return Credentials{}, &TokenError{Status: response.StatusCode, Code: failure.Error, Description: failure.ErrorDescription}
	}

	var tokens tokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return Credentials{}, fmt.Errorf("decode token response: %w", err)
	}
	if tokens.AccessToken == "" {
		return Credentials{}, errors.New("token response has no access_token")
	}

	updated := base
	updated.AccessToken = tokens.AccessToken
	if tokens.RefreshToken != "" {
		updated.RefreshToken = tokens.RefreshToken
	}
	if tokens.IDToken != "" {
		updated.IDToken = tokens.IDToken
	}
	if tokens.Scope != "" {
		updated.Scope = tokens.Scope
	}
	updated.ExpiresAt = oauth.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	return updated, nil
}

// Принимает: err — ошибка обновления токена.
// Делает: проверяет коды, при которых токен обновления больше не годится и нужен новый вход.
// Возвращает: true для invalid_grant и родственных кодов.
func isTerminalRefreshError(err error) bool {
	var tokenErr *TokenError
	if !errors.As(err, &tokenErr) {
		return false
	}
	switch tokenErr.Code {
	case "invalid_grant", "invalid_refresh_token", "token_expired", "refresh_token_expired",
		"refresh_token_invalidated", "refresh_token_reused":
		return true
	}
	return false
}
