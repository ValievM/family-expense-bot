package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/adapters/postgres"
	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/domain/domaintest"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Проверяет: Upsert вставляет нового участника, List возвращает его данные.
func TestMemberRepositoryUpsertInserts(t *testing.T) {
	repository, ctx := newMemberRepository(t)
	member := newMember(t, domaintest.UserID(t, 101), "Магомед", "valiev")

	if err := repository.Upsert(ctx, member); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0] != member {
		t.Fatalf("got %+v, want [%+v]", got, member)
	}
}

// Проверяет: повторный Upsert обновляет имя и ник, строка остаётся одна.
func TestMemberRepositoryUpsertTwiceUpdates(t *testing.T) {
	repository, ctx := newMemberRepository(t)
	for _, name := range []string{"Старое", "Новое"} {
		if err := repository.Upsert(ctx, newMember(t, domaintest.UserID(t, 101), name, name+"_nick")); err != nil {
			t.Fatalf("upsert %q: %v", name, err)
		}
	}
	got, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Name() != "Новое" || got[0].Username() != "Новое_nick" {
		t.Fatalf("got %+v, want one member with the latest name and username", got)
	}
}

// Проверяет: List отдаёт участников по возрастанию ID независимо от порядка вставки.
func TestMemberRepositoryListOrderedByID(t *testing.T) {
	repository, ctx := newMemberRepository(t)
	for _, id := range []domain.UserID{domaintest.UserID(t, 202), domaintest.UserID(t, 101)} {
		if err := repository.Upsert(ctx, newMember(t, id, "name", "nick")); err != nil {
			t.Fatalf("upsert %d: %v", id.Int64(), err)
		}
	}
	got, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 || got[0].ID().Int64() >= got[1].ID().Int64() {
		t.Fatalf("members are not ordered by id: %+v", got)
	}
}

// Проверяет: при пустой таблице List возвращает пустой срез, а не nil.
func TestMemberRepositoryListEmptyIsNotNil(t *testing.T) {
	repository, ctx := newMemberRepository(t)
	got, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("got %#v, want empty non-nil slice", got)
	}
}

func newMemberRepository(t *testing.T) (*postgres.MemberRepository, context.Context) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL or run make test-integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open migration database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "TRUNCATE members"); err != nil {
		t.Fatalf("truncate members: %v", err)
	}
	return postgres.NewMemberRepository(pool), ctx
}

func newMember(t *testing.T, id domain.UserID, name, username string) domain.Member {
	t.Helper()
	member, err := domain.NewMember(id, name, username)
	if err != nil {
		t.Fatalf("new member: %v", err)
	}
	return member
}
