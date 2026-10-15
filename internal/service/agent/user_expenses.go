// Package agent declares the contracts consumed by the expense agent.
package agent

import (
	"context"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

// UserExpenses exposes only operations scoped to the Telegram user's identity.
type UserExpenses interface {
	Create(ctx context.Context, expense domain.Expense) (domain.Expense, error)
	Get(ctx context.Context, id int64) (domain.Expense, error)
	List(ctx context.Context, filter expense.ListFilter) ([]domain.Expense, error)
	Update(ctx context.Context, id int64, patch expense.Patch) (domain.Expense, error)
	Delete(ctx context.Context, id int64) error
}

var _ UserExpenses = expense.UserExpenses{}
