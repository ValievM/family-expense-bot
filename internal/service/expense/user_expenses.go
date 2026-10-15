package expense

import (
	"context"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// UserExpenses binds every operation to a single owner.
type UserExpenses struct {
	service *Service
	userID  int64
}

// Create assigns the scoped owner before validating and persisting the expense.
func (user UserExpenses) Create(ctx context.Context, item domain.Expense) (domain.Expense, error) {
	if err := user.validate(); err != nil {
		return domain.Expense{}, err
	}
	item.UserID = user.userID
	return user.service.Create(ctx, item)
}

// Get returns an expense only when it belongs to the scoped owner.
func (user UserExpenses) Get(ctx context.Context, id int64) (domain.Expense, error) {
	if err := user.validate(); err != nil {
		return domain.Expense{}, err
	}
	return user.service.getOwned(ctx, user.userID, id)
}

// List restricts the result to the scoped owner, overriding any supplied owner filter.
func (user UserExpenses) List(ctx context.Context, filter ListFilter) ([]domain.Expense, error) {
	if err := user.validate(); err != nil {
		return nil, err
	}
	filter.UserID = &user.userID
	return user.service.List(ctx, filter)
}

// Update applies a patch only after confirming that the expense belongs to the scoped owner.
func (user UserExpenses) Update(ctx context.Context, id int64, patch Patch) (domain.Expense, error) {
	if err := user.validate(); err != nil {
		return domain.Expense{}, err
	}
	return user.service.updateForUser(ctx, user.userID, id, patch)
}

// Delete removes an expense only when it belongs to the scoped owner.
func (user UserExpenses) Delete(ctx context.Context, id int64) error {
	if err := user.validate(); err != nil {
		return err
	}
	return user.service.deleteForUser(ctx, user.userID, id)
}

func (user UserExpenses) validate() error {
	if user.userID <= 0 {
		return domain.ErrInvalidUser
	}
	return nil
}
