// Package postgres — хранение расходов в PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// expenseColumns — список колонок таблицы expenses в порядке, который читает scanExpense.
const expenseColumns = "id, user_id, amount, category, spent_at, comment, created_at, updated_at"

// Repository — хранение расходов в PostgreSQL через pgx.
type Repository struct {
	pool *pgxpool.Pool
}

// Проверка на этапе компиляции, что Repository реализует порт ExpenseRepository.
var _ ports.ExpenseRepository = (*Repository)(nil)

// Принимает: pool — пул соединений.
// Делает: привязывает пул; дни хранятся типом DATE без часового пояса, поэтому пояс не нужен.
// Возвращает: готовый *Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Принимает: ctx — контекст запроса; item — новый расход без ID.
// Делает: вставляет расход одним INSERT ... RETURNING.
// Возвращает: сохранённый расход с присвоенным ID или ошибку БД.
func (repository *Repository) Create(ctx context.Context, item domain.Expense) (domain.Expense, error) {
	query := `INSERT INTO expenses (user_id, amount, category, spent_at, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + expenseColumns
	created, err := scanExpense(repository.pool.QueryRow(ctx, query,
		item.Owner().Int64(),
		item.Amount().Kopecks(),
		string(item.Category()),
		toPostgresDate(item.SpentOn()),
		item.Comment().String(),
		item.CreatedAt(),
		item.UpdatedAt(),
	))
	if err != nil {
		return domain.Expense{}, fmt.Errorf("create expense: %w", err)
	}
	return created, nil
}

// Принимает: ctx — контекст запроса; id — идентификатор расхода.
// Делает: читает расход по ID.
// Возвращает: расход или ports.ErrNotFound, если записи нет.
func (repository *Repository) Get(ctx context.Context, id domain.ExpenseID) (domain.Expense, error) {
	item, err := scanExpense(repository.pool.QueryRow(ctx,
		"SELECT "+expenseColumns+" FROM expenses WHERE id = $1", int64(id)))
	if err != nil {
		return domain.Expense{}, fmt.Errorf("get expense %d: %w", id, err)
	}
	return item, nil
}

// Принимает: ctx — контекст запроса; filter — необязательные условия по владельцу, категории и дням.
// Делает: выбирает расходы, применяя заданные условия (дни включительно), и сортирует по дню трат по убыванию, затем по ID.
// Возвращает: срез расходов (пустой, не nil) или ошибку БД.
func (repository *Repository) List(ctx context.Context, filter ports.ListFilter) ([]domain.Expense, error) {
	var owner, category, from, to any
	if filter.Owner != nil {
		owner = filter.Owner.Int64()
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
		ORDER BY spent_at DESC, id DESC`, owner, category, from, to)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Expense, 0)
	for rows.Next() {
		item, err := scanExpense(rows)
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

// Принимает: ctx — контекст запроса.
// Делает: выбирает уникальных владельцев расходов.
// Возвращает: идентификаторы пользователей по возрастанию или ошибку БД.
func (repository *Repository) ListUserIDs(ctx context.Context) ([]domain.UserID, error) {
	rows, err := repository.pool.Query(ctx, "SELECT DISTINCT user_id FROM expenses ORDER BY user_id")
	if err != nil {
		return nil, fmt.Errorf("list expense owners: %w", err)
	}
	defer rows.Close()

	userIDs := make([]domain.UserID, 0)
	for rows.Next() {
		var raw int64
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan expense owner: %w", err)
		}
		userID, err := domain.NewUserID(raw)
		if err != nil {
			return nil, fmt.Errorf("restore expense owner %d: %w", raw, err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expense owners: %w", err)
	}
	return userIDs, nil
}

// Принимает: ctx — контекст запроса; item — расход с ID и новыми значениями.
// Делает: обновляет только изменяемые поля (сумма, категория, день, комментарий, updated_at); владелец и created_at не меняются.
// Возвращает: сохранённый расход или ports.ErrNotFound, если записи нет.
func (repository *Repository) Update(ctx context.Context, item domain.Expense) (domain.Expense, error) {
	query := `UPDATE expenses
		SET amount = $2, category = $3, spent_at = $4, comment = $5, updated_at = $6
		WHERE id = $1
		RETURNING ` + expenseColumns
	updated, err := scanExpense(repository.pool.QueryRow(ctx, query,
		int64(item.ID()),
		item.Amount().Kopecks(),
		string(item.Category()),
		toPostgresDate(item.SpentOn()),
		item.Comment().String(),
		item.UpdatedAt(),
	))
	if err != nil {
		return domain.Expense{}, fmt.Errorf("update expense %d: %w", item.ID(), err)
	}
	return updated, nil
}

// Принимает: ctx — контекст запроса; id — идентификатор расхода.
// Делает: удаляет расход по ID.
// Возвращает: nil при успехе, ports.ErrNotFound если ничего не удалено, либо ошибку БД.
func (repository *Repository) Delete(ctx context.Context, id domain.ExpenseID) error {
	result, err := repository.pool.Exec(ctx, "DELETE FROM expenses WHERE id = $1", int64(id))
	if err != nil {
		return fmt.Errorf("delete expense %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete expense %d: %w", id, ports.ErrNotFound)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

// Принимает: row — строка результата запроса.
// Делает: считывает колонки, собирает domain.Snapshot из значений-объектов и восстанавливает агрегат; бесконечные и пустые даты отвергаются.
// Возвращает: расход, ports.ErrNotFound при отсутствии строки либо ошибку чтения или повреждённой записи (не ErrNotFound).
func scanExpense(row rowScanner) (domain.Expense, error) {
	var (
		id, userID, amount int64
		category, comment  string
		spentOn            pgtype.Date
		createdAt          time.Time
		updatedAt          time.Time
	)
	if err := row.Scan(&id, &userID, &amount, &category, &spentOn, &comment, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Expense{}, ports.ErrNotFound
		}
		return domain.Expense{}, fmt.Errorf("scan expense: %w", err)
	}
	if !spentOn.Valid || spentOn.InfinityModifier != pgtype.Finite {
		return domain.Expense{}, fmt.Errorf("corrupted expense %d: %w", id, domain.ErrInvalidDate)
	}

	owner, err := domain.NewUserID(userID)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("corrupted expense %d: %w", id, err)
	}
	money, err := domain.NewMoney(amount)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("corrupted expense %d: %w", id, err)
	}
	year, month, day := spentOn.Time.Date()
	spentDay, err := domain.NewDay(year, month, day)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("corrupted expense %d: %w", id, err)
	}
	note, err := domain.NewComment(comment)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("corrupted expense %d: %w", id, err)
	}
	restored, err := domain.Restore(domain.Snapshot{
		ID: domain.ExpenseID(id), Owner: owner, Amount: money, Category: domain.Category(category),
		SpentOn: spentDay, Comment: note, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
	if err != nil {
		return domain.Expense{}, fmt.Errorf("corrupted expense %d: %w", id, err)
	}
	return restored, nil
}

// Принимает: day — календарный день.
// Делает: превращает день в pgtype.Date по его UTC-полуночи, без сдвигов часовых поясов.
// Возвращает: валидный pgtype.Date.
func toPostgresDate(day domain.Day) pgtype.Date {
	return pgtype.Date{Time: day.Time(), Valid: true}
}
