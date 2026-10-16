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
	defaultLLMBaseURL = "https://api.z.ai/api/paas/v4"
	defaultLLMModel   = "glm-5.3-flash"
)

type Config struct {
	DatabaseURL     string
	HTTPAddr        string
	TelegramToken   string
	AllowedUserIDs  map[int64]struct{}
	LLMBaseURL      string
	LLMAPIKey       string
	LLMModel        string
	Location        *time.Location
	TelegramEnabled bool
}

// Принимает: getenv — чтение переменной окружения (os.Getenv или словарь в тестах).
// Делает: читает настройки, подставляет значения по умолчанию, проверяет обязательные и связанные поля.
// Возвращает: Config либо ошибку со всеми найденными проблемами сразу.
func Load(getenv func(string) string) (Config, error) {
	cfg := readSettings(getenv)

	problems := cfg.validate()
	var err error
	if cfg.AllowedUserIDs, err = parseUserIDs(getenv("ALLOWED_USER_IDS")); err != nil {
		problems = append(problems, err)
	}
	zone := readOr(getenv, "TZ", defaultTimeZone)
	if cfg.Location, err = time.LoadLocation(zone); err != nil {
		problems = append(problems, fmt.Errorf("TZ %q is not a valid time zone: %w", zone, err))
	}

	if len(problems) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(problems...))
	}
	return cfg, nil
}

func readOr(getenv func(string) string, key, fallback string) string {
	if value := strings.TrimSpace(getenv(key)); value != "" {
		return value
	}
	return fallback
}

// Принимает: getenv — чтение переменной окружения.
// Делает: читает строковые настройки с значениями по умолчанию; без проверок.
// Возвращает: частично заполненный Config (без Location и AllowedUserIDs).
func readSettings(getenv func(string) string) Config {
	cfg := Config{
		DatabaseURL:   readOr(getenv, "DATABASE_URL", ""),
		HTTPAddr:      readOr(getenv, "HTTP_ADDR", defaultHTTPAddr),
		TelegramToken: readOr(getenv, "TELEGRAM_BOT_TOKEN", ""),
		LLMBaseURL:    readOr(getenv, "LLM_BASE_URL", defaultLLMBaseURL),
		LLMAPIKey:     readOr(getenv, "LLM_API_KEY", ""),
	}
	cfg.TelegramEnabled = cfg.TelegramToken != ""
	cfg.LLMModel = readOr(getenv, "LLM_MODEL", defaultLLMModel)
	return cfg
}

// Принимает: ничего.
// Делает: проверяет обязательные и связанные настройки.
// Возвращает: найденные проблемы (пусто, если всё в порядке).
func (cfg Config) validate() []error {
	var problems []error
	if cfg.DatabaseURL == "" {
		problems = append(problems, errors.New("DATABASE_URL is required"))
	}
	// Без ключа можно работать только с совместимыми серверами без авторизации, например с локальной Ollama.
	if cfg.TelegramEnabled && cfg.LLMAPIKey == "" && cfg.LLMBaseURL == defaultLLMBaseURL {
		problems = append(problems, errors.New("LLM_API_KEY is required when the bot uses the default LLM endpoint"))
	}
	return problems
}

// Принимает: raw — строка Telegram ID через запятую, например "1, 2".
// Делает: разбирает список; пустая строка означает «разрешены все».
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
// Делает: проверяет доступ по ALLOWED_USER_IDS; пустой список разрешает всех.
// Возвращает: true, если пользователь допущен.
func (cfg Config) IsUserAllowed(userID int64) bool {
	if len(cfg.AllowedUserIDs) == 0 {
		return true
	}
	_, allowed := cfg.AllowedUserIDs[userID]
	return allowed
}
