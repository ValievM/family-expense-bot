// Package tools содержит инструменты ИИ-агента учёта расходов; ядро агента о них не знает и получает набор снаружи.
package tools

import (
	"time"

	"github.com/ValievM/family-expense-bot/internal/service/agent"
)

// Принимает: location — часовой пояс для разбора и показа дат.
// Делает: собирает стандартные инструменты учёта: добавить, показать, изменить, удалить.
// Возвращает: набор из четырёх инструментов для agent.New.
func Default(location *time.Location) *agent.ToolSet {
	return agent.NewToolSet(
		addExpenseTool{location: location},
		listExpensesTool{location: location},
		updateExpenseTool{location: location},
		deleteExpenseTool{location: location},
	)
}
