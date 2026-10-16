package tools

import (
	"context"
	"encoding/json"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

// deleteExpenseTool удаляет трату владельца сообщения.
type deleteExpenseTool struct{}

// deleteExpenseArguments — аргументы delete_expense.
type deleteExpenseArguments struct {
	ID int64 `json:"id"`
}

// Принимает: ничего.
// Делает: описывает инструмент delete_expense с обязательным id.
// Возвращает: ToolSpec.
func (tool deleteExpenseTool) Spec() ports.ToolSpec {
	return ports.ToolSpec{
		Name:        "delete_expense",
		Description: "Удалить трату пользователя по id. id берётся из list_expenses; при нескольких кандидатах сначала уточни у пользователя.",
		Parameters: objectSchema(map[string]any{
			"id": map[string]any{"type": "integer", "description": "id траты из list_expenses."},
		}, "id"),
	}
}

// Принимает: ctx — контекст запроса; expenses — траты владельца сообщения; arguments — JSON от модели.
// Делает: находит трату владельца, удаляет её и возвращает описание удалённого для подтверждения пользователю.
// Возвращает: карту с признаком deleted и данными траты либо ошибку аргументов или «не найдено».
func (tool deleteExpenseTool) Execute(ctx context.Context, expenses ports.UserExpenses, arguments json.RawMessage) (any, error) {
	var args deleteExpenseArguments
	if err := decodeArguments(arguments, &args); err != nil {
		return nil, err
	}
	if args.ID <= 0 {
		return nil, ports.ArgumentError{Message: "id must be a positive integer"}
	}

	item, err := expenses.Get(ctx, domain.ExpenseID(args.ID))
	if err != nil {
		return nil, err
	}
	if err := expenses.Delete(ctx, item.ID()); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "expense": newExpenseView(item)}, nil
}
