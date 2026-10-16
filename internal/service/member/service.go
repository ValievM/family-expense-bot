// Package member — каталог членов семьи: запоминает имена из Telegram и отдаёт их панели.
package member

import (
	"context"
	"fmt"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

type Service struct {
	repository ports.MemberRepository
}

func NewService(repository ports.MemberRepository) *Service {
	return &Service{repository: repository}
}

// Принимает: ctx; userID — Telegram ID автора; name, username — данные профиля.
// Делает: создаёт участника через конструктор домена и сохраняет (обновляя имя, если оно изменилось).
// Возвращает: nil либо доменную ошибку или ошибку хранилища.
func (service *Service) Remember(ctx context.Context, userID int64, name, username string) error {
	id, err := domain.NewUserID(userID)
	if err != nil {
		return err
	}
	member, err := domain.NewMember(id, name, username)
	if err != nil {
		return err
	}
	if err := service.repository.Upsert(ctx, member); err != nil {
		return fmt.Errorf("remember member: %w", err)
	}
	return nil
}

// Принимает: ctx.
// Делает: читает всех известных участников.
// Возвращает: участников по возрастанию ID либо ошибку хранилища.
func (service *Service) List(ctx context.Context) ([]domain.Member, error) {
	members, err := service.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	return members, nil
}

var (
	_ ports.MemberDirectory = (*Service)(nil)
	_ ports.MemberCatalog   = (*Service)(nil)
)
