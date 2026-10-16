// Package tools содержит инструменты ИИ-агента учёта расходов; ядро агента о них не знает и получает набор снаружи.
package tools

import "github.com/ValievM/family-expense-bot/internal/service/agent"

// Принимает: ничего.
// Делает: собирает стандартные инструменты учёта: добавить, показать, изменить, удалить.
// Возвращает: набор из четырёх инструментов для agent.New.
func Default() *agent.ToolSet {
	return agent.NewToolSet(
		addExpenseTool{},
		listExpensesTool{},
		updateExpenseTool{},
		deleteExpenseTool{},
	)
}
