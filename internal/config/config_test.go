package config_test

import (
	"strings"
	"testing"

	"github.com/ValievM/family-expense-bot/internal/config"
)

// Принимает: values — переменные окружения для теста.
// Делает: строит функцию чтения окружения поверх словаря.
// Возвращает: функцию, совместимую с config.Load.
func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

// Принимает: t — тестовый контекст.
// Делает: проверяет значения по умолчанию при минимальном окружении.
// Возвращает: ничего.
func TestLoadAppliesDefaults(t *testing.T) {
	cfg, err := config.Load(env(map[string]string{"DATABASE_URL": "postgres://db"}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != ":8080" || cfg.LLMModel != "gpt-4.1-mini" || cfg.Location.String() != "Europe/Moscow" || cfg.TelegramEnabled {
		t.Errorf("cfg = %+v", cfg)
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет таблицу ошибочных конфигураций, включая несколько проблем сразу.
// Возвращает: ничего.
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
		{"bot without openai key", map[string]string{"DATABASE_URL": "x", "TELEGRAM_BOT_TOKEN": "t"}, []string{"LLM_API_KEY"}},
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

// Принимает: t — тестовый контекст.
// Делает: проверяет, что локальная модель без ключа допустима и что список разрешённых пользователей работает.
// Возвращает: ничего.
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

// Принимает: t — тестовый контекст.
// Делает: проверяет, что пустой список разрешённых пользователей пускает всех.
// Возвращает: ничего.
func TestEmptyAllowListAllowsEveryone(t *testing.T) {
	cfg, err := config.Load(env(map[string]string{"DATABASE_URL": "x"}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.IsUserAllowed(123456) {
		t.Error("IsUserAllowed() = false, want true when ALLOWED_USER_IDS is empty")
	}
}
