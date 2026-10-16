package llm

import (
	"context"
	"net/http"
)

type Authorizer interface {
	Authorize(ctx context.Context, request *http.Request) error
}

type APIKeyAuthorizer struct {
	key string
}

// Принимает: key — API-ключ провайдера.
// Делает: запоминает ключ для подстановки в каждый запрос.
// Возвращает: APIKeyAuthorizer.
func NewAPIKeyAuthorizer(key string) APIKeyAuthorizer {
	return APIKeyAuthorizer{key: key}
}

// Принимает: _ — контекст (не используется); request — исходящий запрос.
// Делает: ставит Authorization: Bearer; при пустом ключе не ставит (локальные серверы вроде Ollama).
// Возвращает: всегда nil.
func (authorizer APIKeyAuthorizer) Authorize(_ context.Context, request *http.Request) error {
	if authorizer.key != "" {
		request.Header.Set("Authorization", "Bearer "+authorizer.key)
	}
	return nil
}
