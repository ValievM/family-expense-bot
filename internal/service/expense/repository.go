// Package expense — сценарии работы с расходами (слой приложения): создание, чтение, изменение, удаление и привязка к владельцу.
package expense

import (
	"context"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// ListFilter — фильтры списка: владелец, категория и включительный период в календарных днях.
type ListFilter struct {
	Owner    *domain.UserID
	Category *domain.Category
	From     *domain.Day
	To       *domain.Day
}

// CreateInput — данные создания расхода; пустой SpentOn означает «сегодня».
type CreateInput struct {
	Owner    domain.UserID
	Amount   domain.Money
	Category domain.Category
	SpentOn  *domain.Day
	Comment  domain.Comment
}

// UserCreateInput — данные создания расхода от имени пользователя; владельца задаёт область UserExpenses.
type UserCreateInput struct {
	Amount   domain.Money
	Category domain.Category
	SpentOn  *domain.Day
	Comment  domain.Comment
}

// Repository — хранилище агрегатов Expense; интерфейс объявлен у потребителя (сервиса).
type Repository interface {
	// Принимает: ctx — контекст; expense — новый расход без ID.
	// Делает: сохраняет расход и присваивает ID.
	// Возвращает: сохранённый расход либо ошибку хранилища.
	Create(ctx context.Context, expense domain.Expense) (domain.Expense, error)
	// Принимает: ctx — контекст; id — идентификатор расхода.
	// Делает: читает расход.
	// Возвращает: расход либо ErrNotFound.
	Get(ctx context.Context, id domain.ExpenseID) (domain.Expense, error)
	// Принимает: ctx — контекст; filter — фильтры списка.
	// Делает: читает расходы, новые сверху (по дню, затем по ID по убыванию).
	// Возвращает: расходы (пустой срез, если ничего нет) либо ошибку хранилища.
	List(ctx context.Context, filter ListFilter) ([]domain.Expense, error)
	// Принимает: ctx — контекст.
	// Делает: читает владельцев, у которых есть расходы.
	// Возвращает: уникальные ID по возрастанию либо ошибку хранилища.
	ListUserIDs(ctx context.Context) ([]domain.UserID, error)
	// Принимает: ctx — контекст; expense — расход с ID и новыми значениями.
	// Делает: заменяет изменяемые поля расхода.
	// Возвращает: сохранённый расход либо ErrNotFound.
	Update(ctx context.Context, expense domain.Expense) (domain.Expense, error)
	// Принимает: ctx — контекст; id — идентификатор расхода.
	// Делает: удаляет расход.
	// Возвращает: nil либо ErrNotFound.
	Delete(ctx context.Context, id domain.ExpenseID) error
}
