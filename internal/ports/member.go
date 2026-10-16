package ports

import (
	"context"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

type MemberRepository interface {
	Upsert(ctx context.Context, member domain.Member) error
	List(ctx context.Context) ([]domain.Member, error)
}

type MemberDirectory interface {
	Remember(ctx context.Context, userID int64, name, username string) error
}

type MemberCatalog interface {
	List(ctx context.Context) ([]domain.Member, error)
}
