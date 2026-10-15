// Package expense — координирует сценарии работы с расходами без зависимости от инфраструктуры.
package expense

import (
	"context"
	"fmt"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// Service — координирует валидацию, владение и сохранение расходов.
type Service struct {
	repository Repository
	clock      Clock
	location   *time.Location
}

// Принимает: repository — хранилище; clock — источник времени; location — часовой пояс для календарных дат.
// Делает: собирает сервис; при nil для clock берёт системное время, при nil для location — UTC.
// Возвращает: указатель на готовый *Service.
func NewService(repository Repository, clock Clock, location *time.Location) *Service {
	if clock == nil {
		clock = systemClock{}
	}
	if location == nil {
		location = time.UTC
	}
	return &Service{repository: repository, clock: clock, location: location}
}

// Принимает: userID — ID владельца.
// Делает: создаёт обёртку, методы которой работают только с расходами этого владельца.
// Возвращает: UserExpenses, привязанный к userID.
func (service *Service) ForUser(userID int64) UserExpenses {
	return UserExpenses{service: service, userID: userID}
}

// Принимает: ctx — контекст запроса; item — новый расход.
// Делает: проставляет дату «сегодня», если SpentAt пустой, нормализует её до даты, валидирует, ставит CreatedAt/UpdatedAt из clock и сохраняет в репозитории.
// Возвращает: созданный расход или ошибку валидации/хранилища.
func (service *Service) Create(ctx context.Context, item domain.Expense) (domain.Expense, error) {
	now := service.clock.Now().UTC()
	if item.SpentAt.IsZero() {
		// An omitted date means "today", which every caller (REST and agent) shares.
		item.SpentAt = now
	}
	item.SpentAt = service.normalizeDate(item.SpentAt)
	if err := item.Validate(); err != nil {
		return domain.Expense{}, err
	}

	item.CreatedAt = now
	item.UpdatedAt = now
	created, err := service.repository.Create(ctx, item)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("create expense: %w", err)
	}
	return created, nil
}

// Принимает: ctx — контекст запроса; id — ID расхода.
// Делает: читает расход из репозитория без проверки владельца (для веб-панели).
// Возвращает: расход или обёрнутую ошибку репозитория (в т.ч. ErrNotFound).
func (service *Service) Get(ctx context.Context, id int64) (domain.Expense, error) {
	item, err := service.repository.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("get expense: %w", err)
	}
	return item, nil
}

// Принимает: ctx — контекст запроса; filter — фильтры списка.
// Делает: нормализует и проверяет фильтр (владелец, категория, период), затем запрашивает список в репозитории.
// Возвращает: список расходов или ошибку фильтра/хранилища.
func (service *Service) List(ctx context.Context, filter ListFilter) ([]domain.Expense, error) {
	filter, err := service.normalizeListFilter(filter)
	if err != nil {
		return nil, err
	}

	items, err := service.repository.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	return items, nil
}

// Принимает: ctx — контекст запроса.
// Делает: запрашивает у репозитория уникальных владельцев расходов.
// Возвращает: ID пользователей или обёрнутую ошибку хранилища.
func (service *Service) ListUserIDs(ctx context.Context) ([]int64, error) {
	userIDs, err := service.repository.ListUserIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list expense owners: %w", err)
	}
	return userIDs, nil
}

// Принимает: ctx — контекст запроса; id — ID расхода; patch — изменяемые поля.
// Делает: читает расход без проверки владельца и применяет к нему patch с единой валидацией результата.
// Возвращает: обновлённый расход или ошибку чтения/валидации/хранилища.
func (service *Service) Update(ctx context.Context, id int64, patch Patch) (domain.Expense, error) {
	current, err := service.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, err
	}
	return service.updateCurrent(ctx, current, patch)
}

// Принимает: ctx — контекст запроса; id — ID расхода.
// Делает: удаляет расход без проверки владельца (для веб-панели).
// Возвращает: nil при успехе или обёрнутую ошибку репозитория.
func (service *Service) Delete(ctx context.Context, id int64) error {
	if err := service.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	return nil
}

// Принимает: ctx — контекст запроса; userID — ID владельца; id — ID расхода.
// Делает: проверяет userID и читает расход, сверяя владельца.
// Возвращает: расход; domain.ErrInvalidUser при userID <= 0; ErrNotFound, если расход чужой или не найден.
func (service *Service) getOwned(ctx context.Context, userID, id int64) (domain.Expense, error) {
	if userID <= 0 {
		return domain.Expense{}, domain.ErrInvalidUser
	}
	item, err := service.Get(ctx, id)
	if err != nil {
		return domain.Expense{}, err
	}
	if item.UserID != userID {
		return domain.Expense{}, ErrNotFound
	}
	return item, nil
}

// Принимает: ctx — контекст запроса; userID — ID владельца; id — ID расхода; patch — изменяемые поля.
// Делает: сначала проверяет владение расходом, затем применяет patch.
// Возвращает: обновлённый расход или ошибку владения/валидации/хранилища.
func (service *Service) updateForUser(ctx context.Context, userID, id int64, patch Patch) (domain.Expense, error) {
	current, err := service.getOwned(ctx, userID, id)
	if err != nil {
		return domain.Expense{}, err
	}
	return service.updateCurrent(ctx, current, patch)
}

// Принимает: ctx — контекст запроса; current — текущий расход; patch — изменяемые поля.
// Делает: переносит заданные поля patch в расход, нормализует дату, валидирует, ставит UpdatedAt из clock и сохраняет.
// Возвращает: обновлённый расход или ошибку валидации/хранилища.
func (service *Service) updateCurrent(ctx context.Context, current domain.Expense, patch Patch) (domain.Expense, error) {
	if patch.Amount != nil {
		current.Amount = *patch.Amount
	}
	if patch.Category != nil {
		current.Category = *patch.Category
	}
	if patch.SpentAt != nil {
		current.SpentAt = *patch.SpentAt
	}
	if patch.Comment != nil {
		current.Comment = *patch.Comment
	}
	current.SpentAt = service.normalizeDate(current.SpentAt)
	if err := current.Validate(); err != nil {
		return domain.Expense{}, err
	}

	current.UpdatedAt = service.clock.Now().UTC()
	updated, err := service.repository.Update(ctx, current)
	if err != nil {
		return domain.Expense{}, fmt.Errorf("update expense: %w", err)
	}
	return updated, nil
}

// Принимает: ctx — контекст запроса; userID — ID владельца; id — ID расхода.
// Делает: проверяет владение расходом и только затем удаляет его.
// Возвращает: nil при успехе или ошибку владения/хранилища.
func (service *Service) deleteForUser(ctx context.Context, userID, id int64) error {
	if _, err := service.getOwned(ctx, userID, id); err != nil {
		return err
	}
	if err := service.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	return nil
}

// Принимает: filter — исходный фильтр списка.
// Делает: проверяет владельца, категорию и даты, приводит From/To к календарной дате в часовом поясе сервиса и сверяет порядок границ.
// Возвращает: нормализованный фильтр или ErrInvalidUser/ErrUnknownCategory/ErrInvalidDate/ErrInvalidPeriod.
func (service *Service) normalizeListFilter(filter ListFilter) (ListFilter, error) {
	if filter.UserID != nil && *filter.UserID <= 0 {
		return ListFilter{}, domain.ErrInvalidUser
	}
	if filter.Category != nil && !filter.Category.IsValid() {
		return ListFilter{}, domain.ErrUnknownCategory
	}
	if filter.From != nil {
		if filter.From.IsZero() {
			return ListFilter{}, domain.ErrInvalidDate
		}
		from := domain.DateOnly(*filter.From, service.location)
		filter.From = &from
	}
	if filter.To != nil {
		if filter.To.IsZero() {
			return ListFilter{}, domain.ErrInvalidDate
		}
		to := domain.DateOnly(*filter.To, service.location)
		filter.To = &to
	}
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return ListFilter{}, ErrInvalidPeriod
	}
	return filter, nil
}

// Принимает: value — момент времени.
// Делает: приводит время к календарной дате в часовом поясе сервиса; нулевое значение не трогает.
// Возвращает: нормализованное время.
func (service *Service) normalizeDate(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return domain.DateOnly(value, service.location)
}
