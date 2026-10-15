package domain

import (
	"errors"
	"testing"
	"time"
)

// Принимает: t — тестовый контекст.
// Делает: проверяет Expense.Validate на корректном расходе и каждом виде нарушения.
// Возвращает: ничего.
func TestExpenseValidate(t *testing.T) {
	validExpense := Expense{
		UserID:    1,
		Amount:    1,
		Category:  CategoryFood,
		SpentAt:   time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
	}
	tests := []struct {
		name    string
		mutate  func(*Expense)
		wantErr error
	}{
		{name: "valid expense"},
		{
			name: "zero amount",
			mutate: func(expense *Expense) {
				expense.Amount = 0
			},
			wantErr: ErrInvalidAmount,
		},
		{
			name: "negative amount",
			mutate: func(expense *Expense) {
				expense.Amount = -1
			},
			wantErr: ErrInvalidAmount,
		},
		{
			name: "zero user ID",
			mutate: func(expense *Expense) {
				expense.UserID = 0
			},
			wantErr: ErrInvalidUser,
		},
		{
			name: "negative user ID",
			mutate: func(expense *Expense) {
				expense.UserID = -1
			},
			wantErr: ErrInvalidUser,
		},
		{
			name: "unknown category",
			mutate: func(expense *Expense) {
				expense.Category = "travel"
			},
			wantErr: ErrUnknownCategory,
		},
		{
			name: "missing date",
			mutate: func(expense *Expense) {
				expense.SpentAt = time.Time{}
			},
			wantErr: ErrInvalidDate,
		},
		{
			name: "date with time of day",
			mutate: func(expense *Expense) {
				expense.SpentAt = expense.SpentAt.Add(time.Hour)
			},
			wantErr: ErrInvalidDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expense := validExpense
			if tt.mutate != nil {
				tt.mutate(&expense)
			}

			err := expense.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
