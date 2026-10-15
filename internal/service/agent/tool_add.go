package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// addExpenseTool создаёт трату владельца сообщения.
type addExpenseTool struct {
	location *time.Location
}

// addExpenseArguments — аргументы add_expense; поля user_id намеренно нет.
type addExpenseArguments struct {
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	Date     *string `json:"date"`
	Comment  string  `json:"comment"`
}

// Принимает: ничего.
// Делает: описывает инструмент add_expense и схему его аргументов.
// Возвращает: ToolSpec с обязательными amount и category.
func (tool addExpenseTool) Spec() ToolSpec {
	return ToolSpec{
		Name:        "add_expense",
		Description: "Добавить трату пользователя. Дата необязательна: по умолчанию сегодня.",
		Parameters: objectSchema(map[string]any{
			"amount":   map[string]any{"type": "number", "description": "Сумма в рублях, например 350 или 99.90."},
			"category": categoryProperty(),
			"date":     dateProperty("Дата траты."),
			"comment":  map[string]any{"type": "string", "description": "Короткое пояснение, например «кофе»."},
		}, "amount", "category"),
	}
}

// Принимает: ctx — контекст запроса; expenses — траты владельца сообщения; arguments — JSON от модели.
// Делает: разбирает аргументы, переводит рубли в копейки и сохраняет трату через сервис.
// Возвращает: созданную трату в виде expenseView либо ошибку аргументов или валидации.
func (tool addExpenseTool) Execute(ctx context.Context, expenses UserExpenses, arguments json.RawMessage) (any, error) {
	var args addExpenseArguments
	if err := decodeArguments(arguments, &args); err != nil {
		return nil, err
	}
	kopecks, err := rublesToKopecks(args.Amount)
	if err != nil {
		return nil, err
	}
	spentAt, err := parseDateArgument(args.Date, tool.location)
	if err != nil {
		return nil, err
	}

	item := domain.Expense{Amount: kopecks, Category: domain.Category(args.Category), Comment: args.Comment}
	if spentAt != nil {
		item.SpentAt = *spentAt
	}
	created, err := expenses.Create(ctx, item)
	if err != nil {
		return nil, err
	}
	return newExpenseView(created, tool.location), nil
}
