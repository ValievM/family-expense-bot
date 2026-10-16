package agent

import (
	"errors"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

type ToolSet struct {
	ordered []ports.Tool
	byName  map[string]ports.Tool
}

func NewToolSet(tools ...ports.Tool) *ToolSet {
	set := &ToolSet{ordered: tools, byName: make(map[string]ports.Tool, len(tools))}
	for _, tool := range tools {
		set.byName[tool.Spec().Name] = tool
	}
	return set
}

func (set *ToolSet) Specs() []ports.ToolSpec {
	specs := make([]ports.ToolSpec, 0, len(set.ordered))
	for _, tool := range set.ordered {
		specs = append(specs, tool.Spec())
	}
	return specs
}

func (set *ToolSet) Get(name string) (ports.Tool, bool) {
	tool, found := set.byName[name]
	return tool, found
}

// Принимает: err — ошибка выполнения инструмента.
// Делает: отличает ошибки, которые модель исправит сама, от сбоев инфраструктуры.
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
