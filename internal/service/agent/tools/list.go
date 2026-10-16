package tools

import (
	"context"
	"encoding/json"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

// maxListedExpenses ограничивает размер ответа модели; итог считается по всем найденным тратам.
const maxListedExpenses = 50

// listExpensesTool показывает траты владельца сообщения с фильтрами.
type listExpensesTool struct{}

// listExpensesArguments — аргументы list_expenses; фильтра по владельцу нет.
type listExpensesArguments struct {
	Category *string `json:"category"`
	From     *string `json:"from"`
	To       *string `json:"to"`
}

// listExpensesResult — ответ list_expenses: траты, посчитанный итог и id последней добавленной траты.
type listExpensesResult struct {
	Count     int           `json:"count"`
	TotalRub  string        `json:"total_rub"`
	LatestID  int64         `json:"latest_id"`
	Truncated bool          `json:"truncated"`
	Expenses  []expenseView `json:"expenses"`
}

// Принимает: ничего.
// Делает: описывает инструмент list_expenses и необязательные фильтры.
// Возвращает: ToolSpec без обязательных аргументов.
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

// Принимает: ctx — контекст запроса; expenses — траты владельца сообщения; arguments — JSON от модели.
// Делает: применяет фильтры, считает итог и id последней траты по всем найденным записям и обрезает список до лимита.
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

// Принимает: items — все найденные траты владельца.
// Делает: считает итог и максимальный id по всем записям, показывает не больше maxListedExpenses.
// Возвращает: готовый listExpensesResult.
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
