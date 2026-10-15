package expense

import (
	"context"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// ListFilter — фильтры списка: владелец, категория и границы периода по календарным датам включительно.
type ListFilter struct {
	UserID   *int64
	Category *domain.Category
	From     *time.Time
	To       *time.Time
}

// Repository — операции хранилища для Service; List сортирует по SpentAt по убыванию, ListUserIDs отдаёт уникальные ID по возрастанию, отсутствующий ID даёт ErrNotFound.
type Repository interface {
	// Принимает: ctx — контекст запроса; expense — сохраняемый расход.
	// Делает: объявляет сохранение нового расхода.
	// Возвращает: сохранённый расход (с присвоенным ID) или ошибку хранилища.
	Create(ctx context.Context, expense domain.Expense) (domain.Expense, error)
	// Принимает: ctx — контекст запроса; id — идентификатор расхода.
	// Делает: объявляет чтение расхода по ID.
	// Возвращает: расход или ErrNotFound, если его нет.
	Get(ctx context.Context, id int64) (domain.Expense, error)
	// Принимает: ctx — контекст запроса; filter — фильтры списка.
	// Делает: объявляет выборку расходов по фильтру, сортировка по SpentAt по убыванию.
	// Возвращает: список расходов или ошибку хранилища.
	List(ctx context.Context, filter ListFilter) ([]domain.Expense, error)
	// Принимает: ctx — контекст запроса.
	// Делает: объявляет получение уникальных владельцев расходов.
	// Возвращает: ID пользователей по возрастанию или ошибку хранилища.
	ListUserIDs(ctx context.Context) ([]int64, error)
	// Принимает: ctx — контекст запроса; expense — расход с новыми значениями.
	// Делает: объявляет сохранение изменений существующего расхода.
	// Возвращает: обновлённый расход или ErrNotFound, если его нет.
	Update(ctx context.Context, expense domain.Expense) (domain.Expense, error)
	// Принимает: ctx — контекст запроса; id — идентификатор расхода.
	// Делает: объявляет удаление расхода по ID.
	// Возвращает: nil или ErrNotFound, если расхода нет.
	Delete(ctx context.Context, id int64) error
}
