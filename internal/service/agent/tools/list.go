package tools

import (
	"context"
	"encoding/json"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

// Ограничивает размер ответа модели; итог считается по всем найденным тратам.
const maxListedExpenses = 50

type listExpensesTool struct{}

// Фильтра по владельцу нет намеренно.
type listExpensesArguments struct {
	Category *string `json:"category"`
	From     *string `json:"from"`
	To       *string `json:"to"`
}

// total_rub и latest_id считаются по всем найденным тратам, а не по обрезанному списку.
type listExpensesResult struct {
	Count     int           `json:"count"`
	TotalRub  string        `json:"total_rub"`
	LatestID  int64         `json:"latest_id"`
	Truncated bool          `json:"truncated"`
	Expenses  []expenseView `json:"expenses"`
}

func (tool listExpensesTool) Spec() ports.ToolSpec {
	category := categoryProperty()
	category["description"] = "Фильтр по категории. " + category["description"].(string)
	return ports.ToolSpec{
		Name:        "list_expenses",
		Description: "Показать траты пользователя (новые сверху) с итогом total_rub. Без фильтров вернёт все.",
		Parameters: objectSchema(map[string]any{
			"category": category,
			"from":     dateProperty("Начало периода включительно."),
			"to":       dateProperty("Конец периода включительно."),
		}),
	}
}

// Принимает: ctx; expenses; arguments — JSON от модели.
// Делает: применяет фильтры и отдаёт сводку.
// Возвращает: listExpensesResult либо ошибку аргументов или периода.
func (tool listExpensesTool) Execute(ctx context.Context, expenses ports.UserExpenses, arguments json.RawMessage) (any, error) {
	var args listExpensesArguments
	if err := decodeArguments(arguments, &args); err != nil {
		return nil, err
	}
	from, err := parseDateArgument(args.From)
	if err != nil {
		return nil, err
	}
	to, err := parseDateArgument(args.To)
	if err != nil {
		return nil, err
	}

	filter := ports.ListFilter{From: from, To: to}
	if args.Category != nil && *args.Category != "" {
		category, err := domain.ParseCategory(*args.Category)
		if err != nil {
			return nil, err
		}
		filter.Category = &category
	}
	items, err := expenses.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return tool.summarize(items), nil
}

// Принимает: items — все найденные траты.
// Делает: считает итог и максимальный id по всем, показывает не больше maxListedExpenses.
// Возвращает: listExpensesResult.
func (tool listExpensesTool) summarize(items []domain.Expense) listExpensesResult {
	result := listExpensesResult{Count: len(items), Expenses: make([]expenseView, 0, len(items))}
	var total domain.Money
	for _, item := range items {
		total = total.Add(item.Amount())
		if id := int64(item.ID()); id > result.LatestID {
			result.LatestID = id
		}
		if len(result.Expenses) < maxListedExpenses {
			result.Expenses = append(result.Expenses, newExpenseView(item))
		}
	}
	result.TotalRub = total.String()
	result.Truncated = len(items) > maxListedExpenses
	return result
}
