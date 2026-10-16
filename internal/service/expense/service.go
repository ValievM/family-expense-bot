// Package expense — сценарии работы с расходами (слой приложения).
package expense

import (
	"context"
	"fmt"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

var (
	_ ports.ExpenseService = (*Service)(nil)
	_ ports.UserExpenses   = UserExpenses{}
)

// Service координирует сценарии над агрегатом; инварианты проверяет домен.
type Service struct {
	repository ports.ExpenseRepository
	clock      ports.Clock
	location   *time.Location
}

// Принимает: repository, clock (обязательны); location — пояс «сегодня», nil = UTC.
// Делает: собирает сервис.
// Возвращает: готовый *Service.
func NewService(repository ports.ExpenseRepository, clock ports.Clock, location *time.Location) *Service {
	if location == nil {
		location = time.UTC
	}
	return &Service{repository: repository, clock: clock, location: location}
}

// Принимает: owner — владелец данных.
// Делает: сужает сервис до трат одного владельца.
// Возвращает: UserExpenses без возможности выбрать другого владельца.
func (service *Service) ForUser(owner domain.UserID) UserExpenses {
	return UserExpenses{service: service, owner: owner}
}

// Принимает: ctx; input — данные создания.
// Делает: подставляет «сегодня» вместо пустого дня, создаёт агрегат и сохраняет.
// Возвращает: сохранённый расход либо доменную ошибку или ошибку хранилища.
func (service *Service) Create(ctx context.Context, input ports.CreateInput) (domain.Expense, error) {
	now := service.clock.Now().UTC()
	// Пропущенный день означает «сегодня»; правило живёт здесь, чтобы REST и агент вели себя одинаково.
	spentOn := domain.DayOf(now, service.location)
	if input.SpentOn != nil {
		spentOn = *input.SpentOn
	}

	expense, err := domain.NewExpense(domain.NewExpenseInput{
		Owner: input.Owner, Amount: input.Amount, Category: input.Category,
		SpentOn: spentOn, Comment: input.Comment, At: now,
	})
	if err != nil {
		return domain.Expense{}, err
	}
	created, err := service.repository.Create(ctx, expense)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("create expense: %w", err)
	}
	return created, nil
}

// Принимает: ctx; id — идентификатор.
// Делает: читает расход без ограничения по владельцу (панель главы семьи).
// Возвращает: расход либо ErrNotFound.
func (service *Service) Get(ctx context.Context, id domain.ExpenseID) (domain.Expense, error) {
	expense, err := service.repository.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("get expense: %w", err)
	}
	return expense, nil
}

// Принимает: ctx; filter — фильтры.
// Делает: проверяет период и категорию, читает список.
// Возвращает: расходы либо ErrInvalidPeriod, ErrUnknownCategory или ошибку хранилища.
func (service *Service) List(ctx context.Context, filter ports.ListFilter) ([]domain.Expense, error) {
	if filter.Category != nil && !filter.Category.IsValid() {
		return nil, domain.ErrUnknownCategory
	}
	if filter.From != nil && filter.To != nil && filter.From.Compare(*filter.To) > 0 {
		return nil, ports.ErrInvalidPeriod
	}
	expenses, err := service.repository.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	return expenses, nil
}

func (service *Service) ListUserIDs(ctx context.Context) ([]domain.UserID, error) {
	userIDs, err := service.repository.ListUserIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list expense owners: %w", err)
	}
	return userIDs, nil
}

// Принимает: ctx; id; changes — изменяемые поля.
// Делает: применяет изменения методом агрегата и сохраняет.
// Возвращает: обновлённый расход либо ErrNotFound, доменную ошибку или ошибку хранилища.
func (service *Service) Update(ctx context.Context, id domain.ExpenseID, changes domain.Changes) (domain.Expense, error) {
	current, err := service.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, err
	}
	return service.apply(ctx, current, changes)
}

// Принимает: ctx; id.
// Делает: удаляет расход без ограничения по владельцу.
// Возвращает: nil либо ErrNotFound.
func (service *Service) Delete(ctx context.Context, id domain.ExpenseID) error {
	if err := service.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	return nil
}

// Принимает: ctx; owner; id.
// Делает: читает расход и проверяет принадлежность; чужой неотличим от несуществующего.
// Возвращает: расход либо ErrNotFound.
func (service *Service) getOwned(ctx context.Context, owner domain.UserID, id domain.ExpenseID) (domain.Expense, error) {
	expense, err := service.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, err
	}
	if !expense.OwnedBy(owner) {
		return domain.Expense{}, ports.ErrNotFound
	}
	return expense, nil
}

// Принимает: ctx; current — текущий расход; changes.
// Делает: применяет изменения с текущим временем и сохраняет.
// Возвращает: сохранённый расход либо доменную ошибку или ошибку хранилища.
func (service *Service) apply(ctx context.Context, current domain.Expense, changes domain.Changes) (domain.Expense, error) {
	updated, err := current.Apply(changes, service.clock.Now().UTC())
	if err != nil {
		return domain.Expense{}, err
	}
	saved, err := service.repository.Update(ctx, updated)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("update expense: %w", err)
	}
	return saved, nil
}
