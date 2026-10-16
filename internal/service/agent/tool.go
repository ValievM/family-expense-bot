package agent

import (
	"errors"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

// ToolSet — упорядоченный набор инструментов с поиском по имени; новый инструмент добавляется без правок цикла агента.
type ToolSet struct {
	ordered []ports.Tool
	byName  map[string]ports.Tool
}

// Принимает: tools — инструменты в порядке показа модели.
// Делает: индексирует инструменты по имени из Spec.
// Возвращает: готовый набор.
func NewToolSet(tools ...ports.Tool) *ToolSet {
	set := &ToolSet{ordered: tools, byName: make(map[string]ports.Tool, len(tools))}
	for _, tool := range tools {
		set.byName[tool.Spec().Name] = tool
	}
	return set
}

// Принимает: ничего.
// Делает: собирает описания всех инструментов для запроса к модели.
// Возвращает: срез ToolSpec в порядке регистрации.
func (set *ToolSet) Specs() []ports.ToolSpec {
	specs := make([]ports.ToolSpec, 0, len(set.ordered))
	for _, tool := range set.ordered {
		specs = append(specs, tool.Spec())
	}
	return specs
}

// Принимает: name — имя инструмента из вызова модели.
// Делает: ищет инструмент в наборе.
// Возвращает: инструмент и true, если он найден.
func (set *ToolSet) Get(name string) (ports.Tool, bool) {
	tool, found := set.byName[name]
	return tool, found
}

// Принимает: err — ошибка выполнения инструмента.
// Делает: отличает ошибки, которые модель способна исправить сама, от сбоев инфраструктуры.
// Возвращает: true для ошибок аргументов, валидации и «не найдено».
func isRecoverable(err error) bool {
	var argumentErr ports.ArgumentError
	return errors.As(err, &argumentErr) ||
		errors.Is(err, domain.ErrInvalidAmount) ||
		errors.Is(err, domain.ErrInvalidUser) ||
		errors.Is(err, domain.ErrUnknownCategory) ||
		errors.Is(err, domain.ErrInvalidDate) ||
		errors.Is(err, domain.ErrCommentTooLong) ||
		errors.Is(err, ports.ErrInvalidPeriod) ||
		errors.Is(err, ports.ErrNotFound)
}
