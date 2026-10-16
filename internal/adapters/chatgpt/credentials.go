// Package chatgpt реализует вход через аккаунт ChatGPT (OAuth с PKCE) и хранение токенов для оплаты запросов подпиской.
package chatgpt

import (
	"errors"
	"strings"
	"time"
)

const (
	// PlanScope — право расходовать подписку ChatGPT на запросы к модели.
	PlanScope = "chatgpt.tokens.use.direct"
	Resource  = "https://api.openai.com/v1"
	// DynamicClientID нужен только при первой регистрации; дальше хранится выданный.
	DynamicClientID = "dynamic_agent_client"
)

var ErrNotLoggedIn = errors.New("not signed in with ChatGPT: run `make chatgpt-login`")

var ErrReauthRequired = errors.New("ChatGPT sign-in expired: run `make chatgpt-login` again")

type Credentials struct {
	ClientID     string    `json:"client_id"`
	HostID       string    `json:"ext_agent_host_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope"`
}

func (credentials Credentials) HasPlanScope() bool {
	for _, scope := range strings.Fields(credentials.Scope) {
		if scope == PlanScope {
			return true
		}
	}
	return false
}

// Принимает: now — текущее время; window — запас до истечения.
// Делает: определяет, пора ли обновлять токен доступа.
// Возвращает: true, если токен уже истёк или истечёт в пределах window.
func (credentials Credentials) ExpiresWithin(now time.Time, window time.Duration) bool {
	return !now.Add(window).Before(credentials.ExpiresAt)
}
