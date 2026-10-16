package tools

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

// Принимает: raw — JSON от модели; destination — структура аргументов.
// Делает: строго разбирает JSON, неизвестные поля (в т.ч. user_id) запрещены.
// Возвращает: nil или ArgumentError.
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

// Принимает: raw — необязательная дата YYYY-MM-DD.
// Делает: разбирает дату, если она передана.
// Возвращает: указатель на день, nil для пустой даты либо ArgumentError.
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

// Принимает: properties; required — обязательные свойства.
// Делает: строит JSON Schema объекта без лишних свойств; паникует при сбое сериализации (ошибка программиста).
// Возвращает: схему.
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

func categoryProperty() map[string]any {
	categories := domain.Categories()
	keys := make([]string, 0, len(categories))
	for _, category := range categories {
		keys = append(keys, string(category))
	}
	return map[string]any{"type": "string", "enum": keys, "description": "Ключ категории из закрытого списка."}
}

func dateProperty(description string) map[string]any {
	return map[string]any{"type": "string", "description": description + " Формат YYYY-MM-DD."}
}

// amountText хранит сумму в рублях как текст из JSON: принимает и число (99.9), и строку ("99,90"), чтобы не проходить через float.
type amountText string

// Принимает: data — JSON-значение: число или строка.
// Делает: сохраняет литерал как текст без преобразования в число с плавающей точкой.
// Возвращает: nil либо ошибку, если значение не число и не строка.
func (amount *amountText) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		*amount = amountText(text)
		return nil
	}
	if len(data) == 0 || data[0] == '{' || data[0] == '[' || string(data) == "null" || string(data) == "true" || string(data) == "false" {
		return fmt.Errorf("amount must be a number or a string")
	}
	*amount = amountText(data)
	return nil
}

// amountProperty описывает свойство суммы: строка с рублями, чтобы число не превращалось в float по дороге.
func amountProperty(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"pattern":     `^\d+([.,]\d{1,2})?$`,
		"description": description + ` Строкой в рублях, не более двух знаков после запятой: "350", "99.90".`,
	}
}
