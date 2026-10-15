// Package expense_test exercises expense use cases through their consumer contract.
package expense_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/expense"
)

func TestCreateForUserUsesTrustedOwner(t *testing.T) {
	repository := newFakeRepository()
	service := expense.NewService(repository)
	input := validExpense(999, 0, 100)

	created, err := service.CreateForUser(context.Background(), 7, input)
	if err != nil {
		t.Fatalf("CreateForUser() error = %v, want nil", err)
	}
	if created.UserID != 7 {
		t.Errorf("created UserID = %d, want trusted owner 7", created.UserID)
	}
	if stored := repository.records[created.ID]; stored.UserID != 7 {
		t.Errorf("stored UserID = %d, want trusted owner 7", stored.UserID)
	}
}

func TestGetForUserHidesForeignExpenses(t *testing.T) {
	stored := validExpense(8, 42, 500)
	repository := newFakeRepository(stored)
	service := expense.NewService(repository)

	if _, err := service.GetForUser(context.Background(), 7, stored.ID); !errors.Is(err, expense.ErrNotFound) {
		t.Fatalf("GetForUser() error = %v, want ErrNotFound", err)
	}
	if got, err := service.GetForUser(context.Background(), 8, stored.ID); err != nil || got.ID != stored.ID {
		t.Errorf("owner GetForUser() = (%+v, %v), want expense %d", got, err, stored.ID)
	}
}

func TestListForUserOverridesPayloadOwner(t *testing.T) {
	repository := newFakeRepository(
		validExpense(7, 1, 100),
		validExpense(8, 2, 200),
	)
	service := expense.NewService(repository)
	requestedOwner := int64(8)

	got, err := service.ListForUser(context.Background(), 7, expense.ListFilter{UserID: &requestedOwner})
	if err != nil {
		t.Fatalf("ListForUser() error = %v, want nil", err)
	}
	if len(got) != 1 || got[0].UserID != 7 {
		t.Errorf("ListForUser() = %+v, want only expenses owned by user 7", got)
	}
}

func TestListAppliesFiltersAndReturnsNewestFirst(t *testing.T) {
	older := validExpense(7, 1, 100)
	older.SpentAt = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	newer := validExpense(7, 2, 200)
	newer.SpentAt = time.Date(2026, time.January, 3, 0, 0, 0, 0, time.UTC)
	otherCategory := validExpense(7, 3, 300)
	otherCategory.Category = domain.CategoryHealth
	otherCategory.SpentAt = time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	repository := newFakeRepository(older, newer, otherCategory)
	service := expense.NewService(repository)
	userID := int64(7)
	category := domain.CategoryFood
	from := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.January, 3, 0, 0, 0, 0, time.UTC)

	got, err := service.List(context.Background(), expense.ListFilter{
		UserID:   &userID,
		Category: &category,
		From:     &from,
		To:       &to,
	})
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(got) != 2 || got[0].ID != newer.ID || got[1].ID != older.ID {
		t.Errorf("List() = %+v, want matching expenses newest first", got)
	}
}

func TestUpdateForUserPreservesOwnerAndCreationTime(t *testing.T) {
	createdAt := time.Date(2025, time.March, 1, 8, 0, 0, 0, time.UTC)
	stored := validExpense(7, 42, 100)
	stored.CreatedAt = createdAt
	repository := newFakeRepository(stored)
	service := expense.NewService(repository)
	update := stored
	update.UserID = 8
	update.CreatedAt = time.Time{}
	update.Amount = 250

	got, err := service.UpdateForUser(context.Background(), 7, update)
	if err != nil {
		t.Fatalf("UpdateForUser() error = %v, want nil", err)
	}
	if got.UserID != 7 || got.CreatedAt != createdAt || got.Amount != 250 {
		t.Errorf("UpdateForUser() = %+v, want owner 7, original creation time, and amount 250", got)
	}
}

func TestUpdateForUserRejectsForeignExpense(t *testing.T) {
	stored := validExpense(8, 42, 100)
	repository := newFakeRepository(stored)
	service := expense.NewService(repository)
	update := stored
	update.Amount = 250

	if _, err := service.UpdateForUser(context.Background(), 7, update); !errors.Is(err, expense.ErrNotFound) {
		t.Fatalf("UpdateForUser() error = %v, want ErrNotFound", err)
	}
	if got := repository.records[stored.ID].Amount; got != 100 {
		t.Errorf("foreign expense amount changed to %d, want 100", got)
	}
}

func TestDeleteForUserRejectsForeignExpense(t *testing.T) {
	stored := validExpense(8, 42, 100)
	repository := newFakeRepository(stored)
	service := expense.NewService(repository)

	if err := service.DeleteForUser(context.Background(), 7, stored.ID); !errors.Is(err, expense.ErrNotFound) {
		t.Fatalf("DeleteForUser() error = %v, want ErrNotFound", err)
	}
	if _, exists := repository.records[stored.ID]; !exists {
		t.Fatal("foreign expense was deleted")
	}
	if err := service.DeleteForUser(context.Background(), 8, stored.ID); err != nil {
		t.Fatalf("owner DeleteForUser() error = %v, want nil", err)
	}
	if _, exists := repository.records[stored.ID]; exists {
		t.Fatal("owned expense remains after delete")
	}
}

func TestCreateRejectsInvalidExpenseBeforePersistence(t *testing.T) {
	repository := newFakeRepository()
	service := expense.NewService(repository)
	invalid := validExpense(7, 0, 0)

	if _, err := service.Create(context.Background(), invalid); !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("Create() error = %v, want ErrInvalidAmount", err)
	}
	if len(repository.records) != 0 {
		t.Errorf("Create() persisted %d invalid expenses", len(repository.records))
	}
}

func TestListForUserRejectsInvalidOwner(t *testing.T) {
	service := expense.NewService(newFakeRepository())
	if _, err := service.ListForUser(context.Background(), 0, expense.ListFilter{}); !errors.Is(err, domain.ErrInvalidUser) {
		t.Errorf("ListForUser() error = %v, want ErrInvalidUser", err)
	}
}

type fakeRepository struct {
	records map[int64]domain.Expense
	nextID  int64
}

func newFakeRepository(expenses ...domain.Expense) *fakeRepository {
	repository := &fakeRepository{records: make(map[int64]domain.Expense)}
	for _, item := range expenses {
		if item.ID == 0 {
			repository.nextID++
			item.ID = repository.nextID
		} else if item.ID > repository.nextID {
			repository.nextID = item.ID
		}
		repository.records[item.ID] = item
	}
	return repository
}

func (repository *fakeRepository) Create(_ context.Context, item domain.Expense) (domain.Expense, error) {
	repository.nextID++
	item.ID = repository.nextID
	repository.records[item.ID] = item
	return item, nil
}

func (repository *fakeRepository) Get(_ context.Context, id int64) (domain.Expense, error) {
	item, exists := repository.records[id]
	if !exists {
		return domain.Expense{}, expense.ErrNotFound
	}
	return item, nil
}

func (repository *fakeRepository) List(_ context.Context, filter expense.ListFilter) ([]domain.Expense, error) {
	items := make([]domain.Expense, 0, len(repository.records))
	for _, item := range repository.records {
		if filter.UserID != nil && item.UserID != *filter.UserID {
			continue
		}
		if filter.Category != nil && item.Category != *filter.Category {
			continue
		}
		if filter.From != nil && item.SpentAt.Before(*filter.From) {
			continue
		}
		if filter.To != nil && item.SpentAt.After(*filter.To) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].SpentAt.After(items[j].SpentAt)
	})
	return items, nil
}

func (repository *fakeRepository) Update(_ context.Context, item domain.Expense) (domain.Expense, error) {
	if _, exists := repository.records[item.ID]; !exists {
		return domain.Expense{}, expense.ErrNotFound
	}
	repository.records[item.ID] = item
	return item, nil
}

func (repository *fakeRepository) Delete(_ context.Context, id int64) error {
	if _, exists := repository.records[id]; !exists {
		return expense.ErrNotFound
	}
	delete(repository.records, id)
	return nil
}

func validExpense(userID, id, amount int64) domain.Expense {
	return domain.Expense{
		ID:       id,
		UserID:   userID,
		Amount:   amount,
		Category: domain.CategoryFood,
		SpentAt:  time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
	}
}
