package tools

import "github.com/ValievM/family-expense-bot/internal/domain"

// Сумма уже в рублях строкой: модель не должна считать копейки.
type expenseView struct {
	ID            int64  `json:"id"`
	AmountRub     string `json:"amount_rub"`
	Category      string `json:"category"`
	CategoryLabel string `json:"category_label"`
	Date          string `json:"date"`
	Comment       string `json:"comment,omitempty"`
}

func newExpenseView(item domain.Expense) expenseView {
	return expenseView{
		ID:            int64(item.ID()),
		AmountRub:     item.Amount().String(),
		Category:      string(item.Category()),
		CategoryLabel: item.Category().Label(),
		Date:          item.SpentOn().String(),
		Comment:       item.Comment().String(),
	}
}
