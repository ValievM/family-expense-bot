// Package expense coordinates expense use cases without infrastructure dependencies.
package expense

import (
	"context"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// ListFilter carries family, category, and inclusive calendar-date filters.
type ListFilter struct {
	UserID   *int64
	Category *domain.Category
	From     *time.Time
	To       *time.Time
}

// Repository defines persistence operations required by Service. List sorts by SpentAt descending;
// ListUserIDs returns distinct IDs in ascending order, and missing IDs return ErrNotFound.
type Repository interface {
	Create(ctx context.Context, expense domain.Expense) (domain.Expense, error)
	Get(ctx context.Context, id int64) (domain.Expense, error)
	List(ctx context.Context, filter ListFilter) ([]domain.Expense, error)
	ListUserIDs(ctx context.Context) ([]int64, error)
	Update(ctx context.Context, expense domain.Expense) (domain.Expense, error)
	Delete(ctx context.Context, id int64) error
}
