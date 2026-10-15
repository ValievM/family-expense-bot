package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

// updateExpenseTool частично меняет трату владельца сообщения.
type updateExpenseTool struct {
	location *time.Location
}

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
func (tool updateExpenseTool) Spec() ToolSpec {
	return ToolSpec{
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
func (tool updateExpenseTool) Execute(ctx context.Context, expenses UserExpenses, arguments json.RawMessage) (any, error) {
	var args updateExpenseArguments
	if err := decodeArguments(arguments, &args); err != nil {
		return nil, err
	}
	if args.ID <= 0 {
		return nil, ArgumentError{Message: "id must be a positive integer"}
	}
	patch, err := tool.buildPatch(args)
	if err != nil {
		return nil, err
	}

	updated, err := expenses.Update(ctx, args.ID, patch)
	if err != nil {
		return nil, err
	}
	return newExpenseView(updated, tool.location), nil
}

// Принимает: args — разобранные аргументы update_expense.
// Делает: переводит рубли в копейки, разбирает дату и проверяет, что есть хотя бы одно изменение.
// Возвращает: expense.Patch либо ArgumentError или domain.ErrInvalidAmount.
func (tool updateExpenseTool) buildPatch(args updateExpenseArguments) (expense.Patch, error) {
	var patch expense.Patch
	if args.Amount != nil {
		kopecks, err := rublesToKopecks(*args.Amount)
		if err != nil {
			return patch, err
		}
		patch.Amount = &kopecks
	}
	if args.Category != nil {
		category := domain.Category(*args.Category)
		patch.Category = &category
	}
	spentAt, err := parseDateArgument(args.Date, tool.location)
	if err != nil {
		return patch, err
	}
	patch.SpentAt = spentAt
	patch.Comment = args.Comment

	if patch.Amount == nil && patch.Category == nil && patch.SpentAt == nil && patch.Comment == nil {
		return patch, ArgumentError{Message: "nothing to update: pass at least one of amount, category, date, comment"}
	}
	return patch, nil
}
