package tools

import (
	"context"
	"encoding/json"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

type updateExpenseTool struct{}

// Владельца задаёт только область UserExpenses.
type updateExpenseArguments struct {
	ID       int64       `json:"id"`
	Amount   *amountText `json:"amount"`
	Category *string     `json:"category"`
	Date     *string     `json:"date"`
	Comment  *string     `json:"comment"`
}

func (tool updateExpenseTool) Spec() ports.ToolSpec {
	return ports.ToolSpec{
		Name:        "update_expense",
		Description: "Изменить трату пользователя по id. Передавай только изменяемые поля. id берётся из list_expenses.",
		Parameters: objectSchema(map[string]any{
			"id":       map[string]any{"type": "integer", "description": "id траты из list_expenses."},
			"amount":   map[string]any{"type": "number", "description": "Новая сумма в рублях."},
			"category": categoryProperty(),
			"date":     dateProperty("Новая дата траты."),
			"comment":  map[string]any{"type": "string", "description": "Новый комментарий; пустая строка очищает его."},
		}, "id"),
	}
}

// Принимает: ctx; expenses; arguments — JSON от модели.
// Делает: собирает частичное изменение и применяет к трате владельца.
// Возвращает: expenseView обновлённой траты либо ошибку аргументов, валидации или «не найдено».
func (tool updateExpenseTool) Execute(ctx context.Context, expenses ports.UserExpenses, arguments json.RawMessage) (any, error) {
	var args updateExpenseArguments
	if err := decodeArguments(arguments, &args); err != nil {
		return nil, err
	}
	if args.ID <= 0 {
		return nil, ports.ArgumentError{Message: "id must be a positive integer"}
	}
	changes, err := tool.buildChanges(args)
	if err != nil {
		return nil, err
	}

	updated, err := expenses.Update(ctx, domain.ExpenseID(args.ID), changes)
	if err != nil {
		return nil, err
	}
	return newExpenseView(updated), nil
}

// Принимает: args — разобранные аргументы.
// Делает: собирает значения домена только для переданных полей, требует хотя бы одно.
// Возвращает: domain.Changes либо ArgumentError или доменную ошибку.
func (tool updateExpenseTool) buildChanges(args updateExpenseArguments) (domain.Changes, error) {
	var changes domain.Changes
	if args.Amount != nil {
		amount, err := domain.ParseRubles(string(*args.Amount))
		if err != nil {
			return changes, err
		}
		changes.Amount = &amount
	}
	if args.Category != nil {
		category, err := domain.ParseCategory(*args.Category)
		if err != nil {
			return changes, err
		}
		changes.Category = &category
	}
	spentOn, err := parseDateArgument(args.Date)
	if err != nil {
		return changes, err
	}
	changes.SpentOn = spentOn
	if args.Comment != nil {
		comment, err := domain.NewComment(*args.Comment)
		if err != nil {
			return changes, err
		}
		changes.Comment = &comment
	}

	if changes.Amount == nil && changes.Category == nil && changes.SpentOn == nil && changes.Comment == nil {
		return changes, ports.ArgumentError{Message: "nothing to update: pass at least one of amount, category, date, comment"}
	}
	return changes, nil
}
