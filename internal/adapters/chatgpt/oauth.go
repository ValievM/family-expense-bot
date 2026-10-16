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

// OAuth — клиент сервера авторизации OpenAI; адреса подменяются в тестах.
type OAuth struct {
	AuthorizeURL string
	TokenURL     string
	HTTPClient   *http.Client
	Now          func() time.Time
}

func NewOAuth() *OAuth {
	return &OAuth{
		AuthorizeURL: defaultAuthorizeURL,
		TokenURL:     defaultTokenURL,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		Now:          time.Now,
	}
}

type TokenError struct {
	Status      int
	Code        string
	Description string
}

func (err *TokenError) Error() string {
	return fmt.Sprintf("token endpoint returned %d: %s %s", err.Status, err.Code, err.Description)
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

// Принимает: ctx; clientID; code — код из callback; verifier — PKCE-секрет; redirectURI — как в запросе входа.
// Делает: обменивает код авторизации на токены.
// Возвращает: токены, *TokenError или ошибку сети.
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

// Принимает: ctx; current — сохранённые данные с токеном обновления.
// Делает: меняет токен обновления на новую пару; scope не передаётся, чтобы сохранились выданные права.
// Возвращает: обновлённые данные, *TokenError или ошибку сети.
func (oauth *OAuth) Refresh(ctx context.Context, current Credentials) (Credentials, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {current.ClientID},
		"refresh_token": {current.RefreshToken},
		"resource":      {Resource},
	}
	return oauth.requestTokens(ctx, form, current)
}

// Принимает: ctx; form — параметры запроса; base — данные, поля которых остаются, если сервер их не вернул.
// Делает: отправляет POST и разбирает токены, считая время истечения.
// Возвращает: новые данные или ошибку; тело ответа в ошибку не попадает.
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
	if tokens.Scope != "" {
		updated.Scope = tokens.Scope
	}
	updated.ExpiresAt = oauth.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	return updated, nil
}

// Принимает: err — ошибка обновления токена.
// Делает: проверяет, что токен обновления больше не годится и нужен новый вход.
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
