package chatgpt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	callbackPath = "/auth/callback"
	loginScope   = "openid profile email offline_access resource.invoke " + PlanScope
	loginTimeout = 5 * time.Minute
)

type LoginOptions struct {
	// Existing — прежние данные: из них берутся client_id и host id для повторного входа.
	Existing  *Credentials
	AgentName string
	// Port — порт локального callback-сервера на 127.0.0.1; 0 выбирает свободный.
	Port        int
	OpenBrowser func(authURL string) error
	// Timeout — ожидание завершения входа; ноль означает loginTimeout.
	Timeout time.Duration
}

type callbackResult struct {
	code, state, clientID, errorCode string
}

// Принимает: ctx; oauth — клиент сервера авторизации; options — настройки входа.
// Делает: поднимает callback-сервер, открывает браузер, проверяет state, меняет код по PKCE, требует право на подписку.
// Возвращает: новые данные или ошибку отказа, таймаута либо обмена кода.
func Login(ctx context.Context, oauth *OAuth, options LoginOptions) (Credentials, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(options.Port)))
	if err != nil {
		return Credentials{}, fmt.Errorf("listen for the sign-in callback: %w", err)
	}
	defer func() { _ = listener.Close() }()
	redirectURI := fmt.Sprintf("http://%s%s", listener.Addr().String(), callbackPath)

	verifier, err := randomString(48)
	if err != nil {
		return Credentials{}, err
	}
	state, err := randomString(24)
	if err != nil {
		return Credentials{}, err
	}
	nonce, err := randomString(24)
	if err != nil {
		return Credentials{}, err
	}
	hostID := ""
	clientID, agentName := DynamicClientID, options.AgentName
	if options.Existing != nil {
		hostID, clientID, agentName = options.Existing.HostID, options.Existing.ClientID, ""
	}
	if hostID == "" {
		hostID, err = newHostID()
		if err != nil {
			return Credentials{}, err
		}
	}

	results := make(chan callbackResult, 1)
	server := &http.Server{Handler: callbackHandler(results), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Close() }()

	authURL := oauth.authorizationURL(clientID, agentName, hostID, redirectURI, state, nonce, challenge(verifier))
	if err := options.OpenBrowser(authURL); err != nil {
		return Credentials{}, fmt.Errorf("open the browser: %w", err)
	}

	timeout := options.Timeout
	if timeout == 0 {
		timeout = loginTimeout
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var result callbackResult
	select {
	case result = <-results:
	case <-waitCtx.Done():
		return Credentials{}, fmt.Errorf("sign-in was not completed in time: %w", waitCtx.Err())
	}

	if result.errorCode != "" {
		return Credentials{}, fmt.Errorf("sign-in was refused: %s", result.errorCode)
	}
	if result.state != state {
		return Credentials{}, errors.New("sign-in callback has an unexpected state; start again")
	}
	if result.clientID != "" {
		clientID = result.clientID
	}
	if clientID == DynamicClientID {
		return Credentials{}, errors.New("the server did not issue a client id")
	}

	credentials, err := oauth.exchangeCode(ctx, clientID, result.code, verifier, redirectURI)
	if err != nil {
		return Credentials{}, err
	}
	credentials.HostID = hostID
	if !credentials.HasPlanScope() {
		return Credentials{}, errors.New("plan usage was not granted: allow \"use my ChatGPT plan\" on the consent screen")
	}
	return credentials, nil
}

// Принимает: results — канал для результата.
// Делает: отдаёт обработчик возврата: запоминает параметры один раз и показывает короткую страницу.
// Возвращает: http.Handler.
func callbackHandler(results chan<- callbackResult) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+callbackPath, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		result := callbackResult{
			code: query.Get("code"), state: query.Get("state"),
			clientID: query.Get("client_id"), errorCode: query.Get("error"),
		}
		select {
		case results <- result:
		default:
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		message := "Готово. Можно закрыть вкладку и вернуться в терминал."
		if result.errorCode != "" {
			message = "Вход отменён: " + html.EscapeString(result.errorCode)
		}
		_, _ = fmt.Fprintf(w, "<!doctype html><meta charset=utf-8><title>Вход</title><p>%s</p>", message)
	})
	return mux
}

// Принимает: параметры запроса входа; agentName нужен только при первой регистрации.
// Делает: собирает адрес страницы входа.
// Возвращает: полный URL.
func (oauth *OAuth) authorizationURL(clientID, agentName, hostID, redirectURI, state, nonce, codeChallenge string) string {
	query := url.Values{
		"client_id":             {clientID},
		"ext_agent_host_id":     {hostID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {loginScope},
		"resource":              {Resource},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	if agentName != "" {
		query.Set("agent_name_hint", agentName)
	}
	return oauth.AuthorizeURL + "?" + query.Encode()
}

// Принимает: ничего.
// Делает: генерирует UUID v4 в виде URN, как требует ext_agent_host_id.
// Возвращает: строку urn:uuid:... или ошибку источника случайности.
func newHostID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	id[6] = id[6]&0x0f | 0x40
	id[8] = id[8]&0x3f | 0x80
	return fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:]), nil
}

// Принимает: length — число случайных байт.
// Делает: читает криптостойкие байты и кодирует их в base64url без заполнения.
// Возвращает: строку или ошибку источника случайности.
func randomString(length int) (string, error) {
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
