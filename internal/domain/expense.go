package domain

import "time"

// Expense — записанный семейный расход с календарной датой и категорией.
type Expense struct {
	ID     int64
	UserID int64
	// Amount — сумма в копейках, чтобы значения в рублях были точными.
	Amount    int64
	Category  Category
	SpentAt   time.Time
	Comment   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Принимает: ничего (получатель — расход).
// Делает: проверяет сумму, владельца, категорию и то, что SpentAt — дата без времени суток.
// Возвращает: nil для корректного расхода; ErrInvalidAmount, ErrInvalidUser, ErrUnknownCategory или ErrInvalidDate при первом нарушении.
func (expense Expense) Validate() error {
	if expense.Amount <= 0 {
		return ErrInvalidAmount
	}
	if expense.UserID <= 0 {
		return ErrInvalidUser
	}
	if !expense.Category.IsValid() {
		return ErrUnknownCategory
	}
	if !isDateOnly(expense.SpentAt) {
		return ErrInvalidDate
	}
	return nil
}
