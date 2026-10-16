package postgres

import (
	"context"
	"fmt"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MemberRepository struct {
	pool *pgxpool.Pool
}

func NewMemberRepository(pool *pgxpool.Pool) *MemberRepository {
	return &MemberRepository{pool: pool}
}

// Принимает: ctx — контекст, member — участник с Telegram ID и данными профиля.
// Делает: вставляет участника или обновляет имя и ник существующего, сдвигая updated_at.
// Возвращает: ошибку БД либо nil.
func (repository *MemberRepository) Upsert(ctx context.Context, member domain.Member) error {
	const query = `INSERT INTO members (user_id, display_name, username)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET display_name = EXCLUDED.display_name, username = EXCLUDED.username, updated_at = now()`
	if _, err := repository.pool.Exec(ctx, query, member.ID().Int64(), member.Name(), member.Username()); err != nil {
		return fmt.Errorf("upsert member %d: %w", member.ID().Int64(), err)
	}
	return nil
}

// Принимает: ctx — контекст.
// Делает: читает всех участников по возрастанию ID и собирает их через доменные конструкторы.
// Возвращает: непустой срез (пустой, но не nil, если участников нет) либо ошибку БД или повреждённой строки.
func (repository *MemberRepository) List(ctx context.Context) ([]domain.Member, error) {
	rows, err := repository.pool.Query(ctx, `SELECT user_id, display_name, username FROM members ORDER BY user_id`)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	members := []domain.Member{}
	for rows.Next() {
		var (
			rawID    int64
			name     string
			username string
		)
		if err := rows.Scan(&rawID, &name, &username); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		id, err := domain.NewUserID(rawID)
		if err != nil {
			return nil, fmt.Errorf("member row with user_id %d: %w", rawID, err)
		}
		member, err := domain.NewMember(id, name, username)
		if err != nil {
			return nil, fmt.Errorf("rebuild member %d: %w", rawID, err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return members, nil
}

var _ ports.MemberRepository = (*MemberRepository)(nil)
