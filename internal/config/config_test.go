package config_test

import (
	"strings"
	"testing"

	"github.com/ValievM/family-expense-bot/internal/config"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

// Проверяет: значения по умолчанию при минимальном окружении.
func TestLoadAppliesDefaults(t *testing.T) {
	cfg, err := config.Load(env(map[string]string{"DATABASE_URL": "postgres://db"}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != ":8080" || cfg.LLMModel != "glm-5.3-flash" || cfg.Location.String() != "Europe/Moscow" || cfg.TelegramEnabled {
		t.Errorf("cfg = %+v", cfg)
	}
}

// Проверяет: ошибочные конфигурации отклоняются, несколько проблем сообщаются сразу.
func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
		want   []string
	}{
		{"missing database", map[string]string{}, []string{"DATABASE_URL"}},
		{"bad time zone", map[string]string{"DATABASE_URL": "x", "TZ": "Mars/Base"}, []string{"TZ"}},
		{"bad allowed id", map[string]string{"DATABASE_URL": "x", "ALLOWED_USER_IDS": "1,abc"}, []string{"ALLOWED_USER_IDS"}},
		{"negative allowed id", map[string]string{"DATABASE_URL": "x", "ALLOWED_USER_IDS": "-5"}, []string{"ALLOWED_USER_IDS"}},
		{"bot without key", map[string]string{"DATABASE_URL": "x", "TELEGRAM_BOT_TOKEN": "t"}, []string{"LLM_API_KEY"}},
		{"several problems", map[string]string{"TZ": "Mars/Base"}, []string{"DATABASE_URL", "TZ"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Load(env(tt.values))

			if err == nil {
				t.Fatal("Load() error = nil, want a validation error")
			}
			for _, fragment := range tt.want {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("error %q does not mention %s", err, fragment)
				}
			}
		})
	}
}

// Проверяет: endpoint без ключа допустим, список разрешённых пользователей фильтрует.
func TestLoadAllowsKeylessCustomEndpointAndFiltersUsers(t *testing.T) {
	cfg, err := config.Load(env(map[string]string{
		"DATABASE_URL":       "x",
		"TELEGRAM_BOT_TOKEN": "t",
		"LLM_BASE_URL":       "http://ollama:11434/v1",
		"ALLOWED_USER_IDS":   " 10, 20 ",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.TelegramEnabled || !cfg.IsUserAllowed(10) || !cfg.IsUserAllowed(20) || cfg.IsUserAllowed(30) {
		t.Errorf("cfg = %+v", cfg)
	}
}

// Проверяет: пустой список разрешённых пользователей пускает всех.
func TestEmptyAllowListAllowsEveryone(t *testing.T) {
	cfg, err := config.Load(env(map[string]string{"DATABASE_URL": "x"}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.IsUserAllowed(123456) {
		t.Error("IsUserAllowed() = false, want true when ALLOWED_USER_IDS is empty")
	}
}
