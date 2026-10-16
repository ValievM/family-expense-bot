package member_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/member"
)

type memoryRepository struct {
	members map[int64]domain.Member
	err     error
}

func (repository *memoryRepository) Upsert(_ context.Context, item domain.Member) error {
	if repository.err != nil {
		return repository.err
	}
	repository.members[item.ID().Int64()] = item
	return nil
}

func (repository *memoryRepository) List(context.Context) ([]domain.Member, error) {
	items := make([]domain.Member, 0, len(repository.members))
	for _, item := range repository.members {
		items = append(items, item)
	}
	return items, repository.err
}

// Проверяет: Remember сохраняет участника и обновляет имя при повторном сообщении.
func TestRememberStoresAndUpdatesMember(t *testing.T) {
	repository := &memoryRepository{members: map[int64]domain.Member{}}
	service := member.NewService(repository)

	if err := service.Remember(context.Background(), 7, "Магомед", "valiev"); err != nil {
		t.Fatalf("Remember() error = %v", err)
	}
	if err := service.Remember(context.Background(), 7, "Магомед В.", "valiev"); err != nil {
		t.Fatalf("Remember() error = %v", err)
	}

	if got := repository.members[7].Name(); got != "Магомед В." || len(repository.members) != 1 {
		t.Errorf("name = %q, members = %d", got, len(repository.members))
	}
}

// Проверяет: нулевой ID отклоняется до хранилища, а ошибка хранилища оборачивается.
func TestRememberRejectsBadInput(t *testing.T) {
	repository := &memoryRepository{members: map[int64]domain.Member{}}
	service := member.NewService(repository)

	if err := service.Remember(context.Background(), 0, "x", "y"); !errors.Is(err, domain.ErrInvalidUser) {
		t.Errorf("error = %v, want ErrInvalidUser", err)
	}
	repository.err = errors.New("db down")
	if err := service.Remember(context.Background(), 7, "x", "y"); err == nil {
		t.Error("storage error was swallowed")
	}
}
