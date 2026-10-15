package expense

import (
	"context"
	"fmt"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// Service coordinates expense validation, ownership, and persistence.
type Service struct {
	repository Repository
}

// NewService constructs the application service with its persistence port.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// Create validates and persists an expense whose owner was supplied by a trusted caller.
func (service *Service) Create(ctx context.Context, expense domain.Expense) (domain.Expense, error) {
	if err := expense.Validate(); err != nil {
		return domain.Expense{}, err
	}

	created, err := service.repository.Create(ctx, expense)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("create expense: %w", err)
	}
	return created, nil
}

// CreateForUser binds ownership to the trusted user ID rather than the expense payload.
func (service *Service) CreateForUser(ctx context.Context, userID int64, expense domain.Expense) (domain.Expense, error) {
	expense.UserID = userID
	return service.Create(ctx, expense)
}

// Get retrieves an expense by its ID for unrestricted callers such as the family panel.
func (service *Service) Get(ctx context.Context, id int64) (domain.Expense, error) {
	expense, err := service.repository.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("get expense: %w", err)
	}
	return expense, nil
}

// GetForUser returns an expense only when it belongs to userID.
func (service *Service) GetForUser(ctx context.Context, userID, id int64) (domain.Expense, error) {
	if userID <= 0 {
		return domain.Expense{}, domain.ErrInvalidUser
	}

	expense, err := service.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, err
	}
	if expense.UserID != userID {
		return domain.Expense{}, ErrNotFound
	}
	return expense, nil
}

// List validates filters and returns matching expenses.
func (service *Service) List(ctx context.Context, filter ListFilter) ([]domain.Expense, error) {
	if err := validateListFilter(filter); err != nil {
		return nil, err
	}

	expenses, err := service.repository.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	return expenses, nil
}

// ListForUser restricts the result to userID, overriding any payload owner filter.
func (service *Service) ListForUser(ctx context.Context, userID int64, filter ListFilter) ([]domain.Expense, error) {
	if userID <= 0 {
		return nil, domain.ErrInvalidUser
	}
	filter.UserID = &userID
	return service.List(ctx, filter)
}

// Update validates and persists an expense for an unrestricted caller.
func (service *Service) Update(ctx context.Context, expense domain.Expense) (domain.Expense, error) {
	if err := expense.Validate(); err != nil {
		return domain.Expense{}, err
	}

	updated, err := service.repository.Update(ctx, expense)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("update expense: %w", err)
	}
	return updated, nil
}

// UpdateForUser updates an owned expense while preserving its owner and creation time.
func (service *Service) UpdateForUser(ctx context.Context, userID int64, updated domain.Expense) (domain.Expense, error) {
	current, err := service.GetForUser(ctx, userID, updated.ID)
	if err != nil {
		return domain.Expense{}, err
	}

	updated.UserID = current.UserID
	updated.CreatedAt = current.CreatedAt
	return service.Update(ctx, updated)
}

// Delete removes an expense for an unrestricted caller.
func (service *Service) Delete(ctx context.Context, id int64) error {
	if err := service.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	return nil
}

// DeleteForUser removes an expense only when it belongs to userID.
func (service *Service) DeleteForUser(ctx context.Context, userID, id int64) error {
	if _, err := service.GetForUser(ctx, userID, id); err != nil {
		return err
	}
	return service.Delete(ctx, id)
}

func validateListFilter(filter ListFilter) error {
	if filter.UserID != nil && *filter.UserID <= 0 {
		return domain.ErrInvalidUser
	}
	if filter.Category != nil && !filter.Category.IsValid() {
		return domain.ErrUnknownCategory
	}
	return nil
}
