package chatgpt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// refreshWindow — за сколько до истечения токен обновляется заранее.
const refreshWindow = 2 * time.Minute

type Store interface {
	Load() (Credentials, error)
	Save(credentials Credentials) error
}

// TokenSource выдаёт актуальный токен доступа и сам обновляет его; одновременные обновления выполняются по очереди, чтобы не потерять вращаемый токен.
type TokenSource struct {
	mutex       sync.Mutex
	store       Store
	oauth       *OAuth
	credentials Credentials
	loaded      bool
}

// Принимает: store — хранилище; oauth — клиент сервера авторизации.
// Делает: создаёт источник; файл читается лениво при первом запросе токена.
// Возвращает: TokenSource.
func NewTokenSource(store Store, oauth *OAuth) *TokenSource {
	return &TokenSource{store: store, oauth: oauth}
}

// Принимает: ctx — контекст запроса.
// Делает: читает данные при первом вызове, обновляет токен, если он скоро истечёт, и сохраняет новую пару до выдачи.
// Возвращает: токен доступа, ErrNotLoggedIn, ErrReauthRequired при отозванном токене обновления либо ошибку сети.
func (source *TokenSource) Token(ctx context.Context) (string, error) {
	source.mutex.Lock()
	defer source.mutex.Unlock()

	if !source.loaded {
		credentials, err := source.store.Load()
		if err != nil {
			return "", err
		}
		source.credentials, source.loaded = credentials, true
	}
	if !source.credentials.HasPlanScope() {
		return "", errors.New("ChatGPT sign-in has no plan-usage scope: sign in again and allow plan usage")
	}
	if !source.credentials.ExpiresWithin(source.oauth.Now(), refreshWindow) {
		return source.credentials.AccessToken, nil
	}

	refreshed, err := source.oauth.Refresh(ctx, source.credentials)
	if err != nil {
		if isTerminalRefreshError(err) {
			return "", fmt.Errorf("%w: %w", ErrReauthRequired, err)
		}
		return "", fmt.Errorf("refresh ChatGPT token: %w", err)
	}
	// Сначала на диск: токен обновления вращается, потеря новой пары означала бы повторный вход.
	if err := source.store.Save(refreshed); err != nil {
		return "", err
	}
	source.credentials = refreshed
	return refreshed.AccessToken, nil
}

// Принимает: ctx — контекст запроса; request — исходящий запрос к API.
// Делает: подставляет актуальный токен в заголовок Authorization; реализует интерфейс Authorizer клиента LLM.
// Возвращает: nil либо ошибку получения токена.
func (source *TokenSource) Authorize(ctx context.Context, request *http.Request) error {
	token, err := source.Token(ctx)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	return nil
}
