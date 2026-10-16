// Package expense_test проверяет сценарии над расходами через публичный контракт сервиса с хранилищем в памяти.
package expense_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/domain/domaintest"
	"github.com/ValievM/family-expense-bot/internal/ports"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

type fakeClock struct{ now time.Time }

func (clock *fakeClock) Now() time.Time { return clock.now }

type fakeRepository struct {
	records map[domain.ExpenseID]domain.Expense
	nextID  domain.ExpenseID
	writes  int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{records: make(map[domain.ExpenseID]domain.Expense)}
}

func (repository *fakeRepository) Create(_ context.Context, item domain.Expense) (domain.Expense, error) {
	repository.nextID++
	repository.writes++
	item = item.WithID(repository.nextID)
	repository.records[item.ID()] = item
	return item, nil
}

func (repository *fakeRepository) Get(_ context.Context, id domain.ExpenseID) (domain.Expense, error) {
	item, found := repository.records[id]
	if !found {
		return domain.Expense{}, ports.ErrNotFound
	}
	return item, nil
}

func (repository *fakeRepository) List(_ context.Context, filter ports.ListFilter) ([]domain.Expense, error) {
	items := make([]domain.Expense, 0, len(repository.records))
	for _, item := range repository.records {
		if filter.Owner != nil && item.Owner() != *filter.Owner {
			continue
		}
		if filter.Category != nil && item.Category() != *filter.Category {
			continue
		}
		if filter.From != nil && item.SpentOn().Compare(*filter.From) < 0 {
			continue
		}
		if filter.To != nil && item.SpentOn().Compare(*filter.To) > 0 {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if cmp := items[i].SpentOn().Compare(items[j].SpentOn()); cmp != 0 {
			return cmp > 0
		}
		return items[i].ID() > items[j].ID()
	})
	return items, nil
}

func (repository *fakeRepository) ListUserIDs(_ context.Context) ([]domain.UserID, error) {
	seen := make(map[domain.UserID]struct{})
	userIDs := make([]domain.UserID, 0)
	for _, item := range repository.records {
		if _, exists := seen[item.Owner()]; !exists {
			seen[item.Owner()] = struct{}{}
			userIDs = append(userIDs, item.Owner())
		}
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i].Int64() < userIDs[j].Int64() })
	return userIDs, nil
}

func (repository *fakeRepository) Update(_ context.Context, item domain.Expense) (domain.Expense, error) {
	if _, found := repository.records[item.ID()]; !found {
		return domain.Expense{}, ports.ErrNotFound
	}
	repository.writes++
	repository.records[item.ID()] = item
	return item, nil
}

func (repository *fakeRepository) Delete(_ context.Context, id domain.ExpenseID) error {
	if _, found := repository.records[id]; !found {
		return ports.ErrNotFound
	}
	repository.writes++
	delete(repository.records, id)
	return nil
}

func dayPtr(t *testing.T, value string) *domain.Day {
	t.Helper()
	parsed := domaintest.Day(t, value)
	return &parsed
}

func input(t *testing.T, owner int64, kopecks int64) ports.CreateInput {
	t.Helper()
	return ports.CreateInput{Owner: domaintest.UserID(t, owner), Amount: domaintest.Money(t, kopecks), Category: domain.CategoryFood, SpentOn: dayPtr(t, "2026-01-02")}
}

func newService(t *testing.T, now time.Time) (*expense.Service, *fakeRepository) {
	t.Helper()
	repository := newFakeRepository()
	return expense.NewService(repository, &fakeClock{now: now}, time.UTC), repository
}

var now = time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)

// Проверяет: время создания и изменения берётся из часов сервиса.
func TestCreateStampsTimestampsFromClock(t *testing.T) {
	service, _ := newService(t, now)

	created, err := service.Create(context.Background(), input(t, 7, 100))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if !created.CreatedAt().Equal(now) || !created.UpdatedAt().Equal(now) || created.ID() == 0 {
		t.Errorf("created = %+v", created)
	}
}

// Проверяет: пропущенный день означает «сегодня» в часовом поясе сервиса, а не в UTC.
func TestCreateDefaultsOmittedDayToTodayInConfiguredLocation(t *testing.T) {
	moscow := time.FixedZone("UTC+03", 3*60*60)
	// 22:30 UTC первого января уже второе января в Москве.
	clock := &fakeClock{now: time.Date(2026, time.January, 1, 22, 30, 0, 0, time.UTC)}
	service := expense.NewService(newFakeRepository(), clock, moscow)
	request := input(t, 7, 100)
	request.SpentOn = nil

	created, err := service.Create(context.Background(), request)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got := created.SpentOn().String(); got != "2026-01-02" {
		t.Errorf("SpentOn = %s, want 2026-01-02", got)
	}
}

// Проверяет: нарушение инварианта не доходит до хранилища.
func TestCreateRejectsInvalidExpenseWithoutPersistence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ports.CreateInput)
		want   error
	}{
		{"zero amount", func(in *ports.CreateInput) { in.Amount = domain.Money{} }, domain.ErrInvalidAmount},
		{"unknown category", func(in *ports.CreateInput) { in.Category = "pets" }, domain.ErrUnknownCategory},
		{"missing owner", func(in *ports.CreateInput) { in.Owner = domain.UserID{} }, domain.ErrInvalidUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repository := newService(t, now)
			request := input(t, 7, 100)
			tt.mutate(&request)

			if _, err := service.Create(context.Background(), request); !errors.Is(err, tt.want) {
				t.Errorf("error = %v, want %v", err, tt.want)
			}
			if repository.writes != 0 {
				t.Errorf("writes = %d, want 0", repository.writes)
			}
		})
	}
}

// Проверяет: частичное обновление: меняются заданные поля, время изменения берётся из часов, остальное сохраняется.
func TestUpdateMergesChangesAndStampsUpdateTime(t *testing.T) {
	service, repository := newService(t, now)
	created, _ := service.Create(context.Background(), input(t, 7, 100))
	later := now.Add(time.Hour)
	service = expense.NewService(repository, &fakeClock{now: later}, time.UTC)
	amount := domaintest.Money(t, 250)
	comment, _ := domain.NewComment("new comment")

	updated, err := service.Update(context.Background(), created.ID(), domain.Changes{Amount: &amount, Comment: &comment})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Amount().Kopecks() != 250 || updated.Comment().String() != "new comment" || updated.Category() != domain.CategoryFood {
		t.Errorf("updated = %+v", updated)
	}
	if !updated.CreatedAt().Equal(now) || !updated.UpdatedAt().Equal(later) {
		t.Errorf("timestamps = (%v, %v), want (%v, %v)", updated.CreatedAt(), updated.UpdatedAt(), now, later)
	}
}

// Проверяет: изменение, нарушающее инвариант, отклоняется доменом и не сохраняется.
func TestUpdateRejectsInvalidResultWithoutPersistence(t *testing.T) {
	service, repository := newService(t, now)
	created, _ := service.Create(context.Background(), input(t, 7, 100))
	writes := repository.writes
	zero := domain.Money{}

	_, err := service.Update(context.Background(), created.ID(), domain.Changes{Amount: &zero})

	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Errorf("error = %v, want ErrInvalidAmount", err)
	}
	if repository.writes != writes || repository.records[created.ID()].Amount().Kopecks() != 100 {
		t.Error("an invalid update reached the repository")
	}
}

// Проверяет: область пользователя не читает, не меняет и не удаляет чужие расходы, а ответ неотличим от «не найдено».
func TestUserExpensesCannotReadOrChangeForeignExpenses(t *testing.T) {
	service, repository := newService(t, now)
	foreign, _ := service.Create(context.Background(), input(t, 8, 900))
	scope := service.ForUser(domaintest.UserID(t, 7))
	amount := domaintest.Money(t, 1)

	if _, err := scope.Get(context.Background(), foreign.ID()); !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
	if _, err := scope.Update(context.Background(), foreign.ID(), domain.Changes{Amount: &amount}); !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("Update() error = %v, want ErrNotFound", err)
	}
	if err := scope.Delete(context.Background(), foreign.ID()); !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
	if got := repository.records[foreign.ID()]; got.Amount().Kopecks() != 900 {
		t.Errorf("foreign expense changed: %+v", got)
	}
}

// Проверяет: создание и список в области пользователя всегда привязаны к владельцу области, даже если в фильтре указан другой.
func TestUserExpensesCreateAndListBindOwner(t *testing.T) {
	service, _ := newService(t, now)
	_, _ = service.Create(context.Background(), input(t, 8, 900))
	scope := service.ForUser(domaintest.UserID(t, 7))

	created, err := scope.Create(context.Background(), ports.UserCreateInput{Amount: domaintest.Money(t, 100), Category: domain.CategoryHealth, SpentOn: dayPtr(t, "2026-01-02")})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	other := domaintest.UserID(t, 8)
	listed, err := scope.List(context.Background(), ports.ListFilter{Owner: &other})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if created.Owner() != domaintest.UserID(t, 7) {
		t.Errorf("owner = %v, want 7", created.Owner())
	}
	if len(listed) != 1 || listed[0].ID() != created.ID() {
		t.Errorf("listed = %+v, want only the scoped owner's expense", listed)
	}
}

// Проверяет: обновление в области пользователя сохраняет владельца и время создания.
func TestUserExpensesUpdatePreservesOwnerAndCreationTime(t *testing.T) {
	service, repository := newService(t, now)
	scope := service.ForUser(domaintest.UserID(t, 7))
	created, _ := scope.Create(context.Background(), ports.UserCreateInput{Amount: domaintest.Money(t, 100), Category: domain.CategoryFood, SpentOn: dayPtr(t, "2026-01-02")})
	service = expense.NewService(repository, &fakeClock{now: now.Add(time.Hour)}, time.UTC)
	category := domain.CategoryShopping

	updated, err := service.ForUser(domaintest.UserID(t, 7)).Update(context.Background(), created.ID(), domain.Changes{Category: &category})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Owner() != domaintest.UserID(t, 7) || !updated.CreatedAt().Equal(now) || updated.Category() != domain.CategoryShopping {
		t.Errorf("updated = %+v", updated)
	}
}

// Проверяет: отказ нулевой области (ForUser с нулевым владельцем) на каждой операции.
func TestZeroScopeIsRejected(t *testing.T) {
	service, _ := newService(t, now)
	scope := service.ForUser(domain.UserID{})

	if _, err := scope.List(context.Background(), ports.ListFilter{}); !errors.Is(err, domain.ErrInvalidUser) {
		t.Errorf("List() error = %v, want ErrInvalidUser", err)
	}
	if _, err := scope.Create(context.Background(), ports.UserCreateInput{Amount: domaintest.Money(t, 1), Category: domain.CategoryFood}); !errors.Is(err, domain.ErrInvalidUser) {
		t.Errorf("Create() error = %v, want ErrInvalidUser", err)
	}
}

// Проверяет: фильтры списка: период включает границы, категория отбирает, порядок «новые сверху».
func TestListFiltersByInclusivePeriodAndCategory(t *testing.T) {
	service, _ := newService(t, now)
	for _, date := range []string{"2026-01-01", "2026-01-02", "2026-01-03", "2026-01-04"} {
		request := input(t, 7, 100)
		request.SpentOn = dayPtr(t, date)
		if _, err := service.Create(context.Background(), request); err != nil {
			t.Fatal(err)
		}
	}
	other := input(t, 7, 100)
	other.Category = domain.CategoryOther
	other.SpentOn = dayPtr(t, "2026-01-02")
	_, _ = service.Create(context.Background(), other)
	food := domain.CategoryFood

	listed, err := service.List(context.Background(), ports.ListFilter{Category: &food, From: dayPtr(t, "2026-01-02"), To: dayPtr(t, "2026-01-03")})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(listed) != 2 || listed[0].SpentOn().String() != "2026-01-03" || listed[1].SpentOn().String() != "2026-01-02" {
		t.Errorf("listed = %+v", listed)
	}
}

// Проверяет: отказ на перевёрнутом периоде и неизвестной категории до обращения к хранилищу.
func TestListRejectsInvalidFilters(t *testing.T) {
	service, _ := newService(t, now)
	unknown := domain.Category("pets")

	if _, err := service.List(context.Background(), ports.ListFilter{From: dayPtr(t, "2026-01-03"), To: dayPtr(t, "2026-01-01")}); !errors.Is(err, ports.ErrInvalidPeriod) {
		t.Errorf("reversed period error = %v, want ErrInvalidPeriod", err)
	}
	if _, err := service.List(context.Background(), ports.ListFilter{Category: &unknown}); !errors.Is(err, domain.ErrUnknownCategory) {
		t.Errorf("unknown category error = %v, want ErrUnknownCategory", err)
	}
}

// Проверяет: список владельцев: уникальные, по возрастанию.
func TestListUserIDsReturnsUniqueOwners(t *testing.T) {
	service, _ := newService(t, now)
	for _, owner := range []int64{9, 7, 9, 8} {
		if _, err := service.Create(context.Background(), input(t, owner, 100)); err != nil {
			t.Fatal(err)
		}
	}

	userIDs, err := service.ListUserIDs(context.Background())
	if err != nil {
		t.Fatalf("ListUserIDs() error = %v", err)
	}

	if len(userIDs) != 3 || userIDs[0].Int64() != 7 || userIDs[1].Int64() != 8 || userIDs[2].Int64() != 9 {
		t.Errorf("userIDs = %v, want [7 8 9]", userIDs)
	}
}
