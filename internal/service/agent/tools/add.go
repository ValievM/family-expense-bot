package tools

import (
	"context"
	"encoding/json"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

type addExpenseTool struct{}

// user_id намеренно нет: владелец берётся из области, не от модели.
type addExpenseArguments struct {
	Amount   amountText `json:"amount"`
	Category string     `json:"category"`
	Date     *string    `json:"date"`
	Comment  string     `json:"comment"`
}

func (tool addExpenseTool) Spec() ports.ToolSpec {
	return ports.ToolSpec{
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

// Принимает: ctx; expenses — траты владельца; arguments — JSON от модели.
// Делает: разбирает аргументы, собирает значения домена и создаёт трату.
// Возвращает: expenseView созданной траты либо ошибку аргументов или валидации.
func (tool addExpenseTool) Execute(ctx context.Context, expenses ports.UserExpenses, arguments json.RawMessage) (any, error) {
	var args addExpenseArguments
	if err := decodeArguments(arguments, &args); err != nil {
		return nil, err
	}
	amount, err := domain.ParseRubles(string(args.Amount))
	if err != nil {
		return nil, err
	}
	category, err := domain.ParseCategory(args.Category)
	if err != nil {
		return nil, err
	}
	spentOn, err := parseDateArgument(args.Date)
	if err != nil {
		return nil, err
	}
	comment, err := domain.NewComment(args.Comment)
	if err != nil {
		return nil, err
	}

	created, err := expenses.Create(ctx, ports.UserCreateInput{Amount: amount, Category: category, SpentOn: spentOn, Comment: comment})
	if err != nil {
		return nil, err
	}
	return newExpenseView(created), nil
}
