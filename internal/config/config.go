// Package config читает настройки приложения из переменных окружения и проверяет их при старте.
package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeZone   = "Europe/Moscow"
	defaultHTTPAddr   = ":8080"
	defaultLLMBaseURL = "https://api.openai.com/v1"
	defaultLLMModel   = "gpt-4.1-mini"
	defaultAuthFile   = "data/chatgpt-auth.json"

	// AuthAPIKey — доступ к модели по ключу API.
	AuthAPIKey = "apikey"
	// AuthOAuth — доступ по входу через аккаунт ChatGPT (подписка).
	AuthOAuth = "oauth"
)

// Config — все настройки приложения; адаптеры получают из него только нужные им поля.
type Config struct {
	DatabaseURL      string
	HTTPAddr         string
	TelegramToken    string
	AllowedUserIDs   map[int64]struct{}
	LLMBaseURL       string
	LLMAPIKey        string
	LLMModel         string
	LLMAuth          string
	LLMAuthFile      string
	Location         *time.Location
	TelegramEnabled  bool
	RequestedZoneKey string
}

// Принимает: getenv — функция чтения переменной окружения (os.Getenv в приложении, словарь в тестах).
// Делает: читает настройки, подставляет значения по умолчанию и проверяет обязательные и взаимозависимые поля.
// Возвращает: готовую Config либо ошибку со всеми найденными проблемами сразу.
func Load(getenv func(string) string) (Config, error) {
	read := func(key, fallback string) string {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			return value
		}
		return fallback
	}

	cfg := Config{
		DatabaseURL:      read("DATABASE_URL", ""),
		HTTPAddr:         read("HTTP_ADDR", defaultHTTPAddr),
		TelegramToken:    read("TELEGRAM_BOT_TOKEN", ""),
		LLMBaseURL:       read("LLM_BASE_URL", defaultLLMBaseURL),
		LLMAPIKey:        read("LLM_API_KEY", ""),
		LLMAuth:          strings.ToLower(read("LLM_AUTH", AuthAPIKey)),
		LLMAuthFile:      read("LLM_AUTH_FILE", defaultAuthFile),
		RequestedZoneKey: read("TZ", defaultTimeZone),
	}
	cfg.TelegramEnabled = cfg.TelegramToken != ""
	// В режиме подписки модель по умолчанию выбирается из списка, доступного аккаунту.
	if cfg.LLMAuth == AuthAPIKey {
		cfg.LLMModel = read("LLM_MODEL", defaultLLMModel)
	} else {
		cfg.LLMModel = read("LLM_MODEL", "")
	}

	var problems []error
	if cfg.DatabaseURL == "" {
		problems = append(problems, errors.New("DATABASE_URL is required"))
	}

	location, err := time.LoadLocation(cfg.RequestedZoneKey)
	if err != nil {
		problems = append(problems, fmt.Errorf("TZ %q is not a valid time zone: %w", cfg.RequestedZoneKey, err))
	}
	cfg.Location = location

	allowed, err := parseUserIDs(getenv("ALLOWED_USER_IDS"))
	if err != nil {
		problems = append(problems, err)
	}
	cfg.AllowedUserIDs = allowed

	if cfg.LLMAuth != AuthAPIKey && cfg.LLMAuth != AuthOAuth {
		problems = append(problems, fmt.Errorf("LLM_AUTH must be %q or %q, got %q", AuthAPIKey, AuthOAuth, cfg.LLMAuth))
	}
	// Без ключа можно работать только с совместимыми серверами без авторизации, например с локальной Ollama.
	if cfg.TelegramEnabled && cfg.LLMAuth == AuthAPIKey && cfg.LLMAPIKey == "" && cfg.LLMBaseURL == defaultLLMBaseURL {
		problems = append(problems, errors.New("LLM_API_KEY is required when the Telegram bot uses the OpenAI API"))
	}

	if len(problems) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(problems...))
	}
	return cfg, nil
}

// Принимает: raw — строка Telegram ID через запятую, например "1, 2".
// Делает: разбирает список разрешённых пользователей; пустая строка означает «разрешены все».
// Возвращает: множество ID либо ошибку для нечисловых и неположительных значений.
func parseUserIDs(raw string) (map[int64]struct{}, error) {
	ids := make(map[int64]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("ALLOWED_USER_IDS contains invalid Telegram ID %q", part)
		}
		ids[id] = struct{}{}
	}
	return ids, nil
}

// Принимает: userID — Telegram ID автора сообщения.
// Делает: проверяет доступ по списку ALLOWED_USER_IDS; пустой список разрешает всех.
// Возвращает: true, если пользователю можно пользоваться ботом.
func (cfg Config) IsUserAllowed(userID int64) bool {
	if len(cfg.AllowedUserIDs) == 0 {
		return true
	}
	_, allowed := cfg.AllowedUserIDs[userID]
	return allowed
}
