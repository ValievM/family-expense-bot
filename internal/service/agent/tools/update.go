package tools

import (
	"context"
	"encoding/json"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/agent"
)

// updateExpenseTool частично меняет трату владельца сообщения.
type updateExpenseTool struct{}

// updateExpenseArguments — аргументы update_expense; владельца задаёт только scope.
type updateExpenseArguments struct {
	ID       int64    `json:"id"`
	Amount   *float64 `json:"amount"`
	Category *string  `json:"category"`
	Date     *string  `json:"date"`
	Comment  *string  `json:"comment"`
}

// Принимает: ничего.
// Делает: описывает инструмент update_expense: id и любые изменяемые поля.
// Возвращает: ToolSpec с обязательным id.
func (tool updateExpenseTool) Spec() agent.ToolSpec {
	return agent.ToolSpec{
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

// Принимает: ctx — контекст запроса; expenses — траты владельца сообщения; arguments — JSON от модели.
// Делает: собирает частичное изменение и применяет его, только если трата принадлежит владельцу.
// Возвращает: обновлённую трату в виде expenseView либо ошибку аргументов, валидации или «не найдено».
func (tool updateExpenseTool) Execute(ctx context.Context, expenses agent.UserExpenses, arguments json.RawMessage) (any, error) {
	var args updateExpenseArguments
	if err := decodeArguments(arguments, &args); err != nil {
		return nil, err
	}
	if args.ID <= 0 {
		return nil, agent.ArgumentError{Message: "id must be a positive integer"}
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

// Принимает: args — разобранные аргументы update_expense.
// Делает: собирает значения домена только для переданных полей и проверяет, что есть хотя бы одно изменение.
// Возвращает: domain.Changes либо ArgumentError или доменную ошибку значения.
func (tool updateExpenseTool) buildChanges(args updateExpenseArguments) (domain.Changes, error) {
	var changes domain.Changes
	if args.Amount != nil {
		amount, err := domain.MoneyFromRubles(*args.Amount)
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
		return changes, agent.ArgumentError{Message: "nothing to update: pass at least one of amount, category, date, comment"}
	}
	return changes, nil
}
