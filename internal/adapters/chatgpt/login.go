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
// Делает: поднимает callback-сервер, открывает браузер, проверяет ответ, меняет код по PKCE, требует право на подписку.
// Возвращает: новые данные или ошибку отказа, таймаута либо обмена кода.
func Login(ctx context.Context, oauth *OAuth, options LoginOptions) (Credentials, error) {
	secrets, err := newLoginSecrets()
	if err != nil {
		return Credentials{}, err
	}
	identity, err := resolveIdentity(options)
	if err != nil {
		return Credentials{}, err
	}
	callback, err := startCallbackServer(options.Port)
	if err != nil {
		return Credentials{}, err
	}
	defer callback.stop()

	authURL := oauth.authorizationURL(identity, callback.redirectURI, secrets)
	if err := options.OpenBrowser(authURL); err != nil {
		return Credentials{}, fmt.Errorf("open the browser: %w", err)
	}

	result, err := callback.await(ctx, options.Timeout)
	if err != nil {
		return Credentials{}, err
	}
	clientID, err := result.validate(secrets.state, identity.clientID)
	if err != nil {
		return Credentials{}, err
	}

	credentials, err := oauth.exchangeCode(ctx, clientID, result.code, secrets.verifier, callback.redirectURI)
	if err != nil {
		return Credentials{}, err
	}
	credentials.HostID = identity.hostID
	if !credentials.HasPlanScope() {
		return Credentials{}, errors.New("plan usage was not granted: allow \"use my ChatGPT plan\" on the consent screen")
	}
	return credentials, nil
}

type loginSecrets struct {
	verifier, state, nonce string
}

type loginIdentity struct {
	clientID, hostID, agentName string
}

type callbackServer struct {
	redirectURI string
	results     <-chan callbackResult
	stop        func()
}

// Принимает: ничего.
// Делает: генерирует одноразовые значения PKCE, state и nonce для одной попытки входа.
// Возвращает: набор секретов или ошибку источника случайности.
func newLoginSecrets() (loginSecrets, error) {
	var secrets loginSecrets
	var err error
	for target, length := range map[*string]int{&secrets.verifier: 48, &secrets.state: 24, &secrets.nonce: 24} {
		if *target, err = randomString(length); err != nil {
			return loginSecrets{}, err
		}
	}
	return secrets, nil
}

// Принимает: options — настройки входа.
// Делает: берёт идентификаторы клиента и хоста из прежнего входа либо готовит регистрацию с новым хостом.
// Возвращает: идентичность для запроса входа; имя приложения заполнено только при первой регистрации.
func resolveIdentity(options LoginOptions) (loginIdentity, error) {
	identity := loginIdentity{clientID: DynamicClientID, agentName: options.AgentName}
	if options.Existing != nil {
		identity = loginIdentity{clientID: options.Existing.ClientID, hostID: options.Existing.HostID}
	}
	if identity.hostID == "" {
		hostID, err := newHostID()
		if err != nil {
			return loginIdentity{}, err
		}
		identity.hostID = hostID
	}
	return identity, nil
}

// Принимает: port — порт на 127.0.0.1; ноль выбирает свободный.
// Делает: слушает порт и запускает сервер приёма ответа браузера.
// Возвращает: адрес возврата, канал результата и функцию остановки либо ошибку порта.
func startCallbackServer(port int) (callbackServer, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return callbackServer{}, fmt.Errorf("listen for the sign-in callback: %w", err)
	}
	results := make(chan callbackResult, 1)
	server := &http.Server{Handler: callbackHandler(results), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	return callbackServer{
		redirectURI: fmt.Sprintf("http://%s%s", listener.Addr().String(), callbackPath),
		results:     results,
		stop:        func() { _ = server.Close() },
	}, nil
}

// Принимает: ctx; timeout — сколько ждать, ноль означает loginTimeout.
// Делает: ждёт ответ браузера.
// Возвращает: параметры возврата либо ошибку таймаута или отмены.
func (callback callbackServer) await(ctx context.Context, timeout time.Duration) (callbackResult, error) {
	if timeout == 0 {
		timeout = loginTimeout
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case result := <-callback.results:
		return result, nil
	case <-waitCtx.Done():
		return callbackResult{}, fmt.Errorf("sign-in was not completed in time: %w", waitCtx.Err())
	}
}

// Принимает: state — ожидаемое значение; knownClientID — идентификатор клиента, известный до ответа.
// Делает: отклоняет отказ пользователя, чужой state и ответ без выданного идентификатора клиента.
// Возвращает: идентификатор клиента для обмена кода либо ошибку.
func (result callbackResult) validate(state, knownClientID string) (string, error) {
	if result.errorCode != "" {
		return "", fmt.Errorf("sign-in was refused: %s", result.errorCode)
	}
	if result.state != state {
		return "", errors.New("sign-in callback has an unexpected state; start again")
	}
	clientID := knownClientID
	if result.clientID != "" {
		clientID = result.clientID
	}
	if clientID == DynamicClientID {
		return "", errors.New("the server did not issue a client id")
	}
	return clientID, nil
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

// Принимает: identity — клиент и хост; redirectURI — адрес возврата; secrets — одноразовые значения попытки.
// Делает: собирает адрес страницы входа; имя приложения передаётся только при первой регистрации.
// Возвращает: полный URL.
func (oauth *OAuth) authorizationURL(identity loginIdentity, redirectURI string, secrets loginSecrets) string {
	query := url.Values{
		"client_id":             {identity.clientID},
		"ext_agent_host_id":     {identity.hostID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {loginScope},
		"resource":              {Resource},
		"state":                 {secrets.state},
		"nonce":                 {secrets.nonce},
		"code_challenge":        {challenge(secrets.verifier)},
		"code_challenge_method": {"S256"},
	}
	if identity.agentName != "" {
		query.Set("agent_name_hint", identity.agentName)
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
