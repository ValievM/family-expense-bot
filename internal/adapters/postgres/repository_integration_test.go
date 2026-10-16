// Package postgres_test — интеграционные тесты репозитория на реальной PostgreSQL.
package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/adapters/postgres"
	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Принимает: t — тестовый контекст.
// Делает: проверяет создание, чтение, фильтрацию, порядок, список владельцев, обновление и удаление расходов в реальной PostgreSQL, включая ErrNotFound.
// Возвращает: ничего.
func TestRepositoryPostgresCRUDAndFilters(t *testing.T) {
	repository := newPostgresRepository(t)
	ctx := context.Background()
	base := time.Now().UnixNano()
	owner := mustUser(t, base)
	other := mustUser(t, base+1)

	createdAt := time.Date(2025, time.February, 20, 12, 30, 0, 0, time.UTC)
	inputs := []domain.Expense{
		newExpense(t, owner, 1250, domain.CategoryFood, "2025-02-10", "food on end date", createdAt),
		newExpense(t, owner, 2500, domain.CategoryTransport, "2025-02-08", "transport", createdAt),
		newExpense(t, other, 3000, domain.CategoryFood, "2025-02-10", "other owner", createdAt),
		newExpense(t, owner, 900, domain.CategoryFood, "2025-02-05", "food on start date", createdAt),
	}
	created := make([]domain.Expense, 0, len(inputs))
	t.Cleanup(func() {
		for _, item := range created {
			_ = repository.Delete(context.Background(), item.ID())
		}
	})
	for index, input := range inputs {
		got, err := repository.Create(ctx, input)
		if err != nil {
			t.Fatalf("Create(%d): %v", index, err)
		}
		created = append(created, got)
		if got.ID() <= 0 {
			t.Fatalf("Create(%d) returned non-positive ID: %d", index, got.ID())
		}
		assertSameValues(t, got, input)
	}

	got, err := repository.Get(ctx, created[0].ID())
	if err != nil {
		t.Fatalf("Get(%d): %v", created[0].ID(), err)
	}
	if got.ID() != created[0].ID() {
		t.Fatalf("Get returned ID %d, want %d", got.ID(), created[0].ID())
	}
	assertSameValues(t, got, inputs[0])

	allForOwner, err := repository.List(ctx, expense.ListFilter{Owner: &owner})
	if err != nil {
		t.Fatalf("List by owner: %v", err)
	}
	assertIDs(t, "List by owner", allForOwner, created[0].ID(), created[1].ID(), created[3].ID())

	from, to := mustDay(t, "2025-02-05"), mustDay(t, "2025-02-10")
	category := domain.CategoryFood
	filtered, err := repository.List(ctx, expense.ListFilter{Owner: &owner, Category: &category, From: &from, To: &to})
	if err != nil {
		t.Fatalf("List by owner, category, and inclusive period: %v", err)
	}
	assertIDs(t, "filtered List", filtered, created[0].ID(), created[3].ID())

	userIDs, err := repository.ListUserIDs(ctx)
	if err != nil {
		t.Fatalf("ListUserIDs: %v", err)
	}
	if !strictlyAscending(userIDs) || !contains(userIDs, owner) || !contains(userIDs, other) {
		t.Fatalf("ListUserIDs did not return distinct ascending owners including test users: %v", userIDs)
	}

	changes := domain.Changes{
		Amount:   ptr(mustMoney(t, 5550)),
		Category: ptr(domain.CategoryHousing),
		SpentOn:  ptr(mustDay(t, "2025-02-12")),
		Comment:  ptr(mustComment(t, "updated comment")),
	}
	updatedAt := createdAt.Add(24 * time.Hour)
	toUpdate, err := created[0].Apply(changes, updatedAt)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	updated, err := repository.Update(ctx, toUpdate)
	if err != nil {
		t.Fatalf("Update(%d): %v", toUpdate.ID(), err)
	}
	assertSameValues(t, updated, toUpdate)
	if updated.Owner() != owner || !updated.CreatedAt().Equal(createdAt) {
		t.Fatalf("Update changed immutable owner or creation time: %+v", updated)
	}

	if err := repository.Delete(ctx, created[2].ID()); err != nil {
		t.Fatalf("Delete(%d): %v", created[2].ID(), err)
	}
	if _, err := repository.Get(ctx, created[2].ID()); !errors.Is(err, expense.ErrNotFound) {
		t.Fatalf("Get after Delete error = %v, want ErrNotFound", err)
	}
	if _, err := repository.Update(ctx, created[2]); !errors.Is(err, expense.ErrNotFound) {
		t.Fatalf("Update missing expense error = %v, want ErrNotFound", err)
	}
	if err := repository.Delete(ctx, created[2].ID()); !errors.Is(err, expense.ErrNotFound) {
		t.Fatalf("Delete missing expense error = %v, want ErrNotFound", err)
	}
	userIDs, err = repository.ListUserIDs(ctx)
	if err != nil {
		t.Fatalf("ListUserIDs after Delete: %v", err)
	}
	if contains(userIDs, other) {
		t.Fatalf("ListUserIDs still contains owner %d after its expense was deleted: %v", other.Int64(), userIDs)
	}
}

// Принимает: t — тестовый контекст.
// Делает: создаёт расходы на границах месяца и года и проверяет, что день, созданный через ParseDay, возвращается с тем же String().
// Возвращает: ничего.
func TestRepositoryPostgresDayRoundTripWithoutDrift(t *testing.T) {
	repository := newPostgresRepository(t)
	ctx := context.Background()
	owner := mustUser(t, time.Now().UnixNano())
	createdAt := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	for _, value := range []string{"2025-01-01", "2024-02-29", "2025-12-31", "2025-06-15"} {
		created, err := repository.Create(ctx, newExpense(t, owner, 100, domain.CategoryOther, value, "", createdAt))
		if err != nil {
			t.Fatalf("Create(%s): %v", value, err)
		}
		id := created.ID()
		t.Cleanup(func() { _ = repository.Delete(context.Background(), id) })

		got, err := repository.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get(%s): %v", value, err)
		}
		if created.SpentOn().String() != value || got.SpentOn().String() != value {
			t.Fatalf("day %s came back as created=%s, got=%s", value, created.SpentOn(), got.SpentOn())
		}
	}
}

// Принимает: t — тестовый контекст.
// Делает: подключается к БД из TEST_DATABASE_URL (без неё пропускает тест), применяет миграции и создаёт пул и репозиторий.
// Возвращает: *postgres.Repository.
func newPostgresRepository(t *testing.T) *postgres.Repository {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL or run make test-integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open migration database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.PingContext(ctx); err != nil {
		t.Fatalf("ping migration database: %v", err)
	}
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping pgx pool: %v", err)
	}
	return postgres.NewRepository(pool)
}

// Принимает: t — тестовый контекст; owner, kopecks, category, day, comment, createdAt — поля расхода (day в формате YYYY-MM-DD).
// Делает: собирает новый расход через доменный конструктор.
// Возвращает: Expense без ID.
func newExpense(t *testing.T, owner domain.UserID, kopecks int64, category domain.Category, day, comment string, createdAt time.Time) domain.Expense {
	t.Helper()
	item, err := domain.NewExpense(domain.NewExpenseInput{
		Owner:    owner,
		Amount:   mustMoney(t, kopecks),
		Category: category,
		SpentOn:  mustDay(t, day),
		Comment:  mustComment(t, comment),
		At:       createdAt,
	})
	if err != nil {
		t.Fatalf("NewExpense: %v", err)
	}
	return item
}

// Принимает: t — тестовый контекст; value — Telegram ID.
// Делает: создаёт UserID.
// Возвращает: UserID.
func mustUser(t *testing.T, value int64) domain.UserID {
	t.Helper()
	user, err := domain.NewUserID(value)
	if err != nil {
		t.Fatalf("NewUserID(%d): %v", value, err)
	}
	return user
}

// Принимает: t — тестовый контекст; kopecks — сумма в копейках.
// Делает: создаёт Money.
// Возвращает: Money.
func mustMoney(t *testing.T, kopecks int64) domain.Money {
	t.Helper()
	money, err := domain.NewMoney(kopecks)
	if err != nil {
		t.Fatalf("NewMoney(%d): %v", kopecks, err)
	}
	return money
}

// Принимает: t — тестовый контекст; value — день в формате YYYY-MM-DD.
// Делает: разбирает день через domain.ParseDay.
// Возвращает: Day.
func mustDay(t *testing.T, value string) domain.Day {
	t.Helper()
	day, err := domain.ParseDay(value)
	if err != nil {
		t.Fatalf("ParseDay(%q): %v", value, err)
	}
	return day
}

// Принимает: t — тестовый контекст; text — текст комментария.
// Делает: создаёт Comment.
// Возвращает: Comment.
func mustComment(t *testing.T, text string) domain.Comment {
	t.Helper()
	comment, err := domain.NewComment(text)
	if err != nil {
		t.Fatalf("NewComment(%q): %v", text, err)
	}
	return comment
}

// Принимает: value — любое значение.
// Делает: берёт его адрес для необязательных полей Changes.
// Возвращает: указатель на копию значения.
func ptr[T any](value T) *T {
	return &value
}

// Принимает: t — тестовый контекст; got, want — сравниваемые расходы.
// Делает: сравнивает все поля, кроме ID, времена через Equal.
// Возвращает: ничего.
func assertSameValues(t *testing.T, got, want domain.Expense) {
	t.Helper()
	if got.Owner() != want.Owner() || got.Amount() != want.Amount() || got.Category() != want.Category() ||
		got.SpentOn() != want.SpentOn() || got.Comment() != want.Comment() {
		t.Fatalf("expense values differ: got %+v, want %+v", got, want)
	}
	if !got.CreatedAt().Equal(want.CreatedAt()) || !got.UpdatedAt().Equal(want.UpdatedAt()) {
		t.Fatalf("expense timestamps differ: got %v/%v, want %v/%v", got.CreatedAt(), got.UpdatedAt(), want.CreatedAt(), want.UpdatedAt())
	}
}

// Принимает: t — тестовый контекст; name — имя проверки; got — список расходов; want — ожидаемые ID по порядку.
// Делает: сверяет порядок ID в списке.
// Возвращает: ничего.
func assertIDs(t *testing.T, name string, got []domain.Expense, want ...domain.ExpenseID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s returned %d rows, want %d: %+v", name, len(got), len(want), got)
	}
	for index, id := range want {
		if got[index].ID() != id {
			t.Fatalf("%s row %d has ID %d, want %d", name, index, got[index].ID(), id)
		}
	}
}

// Принимает: values — список владельцев.
// Делает: проверяет, что значения идут строго по возрастанию.
// Возвращает: true, если порядок строго возрастающий.
func strictlyAscending(values []domain.UserID) bool {
	for index := 1; index < len(values); index++ {
		if values[index-1].Int64() >= values[index].Int64() {
			return false
		}
	}
	return true
}

// Принимает: values — список владельцев; target — искомый владелец.
// Делает: ищет владельца в списке.
// Возвращает: true, если владелец найден.
func contains(values []domain.UserID, target domain.UserID) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
