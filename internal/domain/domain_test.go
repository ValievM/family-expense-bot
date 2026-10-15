// Package domain_test verifies domain behavior through its public API.
package domain_test

import (
	"errors"
	"testing"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

func TestExpenseValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*domain.Expense)
		wantErr error
	}{
		{name: "positive amount and valid date"},
		{
			name: "zero amount",
			mutate: func(expense *domain.Expense) {
				expense.Amount = 0
			},
			wantErr: domain.ErrInvalidAmount,
		},
		{
			name: "negative amount",
			mutate: func(expense *domain.Expense) {
				expense.Amount = -1
			},
			wantErr: domain.ErrInvalidAmount,
		},
		{
			name: "unknown category",
			mutate: func(expense *domain.Expense) {
				expense.Category = "travel"
			},
			wantErr: domain.ErrUnknownCategory,
		},
		{
			name: "zero date",
			mutate: func(expense *domain.Expense) {
				expense.SpentAt = domain.Date{}
			},
			wantErr: domain.ErrInvalidDate,
		},
		{
			name: "impossible calendar date",
			mutate: func(expense *domain.Expense) {
				expense.SpentAt = domain.Date{Year: 2025, Month: 2, Day: 29}
			},
			wantErr: domain.ErrInvalidDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expense := validExpense()
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

func TestCategoryLabels(t *testing.T) {
	tests := []struct {
		category domain.Category
		want     string
	}{
		{category: domain.CategoryFood, want: "еда"},
		{category: domain.CategoryTransport, want: "транспорт"},
		{category: domain.CategoryHousing, want: "жильё"},
		{category: domain.CategoryEntertainment, want: "развлечения"},
		{category: domain.CategoryHealth, want: "здоровье"},
		{category: domain.CategoryShopping, want: "покупки"},
		{category: domain.CategoryOther, want: "другое"},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			if !tt.category.IsValid() {
				t.Fatalf("%q is not a valid category", tt.category)
			}
			if got := tt.category.Label(); got != tt.want {
				t.Errorf("Label() = %q, want %q", got, tt.want)
			}
		})
	}

	if domain.Category("travel").IsValid() {
		t.Error("unknown category is valid")
	}
}

func TestDateIsValid(t *testing.T) {
	tests := []struct {
		name string
		date domain.Date
		want bool
	}{
		{name: "ordinary date", date: domain.Date{Year: 2026, Month: 10, Day: 16}, want: true},
		{name: "leap day", date: domain.Date{Year: 2024, Month: 2, Day: 29}, want: true},
		{name: "400-year leap day", date: domain.Date{Year: 2000, Month: 2, Day: 29}, want: true},
		{name: "non-leap century", date: domain.Date{Year: 1900, Month: 2, Day: 29}},
		{name: "non-leap year", date: domain.Date{Year: 2025, Month: 2, Day: 29}},
		{name: "year zero", date: domain.Date{Year: 0, Month: 1, Day: 1}},
		{name: "month zero", date: domain.Date{Year: 2026, Month: 0, Day: 1}},
		{name: "month above range", date: domain.Date{Year: 2026, Month: 13, Day: 1}},
		{name: "day zero", date: domain.Date{Year: 2026, Month: 1, Day: 0}},
		{name: "day above month range", date: domain.Date{Year: 2026, Month: 4, Day: 31}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.date.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %t, want %t", got, tt.want)
			}
		})
	}
}

func validExpense() domain.Expense {
	return domain.Expense{
		Amount:   1,
		Category: domain.CategoryFood,
		SpentAt:  domain.Date{Year: 2024, Month: 2, Day: 29},
	}
}
