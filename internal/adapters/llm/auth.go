package llm

import (
	"context"
	"net/http"
)

// Authorizer attaches credentials to an outgoing provider request.
// Alternative sign-in schemes plug in here without touching the client.
type Authorizer interface {
	Authorize(ctx context.Context, request *http.Request) error
}

// APIKeyAuthorizer authenticates with a static bearer key.
type APIKeyAuthorizer struct {
	key string
}

// NewAPIKeyAuthorizer stores the key used for every request.
func NewAPIKeyAuthorizer(key string) APIKeyAuthorizer {
	return APIKeyAuthorizer{key: key}
}

// Authorize sets the bearer header; an empty key sends none, which local servers such as Ollama accept.
func (authorizer APIKeyAuthorizer) Authorize(_ context.Context, request *http.Request) error {
	if authorizer.key != "" {
		request.Header.Set("Authorization", "Bearer "+authorizer.key)
	}
	return nil
}
