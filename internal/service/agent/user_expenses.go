// Package agent — контракты, которые использует агент учёта расходов.
package agent

import (
	"context"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

// UserExpenses — операции с расходами, привязанные к личности Telegram-пользователя.
type UserExpenses interface {
	// Принимает: ctx — контекст запроса; expense — новый расход.
	// Делает: создаёт расход от имени пользователя.
	// Возвращает: созданный расход или ошибку.
	Create(ctx context.Context, expense domain.Expense) (domain.Expense, error)
	// Принимает: ctx — контекст запроса; id — идентификатор расхода.
	// Делает: читает расход пользователя.
	// Возвращает: расход или ошибку (в том числе если он не найден).
	Get(ctx context.Context, id int64) (domain.Expense, error)
	// Принимает: ctx — контекст запроса; filter — условия отбора.
	// Делает: выбирает расходы пользователя по фильтру.
	// Возвращает: срез расходов или ошибку.
	List(ctx context.Context, filter expense.ListFilter) ([]domain.Expense, error)
	// Принимает: ctx — контекст запроса; id — идентификатор расхода; patch — изменяемые поля.
	// Делает: частично обновляет расход пользователя.
	// Возвращает: обновлённый расход или ошибку.
	Update(ctx context.Context, id int64, patch expense.Patch) (domain.Expense, error)
	// Принимает: ctx — контекст запроса; id — идентификатор расхода.
	// Делает: удаляет расход пользователя.
	// Возвращает: nil при успехе или ошибку.
	Delete(ctx context.Context, id int64) error
}

var _ UserExpenses = expense.UserExpenses{}
