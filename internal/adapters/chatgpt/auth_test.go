// Package chatgpt_test проверяет вход через ChatGPT на поддельном сервере авторизации.
package chatgpt_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/adapters/chatgpt"
)

// uuidURN — формат идентификатора хоста, который принимает сервер OpenAI.
var uuidURN = regexp.MustCompile(`^urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

var baseTime = time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)

func testOAuth(tokenURL string) *chatgpt.OAuth {
	return &chatgpt.OAuth{
		AuthorizeURL: "https://auth.test/authorize",
		TokenURL:     tokenURL,
		HTTPClient:   http.DefaultClient,
		Now:          func() time.Time { return baseTime },
	}
}

// Проверяет: чтение несуществующего файла даёт ErrNotLoggedIn, запись создаёт файл 0600, данные не теряются.
func TestFileStoreRoundTripAndPermissions(t *testing.T) {
	store := chatgpt.NewFileStore(filepath.Join(t.TempDir(), "nested", "auth.json"))
	if _, err := store.Load(); !errors.Is(err, chatgpt.ErrNotLoggedIn) {
		t.Fatalf("Load() error = %v, want ErrNotLoggedIn", err)
	}

	want := chatgpt.Credentials{ClientID: "oaiapp_1", HostID: "host", AccessToken: "a", RefreshToken: "r", ExpiresAt: baseTime, Scope: chatgpt.PlanScope}
	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load()
	if err != nil || got != want {
		t.Errorf("Load() = %+v, %v; want %+v", got, err, want)
	}
}

// Проверяет: файл токенов недоступен другим пользователям.
func TestFileStoreWritesPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := chatgpt.NewFileStore(path).Save(chatgpt.Credentials{AccessToken: "secret"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}

type memoryStore struct {
	mutex       sync.Mutex
	credentials chatgpt.Credentials
	saves       int
}

func (store *memoryStore) Load() (chatgpt.Credentials, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.credentials, nil
}

func (store *memoryStore) Save(credentials chatgpt.Credentials) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.credentials = credentials
	store.saves++
	return nil
}

// Проверяет: действующий токен выдаётся без обращения к серверу.
func TestTokenSourceReturnsFreshTokenWithoutRefresh(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	store := &memoryStore{credentials: chatgpt.Credentials{AccessToken: "fresh", ExpiresAt: baseTime.Add(time.Hour), Scope: chatgpt.PlanScope}}

	token, err := chatgpt.NewTokenSource(store, testOAuth(server.URL)).Token(context.Background())

	if err != nil || token != "fresh" || calls.Load() != 0 {
		t.Errorf("token = %q, err = %v, refresh calls = %d", token, err, calls.Load())
	}
}

// Проверяет: запрос обновления без scope, сохранение новой пары до выдачи и ровно одно обновление при гонке горутин.
func TestTokenSourceRefreshesOnceAndPersistsRotatedToken(t *testing.T) {
	var calls atomic.Int32
	var form url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = r.ParseForm()
		form = r.PostForm
		_, _ = w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`))
	}))
	defer server.Close()
	store := &memoryStore{credentials: chatgpt.Credentials{
		ClientID: "oaiapp_1", AccessToken: "old", RefreshToken: "old-refresh", ExpiresAt: baseTime.Add(30 * time.Second), Scope: chatgpt.PlanScope,
	}}
	source := chatgpt.NewTokenSource(store, testOAuth(server.URL))

	var wait sync.WaitGroup
	for range 10 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if token, err := source.Token(context.Background()); err != nil || token != "new-access" {
				t.Errorf("Token() = %q, %v", token, err)
			}
		}()
	}
	wait.Wait()

	if calls.Load() != 1 {
		t.Errorf("refresh calls = %d, want exactly 1", calls.Load())
	}
	if form.Get("grant_type") != "refresh_token" || form.Get("client_id") != "oaiapp_1" || form.Get("refresh_token") != "old-refresh" ||
		form.Get("resource") != chatgpt.Resource || form.Has("scope") {
		t.Errorf("refresh form = %v", form)
	}
	saved, _ := store.Load()
	if saved.RefreshToken != "new-refresh" || saved.AccessToken != "new-access" || !saved.ExpiresAt.Equal(baseTime.Add(time.Hour)) {
		t.Errorf("saved = %+v, want the rotated pair persisted", saved)
	}
}

// Проверяет: отозванный токен обновления даёт ErrReauthRequired, временный сбой сервера нет.
func TestTokenSourceReportsReauthOnlyForTerminalErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantReauth bool
	}{
		{"revoked", http.StatusBadRequest, `{"error":"invalid_grant"}`, true},
		{"reused", http.StatusBadRequest, `{"error":"refresh_token_reused"}`, true},
		{"server down", http.StatusServiceUnavailable, `{"error":"temporarily_unavailable"}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			store := &memoryStore{credentials: chatgpt.Credentials{RefreshToken: "r", ExpiresAt: baseTime, Scope: chatgpt.PlanScope}}

			_, err := chatgpt.NewTokenSource(store, testOAuth(server.URL)).Token(context.Background())

			if got := errors.Is(err, chatgpt.ErrReauthRequired); got != tt.wantReauth || err == nil {
				t.Errorf("error = %v, ErrReauthRequired = %v, want %v", err, got, tt.wantReauth)
			}
			if saved, _ := store.Load(); store.saves != 0 || saved.RefreshToken != "r" {
				t.Errorf("credentials changed after a failed refresh: %+v", saved)
			}
		})
	}
}

// Проверяет: вход без права расходовать подписку отклоняется.
func TestTokenSourceRequiresPlanScope(t *testing.T) {
	store := &memoryStore{credentials: chatgpt.Credentials{AccessToken: "a", ExpiresAt: baseTime.Add(time.Hour), Scope: "openid"}}

	_, err := chatgpt.NewTokenSource(store, testOAuth("http://unused")).Token(context.Background())

	if err == nil || !strings.Contains(err.Error(), "plan-usage") {
		t.Errorf("error = %v, want a plan scope error", err)
	}
}

type fakeBrowser struct {
	t       *testing.T
	mutate  func(query url.Values) url.Values
	authURL *url.URL
}

func (browser *fakeBrowser) open(authURL string) error {
	parsed, err := url.Parse(authURL)
	if err != nil {
		return err
	}
	browser.authURL = parsed
	query := parsed.Query()
	callback := url.Values{"code": {"auth-code"}, "state": {query.Get("state")}, "client_id": {"oaiapp_issued"}}
	if browser.mutate != nil {
		callback = browser.mutate(callback)
	}
	go func() {
		response, err := http.Get(query.Get("redirect_uri") + "?" + callback.Encode())
		if err == nil {
			_ = response.Body.Close()
		}
	}()
	return nil
}

// Проверяет: полный вход: параметры PKCE и ресурса, регистрация с именем, обмен кода с верным verifier, сохранение client_id.
func TestLoginCompletesPKCEFlow(t *testing.T) {
	var exchange url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		exchange = r.PostForm
		_, _ = w.Write([]byte(`{"access_token":"A","refresh_token":"R","expires_in":3600,"scope":"openid chatgpt.tokens.use.direct"}`))
	}))
	defer server.Close()
	browser := &fakeBrowser{t: t}

	credentials, err := chatgpt.Login(context.Background(), testOAuth(server.URL), chatgpt.LoginOptions{
		AgentName: "FamilyExpenseBot", OpenBrowser: browser.open, Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	query := browser.authURL.Query()
	if query.Get("client_id") != chatgpt.DynamicClientID || query.Get("agent_name_hint") != "FamilyExpenseBot" ||
		query.Get("code_challenge_method") != "S256" || query.Get("resource") != chatgpt.Resource ||
		query.Get("response_type") != "code" || !strings.Contains(query.Get("scope"), chatgpt.PlanScope) ||
		!strings.HasPrefix(query.Get("redirect_uri"), "http://127.0.0.1:") || query.Get("ext_agent_host_id") == "" ||
		!uuidURN.MatchString(query.Get("ext_agent_host_id")) || query.Get("nonce") == "" {
		t.Errorf("authorization query = %v", query)
	}
	sum := sha256.Sum256([]byte(exchange.Get("code_verifier")))
	if base64.RawURLEncoding.EncodeToString(sum[:]) != query.Get("code_challenge") {
		t.Error("code_verifier does not match the code_challenge")
	}
	if exchange.Get("grant_type") != "authorization_code" || exchange.Get("code") != "auth-code" ||
		exchange.Get("client_id") != "oaiapp_issued" || exchange.Get("redirect_uri") != query.Get("redirect_uri") {
		t.Errorf("exchange form = %v", exchange)
	}
	if credentials.ClientID != "oaiapp_issued" || credentials.AccessToken != "A" || credentials.HostID != query.Get("ext_agent_host_id") || !credentials.HasPlanScope() {
		t.Errorf("credentials = %+v", credentials)
	}
}

// Проверяет: повторный вход использует сохранённые client_id и host id и не шлёт имя приложения.
func TestLoginReusesIssuedClientOnReauthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"A","refresh_token":"R","expires_in":60,"scope":"chatgpt.tokens.use.direct"}`))
	}))
	defer server.Close()
	browser := &fakeBrowser{t: t, mutate: func(values url.Values) url.Values { values.Del("client_id"); return values }}
	existing := chatgpt.Credentials{ClientID: "oaiapp_old", HostID: "host-1"}

	credentials, err := chatgpt.Login(context.Background(), testOAuth(server.URL), chatgpt.LoginOptions{
		Existing: &existing, AgentName: "ignored", OpenBrowser: browser.open, Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	query := browser.authURL.Query()
	if query.Get("client_id") != "oaiapp_old" || query.Get("ext_agent_host_id") != "host-1" || query.Has("agent_name_hint") {
		t.Errorf("authorization query = %v", query)
	}
	if credentials.ClientID != "oaiapp_old" {
		t.Errorf("ClientID = %q, want the existing one", credentials.ClientID)
	}
}

// Проверяет: чужой state, отказ пользователя и отсутствие права на подписку не дают токенов; обмен кода только в последнем случае.
func TestLoginRejectsUnsafeOutcomes(t *testing.T) {
	tests := []struct {
		name         string
		mutate       func(url.Values) url.Values
		tokenScope   string
		wantContains string
		wantExchange bool
	}{
		{"state mismatch", func(v url.Values) url.Values { v.Set("state", "forged"); return v }, "chatgpt.tokens.use.direct", "state", false},
		{"user denied", func(v url.Values) url.Values {
			return url.Values{"error": {"access_denied"}, "state": {v.Get("state")}}
		}, "chatgpt.tokens.use.direct", "refused", false},
		{"plan scope not granted", nil, "openid", "plan usage", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exchanged atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				exchanged.Store(true)
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "A", "refresh_token": "R", "expires_in": 60, "scope": tt.tokenScope})
			}))
			defer server.Close()
			browser := &fakeBrowser{t: t, mutate: tt.mutate}

			_, err := chatgpt.Login(context.Background(), testOAuth(server.URL), chatgpt.LoginOptions{OpenBrowser: browser.open, Timeout: 5 * time.Second})

			if err == nil || !strings.Contains(err.Error(), tt.wantContains) {
				t.Errorf("error = %v, want it to mention %q", err, tt.wantContains)
			}
			if exchanged.Load() != tt.wantExchange {
				t.Errorf("code exchanged = %v, want %v", exchanged.Load(), tt.wantExchange)
			}
		})
	}
}

// Проверяет: если пользователь не вернулся, вход завершается ошибкой по таймауту, а не зависает.
func TestLoginTimesOutWhenUserNeverReturns(t *testing.T) {
	_, err := chatgpt.Login(context.Background(), testOAuth("http://unused"), chatgpt.LoginOptions{
		OpenBrowser: func(string) error { return nil }, Timeout: 50 * time.Millisecond,
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want a deadline error", err)
	}
}
