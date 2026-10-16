package tools

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

// Принимает: raw — JSON аргументов от модели; destination — указатель на структуру аргументов.
// Делает: строго разбирает JSON и запрещает неизвестные поля, поэтому попытка передать user_id отклоняется.
// Возвращает: nil или ArgumentError с описанием проблемы.
func decodeArguments(raw json.RawMessage, destination any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return ports.ArgumentError{Message: "invalid arguments: " + err.Error()}
	}
	return nil
}

// Принимает: raw — необязательная дата YYYY-MM-DD от модели.
// Делает: разбирает дату, если она передана.
// Возвращает: указатель на день, nil для отсутствующей даты либо ArgumentError при неверном формате.
func parseDateArgument(raw *string) (*domain.Day, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	parsed, err := domain.ParseDay(*raw)
	if err != nil {
		return nil, ports.ArgumentError{Message: fmt.Sprintf("date %q must use the YYYY-MM-DD format", *raw)}
	}
	return &parsed, nil
}

// Принимает: properties — описание свойств объекта; required — имена обязательных свойств.
// Делает: строит JSON Schema объекта без дополнительных свойств; при невозможности сериализовать паникует, потому что это ошибка программиста при старте.
// Возвращает: готовую схему.
func objectSchema(properties map[string]any, required ...string) json.RawMessage {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		panic(fmt.Sprintf("agent: tool schema is not serializable: %v", err))
	}
	return encoded
}

// Принимает: ничего.
// Делает: формирует описание свойства «категория» из единственного источника категорий в домене.
// Возвращает: фрагмент JSON Schema с enum из ключей категорий.
func categoryProperty() map[string]any {
	categories := domain.Categories()
	keys := make([]string, 0, len(categories))
	for _, category := range categories {
		keys = append(keys, string(category))
	}
	return map[string]any{"type": "string", "enum": keys, "description": "Ключ категории из закрытого списка."}
}

// Принимает: description — пояснение к дате.
// Делает: формирует описание свойства даты в формате YYYY-MM-DD.
// Возвращает: фрагмент JSON Schema.
func dateProperty(description string) map[string]any {
	return map[string]any{"type": "string", "description": description + " Формат YYYY-MM-DD."}
}
