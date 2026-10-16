package tools

import (
	"context"
	"encoding/json"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

// addExpenseTool создаёт трату владельца сообщения.
type addExpenseTool struct{}

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

// Принимает: ctx — контекст запроса; expenses — траты владельца сообщения; arguments — JSON от модели.
// Делает: разбирает аргументы, собирает значения домена (сумма, категория, день, комментарий) и сохраняет трату через сервис.
// Возвращает: созданную трату в виде expenseView либо ошибку аргументов или валидации.
func (tool addExpenseTool) Execute(ctx context.Context, expenses ports.UserExpenses, arguments json.RawMessage) (any, error) {
	var args addExpenseArguments
	if err := decodeArguments(arguments, &args); err != nil {
		return nil, err
	}
	amount, err := domain.MoneyFromRubles(args.Amount)
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
