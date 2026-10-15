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

func TestRepositoryPostgresCRUDAndFilters(t *testing.T) {
	repository, location := newPostgresRepository(t)
	ctx := context.Background()
	userID := time.Now().UnixNano()

	date := func(day int) time.Time {
		return time.Date(2025, time.February, day, 0, 0, 0, 0, location)
	}
	createdAt := time.Date(2025, time.February, 20, 12, 30, 0, 0, time.UTC)
	items := []domain.Expense{
		{UserID: userID, Amount: 1250, Category: domain.CategoryFood, SpentAt: date(10), Comment: "food on end date", CreatedAt: createdAt, UpdatedAt: createdAt},
		{UserID: userID, Amount: 2500, Category: domain.CategoryTransport, SpentAt: date(8), Comment: "transport", CreatedAt: createdAt, UpdatedAt: createdAt},
		{UserID: userID + 1, Amount: 3000, Category: domain.CategoryFood, SpentAt: date(10), Comment: "other owner", CreatedAt: createdAt, UpdatedAt: createdAt},
		{UserID: userID, Amount: 900, Category: domain.CategoryFood, SpentAt: date(5), Comment: "food on start date", CreatedAt: createdAt, UpdatedAt: createdAt},
	}
	created := make([]domain.Expense, 0, len(items))
	t.Cleanup(func() {
		for _, item := range created {
			_ = repository.Delete(context.Background(), item.ID)
		}
	})
	for index, item := range items {
		got, err := repository.Create(ctx, item)
		if err != nil {
			t.Fatalf("Create(%d): %v", index, err)
		}
		created = append(created, got)
		if got.ID <= 0 || got.UserID != item.UserID || got.Amount != item.Amount || got.Category != item.Category || got.Comment != item.Comment {
			t.Fatalf("Create(%d) returned unexpected values: %+v", index, got)
		}
		if !got.SpentAt.Equal(item.SpentAt) || !got.CreatedAt.Equal(item.CreatedAt) || !got.UpdatedAt.Equal(item.UpdatedAt) {
			t.Fatalf("Create(%d) returned unexpected dates or timestamps: %+v", index, got)
		}
	}

	got, err := repository.Get(ctx, created[0].ID)
	if err != nil {
		t.Fatalf("Get(%d): %v", created[0].ID, err)
	}
	if got.ID != created[0].ID || got.UserID != userID || got.Amount != 1250 || got.Category != domain.CategoryFood || got.Comment != "food on end date" {
		t.Fatalf("Get returned unexpected expense: %+v", got)
	}
	if !got.SpentAt.Equal(date(10)) || !got.CreatedAt.Equal(createdAt) || !got.UpdatedAt.Equal(createdAt) {
		t.Fatalf("Get returned unexpected dates or timestamps: %+v", got)
	}

	allForUser, err := repository.List(ctx, expense.ListFilter{UserID: &userID})
	if err != nil {
		t.Fatalf("List by user: %v", err)
	}
	wantOrder := []int64{created[0].ID, created[1].ID, created[3].ID}
	if len(allForUser) != len(wantOrder) {
		t.Fatalf("List by user returned %d rows, want %d: %+v", len(allForUser), len(wantOrder), allForUser)
	}
	for index, wantID := range wantOrder {
		if allForUser[index].ID != wantID {
			t.Fatalf("List order at %d = %d, want %d", index, allForUser[index].ID, wantID)
		}
	}

	from, to := date(5), date(10)
	category := domain.CategoryFood
	filtered, err := repository.List(ctx, expense.ListFilter{UserID: &userID, Category: &category, From: &from, To: &to})
	if err != nil {
		t.Fatalf("List by user, category, and inclusive period: %v", err)
	}
	wantFiltered := []int64{created[0].ID, created[3].ID}
	if len(filtered) != len(wantFiltered) {
		t.Fatalf("filtered List returned %d rows, want %d: %+v", len(filtered), len(wantFiltered), filtered)
	}
	for index, wantID := range wantFiltered {
		if filtered[index].ID != wantID {
			t.Fatalf("filtered List row %d has ID %d, want %d", index, filtered[index].ID, wantID)
		}
	}

	userIDs, err := repository.ListUserIDs(ctx)
	if err != nil {
		t.Fatalf("ListUserIDs: %v", err)
	}
	if !strictlyAscending(userIDs) || !contains(userIDs, userID) || !contains(userIDs, userID+1) {
		t.Fatalf("ListUserIDs did not return distinct ascending owners including test users: %v", userIDs)
	}

	updatedInput := created[0]
	updatedInput.UserID = userID + 99
	updatedInput.CreatedAt = createdAt.Add(-24 * time.Hour)
	updatedInput.Amount = 5550
	updatedInput.Category = domain.CategoryHousing
	updatedInput.SpentAt = date(12)
	updatedInput.Comment = "updated comment"
	updatedInput.UpdatedAt = createdAt.Add(24 * time.Hour)
	updated, err := repository.Update(ctx, updatedInput)
	if err != nil {
		t.Fatalf("Update(%d): %v", updatedInput.ID, err)
	}
	if updated.UserID != userID || !updated.CreatedAt.Equal(createdAt) {
		t.Fatalf("Update changed immutable owner or creation time: %+v", updated)
	}
	if updated.Amount != updatedInput.Amount || updated.Category != updatedInput.Category || !updated.SpentAt.Equal(date(12)) || updated.Comment != updatedInput.Comment || !updated.UpdatedAt.Equal(updatedInput.UpdatedAt) {
		t.Fatalf("Update returned unexpected mutable values: %+v", updated)
	}

	if err := repository.Delete(ctx, created[2].ID); err != nil {
		t.Fatalf("Delete(%d): %v", created[2].ID, err)
	}
	if _, err := repository.Get(ctx, created[2].ID); !errors.Is(err, expense.ErrNotFound) {
		t.Fatalf("Get after Delete error = %v, want ErrNotFound", err)
	}
	missing := created[2]
	missing.Amount++
	if _, err := repository.Update(ctx, missing); !errors.Is(err, expense.ErrNotFound) {
		t.Fatalf("Update missing expense error = %v, want ErrNotFound", err)
	}
	if err := repository.Delete(ctx, created[2].ID); !errors.Is(err, expense.ErrNotFound) {
		t.Fatalf("Delete missing expense error = %v, want ErrNotFound", err)
	}
	userIDs, err = repository.ListUserIDs(ctx)
	if err != nil {
		t.Fatalf("ListUserIDs after Delete: %v", err)
	}
	if contains(userIDs, userID+1) {
		t.Fatalf("ListUserIDs still contains owner %d after its expense was deleted: %v", userID+1, userIDs)
	}

}

func newPostgresRepository(t *testing.T) (*postgres.Repository, *time.Location) {
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

	location := time.FixedZone("test-calendar", -8*60*60)
	return postgres.NewRepository(pool, location), location
}

func strictlyAscending(values []int64) bool {
	for index := 1; index < len(values); index++ {
		if values[index-1] >= values[index] {
			return false
		}
	}
	return true
}

func contains(values []int64, target int64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
