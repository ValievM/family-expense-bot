package expense

import (
	"context"
	"fmt"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// Service coordinates expense validation, ownership, and persistence.
type Service struct {
	repository Repository
	clock      Clock
	location   *time.Location
}

// NewService wires persistence, time, and calendar behavior; nil clock and location use system time and UTC.
func NewService(repository Repository, clock Clock, location *time.Location) *Service {
	if clock == nil {
		clock = systemClock{}
	}
	if location == nil {
		location = time.UTC
	}
	return &Service{repository: repository, clock: clock, location: location}
}

// ForUser returns a scope whose methods cannot select a different owner.
func (service *Service) ForUser(userID int64) UserExpenses {
	return UserExpenses{service: service, userID: userID}
}

// Create validates and persists an expense for an unrestricted caller; a zero SpentAt means today.
func (service *Service) Create(ctx context.Context, item domain.Expense) (domain.Expense, error) {
	now := service.clock.Now().UTC()
	if item.SpentAt.IsZero() {
		// An omitted date means "today", which every caller (REST and agent) shares.
		item.SpentAt = now
	}
	item.SpentAt = service.normalizeDate(item.SpentAt)
	if err := item.Validate(); err != nil {
		return domain.Expense{}, err
	}

	item.CreatedAt = now
	item.UpdatedAt = now
	created, err := service.repository.Create(ctx, item)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("create expense: %w", err)
	}
	return created, nil
}

// Get retrieves an expense by its ID for unrestricted callers such as the family panel.
func (service *Service) Get(ctx context.Context, id int64) (domain.Expense, error) {
	item, err := service.repository.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("get expense: %w", err)
	}
	return item, nil
}

// List validates, normalizes, and applies the shared list filters.
func (service *Service) List(ctx context.Context, filter ListFilter) ([]domain.Expense, error) {
	filter, err := service.normalizeListFilter(filter)
	if err != nil {
		return nil, err
	}

	items, err := service.repository.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	return items, nil
}

// ListUserIDs returns the distinct owners that currently have expenses.
func (service *Service) ListUserIDs(ctx context.Context) ([]int64, error) {
	userIDs, err := service.repository.ListUserIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list expense owners: %w", err)
	}
	return userIDs, nil
}

// Update merges a partial update with the stored expense and validates the result once.
func (service *Service) Update(ctx context.Context, id int64, patch Patch) (domain.Expense, error) {
	current, err := service.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, err
	}
	return service.updateCurrent(ctx, current, patch)
}

// Delete removes an expense for an unrestricted caller such as the family panel.
func (service *Service) Delete(ctx context.Context, id int64) error {
	if err := service.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	return nil
}

func (service *Service) getOwned(ctx context.Context, userID, id int64) (domain.Expense, error) {
	if userID <= 0 {
		return domain.Expense{}, domain.ErrInvalidUser
	}
	item, err := service.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, err
	}
	if item.UserID != userID {
		return domain.Expense{}, ErrNotFound
	}
	return item, nil
}

func (service *Service) updateForUser(ctx context.Context, userID, id int64, patch Patch) (domain.Expense, error) {
	current, err := service.getOwned(ctx, userID, id)
	if err != nil {
		return domain.Expense{}, err
	}
	return service.updateCurrent(ctx, current, patch)
}

func (service *Service) updateCurrent(ctx context.Context, current domain.Expense, patch Patch) (domain.Expense, error) {
	if patch.Amount != nil {
		current.Amount = *patch.Amount
	}
	if patch.Category != nil {
		current.Category = *patch.Category
	}
	if patch.SpentAt != nil {
		current.SpentAt = *patch.SpentAt
	}
	if patch.Comment != nil {
		current.Comment = *patch.Comment
	}
	current.SpentAt = service.normalizeDate(current.SpentAt)
	if err := current.Validate(); err != nil {
		return domain.Expense{}, err
	}

	current.UpdatedAt = service.clock.Now().UTC()
	updated, err := service.repository.Update(ctx, current)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("update expense: %w", err)
	}
	return updated, nil
}

func (service *Service) deleteForUser(ctx context.Context, userID, id int64) error {
	if _, err := service.getOwned(ctx, userID, id); err != nil {
		return err
	}
	if err := service.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	return nil
}

func (service *Service) normalizeListFilter(filter ListFilter) (ListFilter, error) {
	if filter.UserID != nil && *filter.UserID <= 0 {
		return ListFilter{}, domain.ErrInvalidUser
	}
	if filter.Category != nil && !filter.Category.IsValid() {
		return ListFilter{}, domain.ErrUnknownCategory
	}
	if filter.From != nil {
		if filter.From.IsZero() {
			return ListFilter{}, domain.ErrInvalidDate
		}
		from := domain.DateOnly(*filter.From, service.location)
		filter.From = &from
	}
	if filter.To != nil {
		if filter.To.IsZero() {
			return ListFilter{}, domain.ErrInvalidDate
		}
		to := domain.DateOnly(*filter.To, service.location)
		filter.To = &to
	}
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return ListFilter{}, ErrInvalidPeriod
	}
	return filter, nil
}

func (service *Service) normalizeDate(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return domain.DateOnly(value, service.location)
}
