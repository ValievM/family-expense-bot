// Package expense_test exercises expense use cases through their consumer contract.
package expense_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

func TestCreateStampsTimestampsFromClock(t *testing.T) {
	now := time.Date(2026, time.January, 2, 15, 4, 5, 6, time.FixedZone("UTC+03", 3*60*60))
	repository := newFakeRepository()
	service := expense.NewService(repository, &fakeClock{now: now}, time.UTC)

	created, err := service.Create(context.Background(), validExpense(7, 0, 100))
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	if !created.CreatedAt.Equal(now.UTC()) || !created.UpdatedAt.Equal(now.UTC()) {
		t.Errorf("created timestamps = (%v, %v), want both %v", created.CreatedAt, created.UpdatedAt, now.UTC())
	}
}

func TestUpdateMergesPatchAndStampsUpdateTime(t *testing.T) {
	createdAt := time.Date(2025, time.March, 1, 8, 0, 0, 0, time.UTC)
	stored := validExpense(7, 42, 100)
	stored.CreatedAt = createdAt
	stored.UpdatedAt = createdAt
	repository := newFakeRepository(stored)
	now := time.Date(2026, time.January, 2, 15, 4, 5, 6, time.UTC)
	service := expense.NewService(repository, &fakeClock{now: now}, time.UTC)
	amount := int64(250)
	comment := "new comment"

	updated, err := service.Update(context.Background(), stored.ID, expense.Patch{
		Amount:  &amount,
		Comment: &comment,
	})
	if err != nil {
		t.Fatalf("Update() error = %v, want nil", err)
	}
	if updated.Amount != amount || updated.Comment != comment || updated.Category != stored.Category {
		t.Errorf("Update() merged fields incorrectly: %+v", updated)
	}
	if !updated.CreatedAt.Equal(createdAt) || !updated.UpdatedAt.Equal(now) {
		t.Errorf("Update() timestamps = (%v, %v), want (%v, %v)", updated.CreatedAt, updated.UpdatedAt, createdAt, now)
	}
}

func TestUpdateRejectsInvalidMergedExpenseWithoutPersistence(t *testing.T) {
	stored := validExpense(7, 42, 100)
	repository := newFakeRepository(stored)
	service := expense.NewService(repository, &fakeClock{now: time.Now()}, time.UTC)
	invalidAmount := int64(0)

	if _, err := service.Update(context.Background(), stored.ID, expense.Patch{Amount: &invalidAmount}); !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("Update() error = %v, want ErrInvalidAmount", err)
	}
	if got := repository.records[stored.ID].Amount; got != stored.Amount {
		t.Errorf("invalid patch changed amount to %d, want %d", got, stored.Amount)
	}
}

func TestUserExpensesCannotReadOrChangeForeignExpenses(t *testing.T) {
	stored := validExpense(8, 42, 100)
	repository := newFakeRepository(stored)
	service := expense.NewService(repository, &fakeClock{now: time.Now()}, time.UTC)
	userExpenses := service.ForUser(7)
	amount := int64(250)

	if _, err := userExpenses.Get(context.Background(), stored.ID); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
	if _, err := userExpenses.Update(context.Background(), stored.ID, expense.Patch{Amount: &amount}); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Update() error = %v, want ErrNotFound", err)
	}
	if err := userExpenses.Delete(context.Background(), stored.ID); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
	if got := repository.records[stored.ID].Amount; got != stored.Amount {
		t.Errorf("foreign expense amount changed to %d, want %d", got, stored.Amount)
	}
	if _, exists := repository.records[stored.ID]; !exists {
		t.Fatal("foreign expense was deleted")
	}
}

func TestUserExpensesCreateAndListBindOwner(t *testing.T) {
	repository := newFakeRepository(validExpense(8, 1, 200))
	service := expense.NewService(repository, &fakeClock{now: time.Now()}, time.UTC)
	userExpenses := service.ForUser(7)
	created, err := userExpenses.Create(context.Background(), validExpense(999, 0, 100))
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	if created.UserID != 7 {
		t.Errorf("Create() UserID = %d, want scoped owner 7", created.UserID)
	}

	requestedOwner := int64(8)
	listed, err := userExpenses.List(context.Background(), expense.ListFilter{UserID: &requestedOwner})
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if len(listed) != 1 || listed[0].UserID != 7 {
		t.Errorf("List() = %+v, want only expenses owned by user 7", listed)
	}
}

func TestUserExpensesUpdatePreservesOwnerAndCreationTime(t *testing.T) {
	createdAt := time.Date(2025, time.March, 1, 8, 0, 0, 0, time.UTC)
	stored := validExpense(7, 42, 100)
	stored.CreatedAt = createdAt
	repository := newFakeRepository(stored)
	now := time.Date(2026, time.January, 2, 15, 0, 0, 0, time.UTC)
	userExpenses := expense.NewService(repository, &fakeClock{now: now}, time.UTC).ForUser(7)
	amount := int64(250)

	updated, err := userExpenses.Update(context.Background(), stored.ID, expense.Patch{Amount: &amount})
	if err != nil {
		t.Fatalf("Update() error = %v, want nil", err)
	}
	if updated.UserID != 7 || !updated.CreatedAt.Equal(createdAt) || updated.Amount != amount {
		t.Errorf("Update() = %+v, want owner 7, original creation time, and amount %d", updated, amount)
	}
}

func TestListNormalizesDatesBeforeCheckingPeriod(t *testing.T) {
	location := time.FixedZone("UTC+03", 3*60*60)
	spentAt := time.Date(2026, time.January, 3, 12, 0, 0, 0, location)
	stored := validExpense(7, 1, 100)
	stored.SpentAt = spentAt
	repository := newFakeRepository(stored)
	service := expense.NewService(repository, &fakeClock{now: time.Now()}, location)
	from := time.Date(2026, time.January, 3, 23, 0, 0, 0, location)
	to := time.Date(2026, time.January, 3, 1, 0, 0, 0, location)

	listed, err := service.List(context.Background(), expense.ListFilter{From: &from, To: &to})
	if err != nil {
		t.Fatalf("List() error = %v, want same-day period to be valid", err)
	}
	if len(listed) != 1 {
		t.Fatalf("List() returned %d expenses, want 1", len(listed))
	}
	if repository.lastFilter.From == nil || repository.lastFilter.To == nil || !repository.lastFilter.From.Equal(*repository.lastFilter.To) {
		t.Errorf("repository received unnormalized bounds: %+v", repository.lastFilter)
	}
}

func TestListRejectsReversedCalendarPeriod(t *testing.T) {
	location := time.FixedZone("UTC+03", 3*60*60)
	repository := newFakeRepository()
	service := expense.NewService(repository, &fakeClock{now: time.Now()}, location)
	from := time.Date(2026, time.January, 4, 0, 0, 0, 0, location)
	to := time.Date(2026, time.January, 3, 23, 0, 0, 0, location)

	if _, err := service.List(context.Background(), expense.ListFilter{From: &from, To: &to}); !errors.Is(err, expense.ErrInvalidPeriod) {
		t.Fatalf("List() error = %v, want ErrInvalidPeriod", err)
	}
	if repository.listCalls != 0 {
		t.Errorf("repository List called %d times for invalid period, want 0", repository.listCalls)
	}
}

func TestListUserIDsReturnsUniqueOwners(t *testing.T) {
	repository := newFakeRepository(
		validExpense(9, 1, 100),
		validExpense(7, 2, 200),
		validExpense(9, 3, 300),
	)
	service := expense.NewService(repository, &fakeClock{now: time.Now()}, time.UTC)

	got, err := service.ListUserIDs(context.Background())
	if err != nil {
		t.Fatalf("ListUserIDs() error = %v, want nil", err)
	}
	if len(got) != 2 || got[0] != 7 || got[1] != 9 {
		t.Errorf("ListUserIDs() = %v, want [7 9]", got)
	}
}

type fakeClock struct {
	now time.Time
}

func (clock *fakeClock) Now() time.Time {
	return clock.now
}

type fakeRepository struct {
	records    map[int64]domain.Expense
	nextID     int64
	listCalls  int
	lastFilter expense.ListFilter
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
	repository.listCalls++
	repository.lastFilter = filter
	items := make([]domain.Expense, 0, len(repository.records))
	for _, item := range repository.records {
		if filter.UserID != nil && item.UserID != *filter.UserID {
			continue
		}
		if filter.Category != nil && item.Category != *filter.Category {
			continue
		}
		if filter.From != nil && domain.CompareDates(item.SpentAt, *filter.From, filter.From.Location()) < 0 {
			continue
		}
		if filter.To != nil && domain.CompareDates(item.SpentAt, *filter.To, filter.To.Location()) > 0 {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].SpentAt.After(items[j].SpentAt)
	})
	return items, nil
}

func (repository *fakeRepository) ListUserIDs(_ context.Context) ([]int64, error) {
	seen := make(map[int64]struct{}, len(repository.records))
	userIDs := make([]int64, 0, len(repository.records))
	for _, item := range repository.records {
		if _, exists := seen[item.UserID]; exists {
			continue
		}
		seen[item.UserID] = struct{}{}
		userIDs = append(userIDs, item.UserID)
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })
	return userIDs, nil
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

func TestCreateDefaultsOmittedDateToTodayInConfiguredLocation(t *testing.T) {
	moscow := time.FixedZone("UTC+03", 3*60*60)
	// 22:30 UTC on 1 Jan is already 2 Jan in Moscow.
	now := time.Date(2026, time.January, 1, 22, 30, 0, 0, time.UTC)
	service := expense.NewService(newFakeRepository(), &fakeClock{now: now}, moscow)
	item := validExpense(7, 0, 100)
	item.SpentAt = time.Time{}

	created, err := service.Create(context.Background(), item)
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	if got, want := domain.FormatDate(created.SpentAt, moscow), "2026-01-02"; got != want {
		t.Errorf("SpentAt = %s, want %s", got, want)
	}
}
