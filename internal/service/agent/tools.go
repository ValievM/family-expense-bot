package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

// Tool — команда, которую модель может вызвать; владельца данных инструмент получает только из scope, а не из аргументов.
type Tool interface {
	// Принимает: ничего.
	// Делает: описывает инструмент для модели: имя, назначение и JSON Schema аргументов.
	// Возвращает: ToolSpec без поля владельца.
	Spec() ToolSpec
	// Принимает: ctx — контекст запроса; expenses — траты, привязанные к владельцу сообщения; arguments — сырой JSON от модели.
	// Делает: проверяет аргументы и выполняет операцию только над данными владельца.
	// Возвращает: результат для сериализации в ответ модели либо ошибку.
	Execute(ctx context.Context, expenses UserExpenses, arguments json.RawMessage) (any, error)
}

// ArgumentError — ошибка аргументов, которую модель может исправить повторным вызовом.
type ArgumentError struct {
	Message string
}

// Принимает: ничего.
// Делает: отдаёт текст ошибки аргументов.
// Возвращает: сообщение для модели.
func (err ArgumentError) Error() string {
	return err.Message
}

// ToolSet — упорядоченный набор инструментов с поиском по имени; новый инструмент добавляется без правок цикла агента.
type ToolSet struct {
	ordered []Tool
	byName  map[string]Tool
}

// Принимает: tools — инструменты в порядке показа модели.
// Делает: индексирует инструменты по имени из Spec.
// Возвращает: готовый набор.
func NewToolSet(tools ...Tool) *ToolSet {
	set := &ToolSet{ordered: tools, byName: make(map[string]Tool, len(tools))}
	for _, tool := range tools {
		set.byName[tool.Spec().Name] = tool
	}
	return set
}

// Принимает: location — часовой пояс для разбора и показа дат.
// Делает: собирает стандартные инструменты учёта: добавить, показать, изменить, удалить.
// Возвращает: набор из четырёх инструментов.
func DefaultTools(location *time.Location) *ToolSet {
	return NewToolSet(
		addExpenseTool{location: location},
		listExpensesTool{location: location},
		updateExpenseTool{location: location},
		deleteExpenseTool{location: location},
	)
}

// Принимает: ничего.
// Делает: собирает описания всех инструментов для запроса к модели.
// Возвращает: срез ToolSpec в порядке регистрации.
func (set *ToolSet) Specs() []ToolSpec {
	specs := make([]ToolSpec, 0, len(set.ordered))
	for _, tool := range set.ordered {
		specs = append(specs, tool.Spec())
	}
	return specs
}

// Принимает: name — имя инструмента из вызова модели.
// Делает: ищет инструмент в наборе.
// Возвращает: инструмент и true, если он найден.
func (set *ToolSet) Get(name string) (Tool, bool) {
	tool, found := set.byName[name]
	return tool, found
}

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
		return ArgumentError{Message: "invalid arguments: " + err.Error()}
	}
	return nil
}

// Принимает: raw — необязательная дата YYYY-MM-DD от модели; location — часовой пояс календаря.
// Делает: разбирает дату, если она передана.
// Возвращает: указатель на дату, nil для отсутствующей даты либо ArgumentError при неверном формате.
func parseDateArgument(raw *string, location *time.Location) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	parsed, err := domain.ParseDate(*raw, location)
	if err != nil {
		return nil, ArgumentError{Message: fmt.Sprintf("date %q must use the YYYY-MM-DD format", *raw)}
	}
	return &parsed, nil
}

// Принимает: err — ошибка выполнения инструмента.
// Делает: отличает ошибки, которые модель способна исправить сама, от сбоев инфраструктуры.
// Возвращает: true для ошибок аргументов, валидации и «не найдено».
func isRecoverable(err error) bool {
	var argumentErr ArgumentError
	return errors.As(err, &argumentErr) ||
		errors.Is(err, domain.ErrInvalidAmount) ||
		errors.Is(err, domain.ErrInvalidUser) ||
		errors.Is(err, domain.ErrUnknownCategory) ||
		errors.Is(err, domain.ErrInvalidDate) ||
		errors.Is(err, expense.ErrInvalidPeriod) ||
		errors.Is(err, expense.ErrNotFound)
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
	return map[string]any{"type": "string", "enum": keys, "description": "Ключ категории: " + categoryGuide()}
}

// Принимает: description — пояснение к дате.
// Делает: формирует описание свойства даты в формате YYYY-MM-DD.
// Возвращает: фрагмент JSON Schema.
func dateProperty(description string) map[string]any {
	return map[string]any{"type": "string", "description": description + " Формат YYYY-MM-DD."}
}
