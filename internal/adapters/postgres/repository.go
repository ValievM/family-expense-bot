// Package postgres persists expenses in PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const expenseColumns = "id, user_id, amount, category, spent_at, comment, created_at, updated_at"

// Repository implements expense persistence with pgx.
type Repository struct {
	pool     *pgxpool.Pool
	location *time.Location
}

var _ expense.Repository = (*Repository)(nil)

// NewRepository binds the pool and calendar location used to map PostgreSQL DATE values.
func NewRepository(pool *pgxpool.Pool, location *time.Location) *Repository {
	if location == nil {
		location = time.UTC
	}
	return &Repository{pool: pool, location: location}
}

// Create inserts an expense and returns its generated ID and stored values.
func (repository *Repository) Create(ctx context.Context, item domain.Expense) (domain.Expense, error) {
	query := `INSERT INTO expenses (user_id, amount, category, spent_at, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + expenseColumns
	created, err := repository.scanExpense(repository.pool.QueryRow(ctx, query,
		item.UserID,
		item.Amount,
		string(item.Category),
		toPostgresDate(item.SpentAt),
		item.Comment,
		item.CreatedAt,
		item.UpdatedAt,
	))
	if err != nil {
		return domain.Expense{}, fmt.Errorf("create expense: %w", err)
	}
	return created, nil
}

// Get returns an expense or expense.ErrNotFound.
func (repository *Repository) Get(ctx context.Context, id int64) (domain.Expense, error) {
	item, err := repository.scanExpense(repository.pool.QueryRow(ctx,
		"SELECT "+expenseColumns+" FROM expenses WHERE id = $1", id))
	if err != nil {
		return domain.Expense{}, fmt.Errorf("get expense %d: %w", id, err)
	}
	return item, nil
}

// List applies inclusive calendar-date filters and orders by spent date descending.
func (repository *Repository) List(ctx context.Context, filter expense.ListFilter) ([]domain.Expense, error) {
	var userID, category, from, to any
	if filter.UserID != nil {
		userID = *filter.UserID
	}
	if filter.Category != nil {
		category = string(*filter.Category)
	}
	if filter.From != nil {
		from = toPostgresDate(*filter.From)
	}
	if filter.To != nil {
		to = toPostgresDate(*filter.To)
	}

	rows, err := repository.pool.Query(ctx, `SELECT `+expenseColumns+` FROM expenses
		WHERE ($1::BIGINT IS NULL OR user_id = $1)
		  AND ($2::TEXT IS NULL OR category = $2)
		  AND ($3::DATE IS NULL OR spent_at >= $3)
		  AND ($4::DATE IS NULL OR spent_at <= $4)
		ORDER BY spent_at DESC, id DESC`, userID, category, from, to)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Expense, 0)
	for rows.Next() {
		item, err := repository.scanExpense(rows)
		if err != nil {
			return nil, fmt.Errorf("list expenses: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expenses: %w", err)
	}
	return items, nil
}

// ListUserIDs returns distinct owners in ascending order.
func (repository *Repository) ListUserIDs(ctx context.Context) ([]int64, error) {
	rows, err := repository.pool.Query(ctx, "SELECT DISTINCT user_id FROM expenses ORDER BY user_id")
	if err != nil {
		return nil, fmt.Errorf("list expense owners: %w", err)
	}
	defer rows.Close()

	userIDs := make([]int64, 0)
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan expense owner: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expense owners: %w", err)
	}
	return userIDs, nil
}

// Update replaces mutable expense values and returns the stored row.
func (repository *Repository) Update(ctx context.Context, item domain.Expense) (domain.Expense, error) {
	query := `UPDATE expenses
		SET amount = $2, category = $3, spent_at = $4, comment = $5, updated_at = $6
		WHERE id = $1
		RETURNING ` + expenseColumns
	updated, err := repository.scanExpense(repository.pool.QueryRow(ctx, query,
		item.ID,
		item.Amount,
		string(item.Category),
		toPostgresDate(item.SpentAt),
		item.Comment,
		item.UpdatedAt,
	))
	if err != nil {
		return domain.Expense{}, fmt.Errorf("update expense %d: %w", item.ID, err)
	}
	return updated, nil
}

// Delete removes an expense or returns expense.ErrNotFound.
func (repository *Repository) Delete(ctx context.Context, id int64) error {
	result, err := repository.pool.Exec(ctx, "DELETE FROM expenses WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete expense %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete expense %d: %w", id, expense.ErrNotFound)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (repository *Repository) scanExpense(row rowScanner) (domain.Expense, error) {
	var item domain.Expense
	var category string
	var spentAt pgtype.Date
	if err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.Amount,
		&category,
		&spentAt,
		&item.Comment,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Expense{}, expense.ErrNotFound
		}
		return domain.Expense{}, fmt.Errorf("scan expense: %w", err)
	}
	if !spentAt.Valid || spentAt.InfinityModifier != pgtype.Finite {
		return domain.Expense{}, fmt.Errorf("scan expense date: %w", domain.ErrInvalidDate)
	}
	year, month, day := spentAt.Time.Date()
	item.SpentAt = time.Date(year, month, day, 0, 0, 0, 0, repository.location)
	item.Category = domain.Category(category)
	return item, nil
}

func toPostgresDate(value time.Time) pgtype.Date {
	year, month, day := value.Date()
	return pgtype.Date{Time: time.Date(year, month, day, 0, 0, 0, 0, time.UTC), Valid: true}
}
