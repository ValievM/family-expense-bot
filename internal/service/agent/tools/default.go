// Package tools — инструменты ИИ-агента; ядро агента получает набор снаружи.
package tools

import "github.com/ValievM/family-expense-bot/internal/service/agent"

func Default() *agent.ToolSet {
	return agent.NewToolSet(
		addExpenseTool{},
		listExpensesTool{},
		updateExpenseTool{},
		deleteExpenseTool{},
	)
}
