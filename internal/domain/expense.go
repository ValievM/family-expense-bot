package domain

import "time"

// Expense is a family's recorded spending with its calendar date and category.
type Expense struct {
	ID     int64
	UserID int64
	// Amount is measured in kopecks to keep RUB values exact.
	Amount    int64
	Category  Category
	SpentAt   time.Time
	Comment   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate prevents invalid spending values from crossing the domain boundary.
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
