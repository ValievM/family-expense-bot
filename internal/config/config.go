// Package config centralizes environment settings so adapters share one configuration source.
package config

import (
	"os"
	"strings"
)

const defaultTimeZone = "Europe/Moscow"

// Config contains settings shared by the application adapters.
type Config struct {
	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string
	TimeZone   string
}

// Load reads the application settings from the environment.
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
