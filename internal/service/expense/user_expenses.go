package expense

import (
	"context"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// UserExpenses — набор операций, жёстко привязанных к одному владельцу.
type UserExpenses struct {
	service *Service
	userID  int64
}

// Принимает: ctx — контекст запроса; item — новый расход.
// Делает: проверяет владельца, принудительно ставит ему UserID из области и делегирует Service.Create.
// Возвращает: созданный расход или ошибку валидации/хранилища.
func (user UserExpenses) Create(ctx context.Context, item domain.Expense) (domain.Expense, error) {
	if err := user.validate(); err != nil {
		return domain.Expense{}, err
	}
	item.UserID = user.userID
	return user.service.Create(ctx, item)
}

// Принимает: ctx — контекст запроса; id — ID расхода.
// Делает: проверяет владельца и читает расход только если он принадлежит этому владельцу.
// Возвращает: расход или ErrNotFound/ErrInvalidUser.
func (user UserExpenses) Get(ctx context.Context, id int64) (domain.Expense, error) {
	if err := user.validate(); err != nil {
		return domain.Expense{}, err
	}
	return user.service.getOwned(ctx, user.userID, id)
}

// Принимает: ctx — контекст запроса; filter — фильтры списка.
// Делает: подменяет filter.UserID на владельца области (чужой фильтр игнорируется) и делегирует Service.List.
// Возвращает: список расходов владельца или ошибку.
func (user UserExpenses) List(ctx context.Context, filter ListFilter) ([]domain.Expense, error) {
	if err := user.validate(); err != nil {
		return nil, err
	}
	filter.UserID = &user.userID
	return user.service.List(ctx, filter)
}

// Принимает: ctx — контекст запроса; id — ID расхода; patch — изменяемые поля.
// Делает: проверяет владельца области и принадлежность расхода, затем применяет patch.
// Возвращает: обновлённый расход или ErrNotFound/ErrInvalidUser/ошибку валидации.
func (user UserExpenses) Update(ctx context.Context, id int64, patch Patch) (domain.Expense, error) {
	if err := user.validate(); err != nil {
		return domain.Expense{}, err
	}
	return user.service.updateForUser(ctx, user.userID, id, patch)
}

// Принимает: ctx — контекст запроса; id — ID расхода.
// Делает: проверяет владельца области и принадлежность расхода, затем удаляет его.
// Возвращает: nil при успехе или ErrNotFound/ErrInvalidUser/ошибку хранилища.
func (user UserExpenses) Delete(ctx context.Context, id int64) error {
	if err := user.validate(); err != nil {
		return err
	}
	return user.service.deleteForUser(ctx, user.userID, id)
}

// Принимает: ничего.
// Делает: проверяет, что ID владельца области положителен.
// Возвращает: nil или domain.ErrInvalidUser.
func (user UserExpenses) validate() error {
	if user.userID <= 0 {
		return domain.ErrInvalidUser
	}
	return nil
}
