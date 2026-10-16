package llm

import (
	"context"
	"net/http"
)

type Authorizer interface {
	Authorize(ctx context.Context, request *http.Request) error
}

// APIKeyAuthorizer — авторизация статическим bearer-ключом.
type APIKeyAuthorizer struct {
	key string
}

// Принимает: key — API-ключ провайдера.
// Делает: сохраняет ключ, который подставляется в каждый запрос.
// Возвращает: APIKeyAuthorizer с заданным ключом.
func NewAPIKeyAuthorizer(key string) APIKeyAuthorizer {
	return APIKeyAuthorizer{key: key}
}

// Принимает: _ — контекст (не используется); request — исходящий HTTP-запрос.
// Делает: ставит заголовок Authorization: Bearer; при пустом ключе заголовок не ставится (так работают локальные серверы вроде Ollama).
// Возвращает: всегда nil.
func (authorizer APIKeyAuthorizer) Authorize(_ context.Context, request *http.Request) error {
	if authorizer.key != "" {
		request.Header.Set("Authorization", "Bearer "+authorizer.key)
	}
	return nil
}
