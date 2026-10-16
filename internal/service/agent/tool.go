package agent

import (
	"context"
	"encoding/json"
	"errors"

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
