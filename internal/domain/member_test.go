package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/domain/domaintest"
)

// Проверяет: NewMember чистит ввод (пробелы, «@»), обрезает слишком длинное и не принимает нулевой ID.
func TestNewMember(t *testing.T) {
	member, err := domain.NewMember(domaintest.UserID(t, 7), "  Алексей В.  ", " @alexey ")
	if err != nil {
		t.Fatalf("NewMember() error = %v", err)
	}
	if member.ID().Int64() != 7 || member.Name() != "Алексей В." || member.Username() != "alexey" {
		t.Errorf("member = %q / %q", member.Name(), member.Username())
	}

	long, _ := domain.NewMember(domaintest.UserID(t, 7), strings.Repeat("я", 200), strings.Repeat("a", 200))
	if len([]rune(long.Name())) != 80 || len([]rune(long.Username())) != 64 {
		t.Errorf("lengths = %d / %d, want 80 / 64", len([]rune(long.Name())), len([]rune(long.Username())))
	}

	if _, err := domain.NewMember(domain.UserID{}, "x", "y"); !errors.Is(err, domain.ErrInvalidUser) {
		t.Errorf("zero id error = %v, want ErrInvalidUser", err)
	}
}
