// Package ports — интерфейсы и объекты передачи данных между слоями: сервисы и адаптеры зависят только от них и от domain.
package ports

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

type ExpenseRepository interface {
	Create(ctx context.Context, expense domain.Expense) (domain.Expense, error)
	Get(ctx context.Context, id domain.ExpenseID) (domain.Expense, error)
	List(ctx context.Context, filter ListFilter) ([]domain.Expense, error)
	ListUserIDs(ctx context.Context) ([]domain.UserID, error)
	Update(ctx context.Context, expense domain.Expense) (domain.Expense, error)
	Delete(ctx context.Context, id domain.ExpenseID) error
}

type ExpenseService interface {
	Create(ctx context.Context, input CreateInput) (domain.Expense, error)
	List(ctx context.Context, filter ListFilter) ([]domain.Expense, error)
	ListUserIDs(ctx context.Context) ([]domain.UserID, error)
	Update(ctx context.Context, id domain.ExpenseID, changes domain.Changes) (domain.Expense, error)
	Delete(ctx context.Context, id domain.ExpenseID) error
}

type UserExpenses interface {
	Create(ctx context.Context, input UserCreateInput) (domain.Expense, error)
	Get(ctx context.Context, id domain.ExpenseID) (domain.Expense, error)
	List(ctx context.Context, filter ListFilter) ([]domain.Expense, error)
	Update(ctx context.Context, id domain.ExpenseID, changes domain.Changes) (domain.Expense, error)
	Delete(ctx context.Context, id domain.ExpenseID) error
}
