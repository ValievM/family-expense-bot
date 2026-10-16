package expense

import (
	"context"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// UserExpenses — область «только траты одного владельца»: ни один метод не принимает владельца аргументом.
type UserExpenses struct {
	service *Service
	owner   domain.UserID
}

// Принимает: ctx — контекст; input — данные создания без владельца.
// Делает: создаёт расход, владельцем которого становится владелец области.
// Возвращает: расход либо доменную ошибку или ошибку хранилища.
func (user UserExpenses) Create(ctx context.Context, input UserCreateInput) (domain.Expense, error) {
	if user.owner.IsZero() {
		return domain.Expense{}, domain.ErrInvalidUser
	}
	return user.service.Create(ctx, CreateInput{
		Owner: user.owner, Amount: input.Amount, Category: input.Category, SpentOn: input.SpentOn, Comment: input.Comment,
	})
}

// Принимает: ctx — контекст; id — идентификатор расхода.
// Делает: читает расход, если он принадлежит владельцу области.
// Возвращает: расход либо ErrNotFound.
func (user UserExpenses) Get(ctx context.Context, id domain.ExpenseID) (domain.Expense, error) {
	if user.owner.IsZero() {
		return domain.Expense{}, domain.ErrInvalidUser
	}
	return user.service.getOwned(ctx, user.owner, id)
}

// Принимает: ctx — контекст; filter — фильтры; поле Owner игнорируется.
// Делает: подставляет владельца области вместо любого заданного владельца и читает список.
// Возвращает: расходы владельца либо ошибку.
func (user UserExpenses) List(ctx context.Context, filter ListFilter) ([]domain.Expense, error) {
	if user.owner.IsZero() {
		return nil, domain.ErrInvalidUser
	}
	owner := user.owner
	filter.Owner = &owner
	return user.service.List(ctx, filter)
}

// Принимает: ctx — контекст; id — идентификатор; changes — изменения.
// Делает: меняет расход, только если он принадлежит владельцу области.
// Возвращает: обновлённый расход либо ErrNotFound или доменную ошибку.
func (user UserExpenses) Update(ctx context.Context, id domain.ExpenseID, changes domain.Changes) (domain.Expense, error) {
	current, err := user.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, err
	}
	return user.service.apply(ctx, current, changes)
}

// Принимает: ctx — контекст; id — идентификатор расхода.
// Делает: удаляет расход, только если он принадлежит владельцу области.
// Возвращает: nil либо ErrNotFound.
func (user UserExpenses) Delete(ctx context.Context, id domain.ExpenseID) error {
	if _, err := user.Get(ctx, id); err != nil {
		return err
	}
	return user.service.Delete(ctx, id)
}
