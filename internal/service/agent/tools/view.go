package tools

import (
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// expenseView — представление траты для модели: сумма уже в рублях строкой, чтобы не считать копейки.
type expenseView struct {
	ID            int64  `json:"id"`
	AmountRub     string `json:"amount_rub"`
	Category      string `json:"category"`
	CategoryLabel string `json:"category_label"`
	Date          string `json:"date"`
	Comment       string `json:"comment,omitempty"`
}

// Принимает: item — трата из домена; location — часовой пояс для даты.
// Делает: преобразует трату в безопасное для модели представление без user_id.
// Возвращает: заполненный expenseView.
func newExpenseView(item domain.Expense, location *time.Location) expenseView {
	return expenseView{
		ID:            item.ID,
		AmountRub:     formatKopecks(item.Amount),
		Category:      string(item.Category),
		CategoryLabel: item.Category.Label(),
		Date:          domain.FormatDate(item.SpentAt, location),
		Comment:       item.Comment,
	}
}
