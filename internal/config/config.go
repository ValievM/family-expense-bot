// Package config — настройки из окружения, общий источник конфигурации для адаптеров.
package config

import (
	"os"
	"strings"
)

// defaultTimeZone — часовой пояс по умолчанию, если TZ не задан.
const defaultTimeZone = "Europe/Moscow"

// Config — настройки, общие для адаптеров приложения.
type Config struct {
	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string
	TimeZone   string
}

// Принимает: ничего.
// Делает: читает LLM_BASE_URL, LLM_API_KEY, LLM_MODEL и TZ из окружения; пустой TZ после обрезки пробелов заменяется на Europe/Moscow.
// Возвращает: заполненную структуру Config.
func Load() Config {
	timeZone := strings.TrimSpace(os.Getenv("TZ"))
	if timeZone == "" {
		timeZone = defaultTimeZone
	}

	return Config{
		LLMBaseURL: os.Getenv("LLM_BASE_URL"),
		LLMAPIKey:  os.Getenv("LLM_API_KEY"),
		LLMModel:   os.Getenv("LLM_MODEL"),
		TimeZone:   timeZone,
	}
}
