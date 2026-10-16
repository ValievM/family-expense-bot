package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

var created = time.Date(2026, time.January, 2, 10, 0, 0, 0, time.UTC)

func validInput(t *testing.T, kopecks int64) domain.NewExpenseInput {
	t.Helper()
	owner, err := domain.NewUserID(7)
	if err != nil {
		t.Fatal(err)
	}
	amount, err := domain.NewMoney(kopecks)
	if err != nil {
		t.Fatal(err)
	}
	day, err := domain.ParseDay("2026-01-02")
	if err != nil {
		t.Fatal(err)
	}
	comment, _ := domain.NewComment("кофе")
	return domain.NewExpenseInput{Owner: owner, Amount: amount, Category: domain.CategoryFood, SpentOn: day, Comment: comment, At: created}
}

// Проверяет: NewExpense принимает корректные данные и ставит время создания и изменения.
func TestNewExpenseAcceptsValidInput(t *testing.T) {
	expense, err := domain.NewExpense(validInput(t, 35000))
	if err != nil {
		t.Fatalf("NewExpense() error = %v", err)
	}

	if expense.Amount().Kopecks() != 35000 || expense.Category() != domain.CategoryFood || expense.SpentOn().String() != "2026-01-02" ||
		expense.Comment().String() != "кофе" || !expense.CreatedAt().Equal(created) || !expense.UpdatedAt().Equal(created) || expense.ID() != 0 {
		t.Errorf("expense = %+v", expense)
	}
}

// Проверяет: NewExpense защищает каждый инвариант агрегата.
func TestNewExpenseRejectsBrokenInvariants(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.NewExpenseInput)
		want   error
	}{
		{"zero owner", func(in *domain.NewExpenseInput) { in.Owner = domain.UserID{} }, domain.ErrInvalidUser},
		{"zero amount", func(in *domain.NewExpenseInput) { in.Amount = domain.Money{} }, domain.ErrInvalidAmount},
		{"unknown category", func(in *domain.NewExpenseInput) { in.Category = "pets" }, domain.ErrUnknownCategory},
		{"empty category", func(in *domain.NewExpenseInput) { in.Category = "" }, domain.ErrUnknownCategory},
		{"missing day", func(in *domain.NewExpenseInput) { in.SpentOn = domain.Day{} }, domain.ErrInvalidDate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validInput(t, 100)
			tt.mutate(&input)

			if _, err := domain.NewExpense(input); !errors.Is(err, tt.want) {
				t.Errorf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

// Проверяет: Apply меняет только заданные поля и updatedAt, исходный расход не затрагивается.
func TestApplyChangesOnlyGivenFieldsAndKeepsOriginal(t *testing.T) {
	original, _ := domain.NewExpense(validInput(t, 35000))
	original = original.WithID(42)
	amount, _ := domain.NewMoney(45000)
	later := created.Add(time.Hour)

	updated, err := original.Apply(domain.Changes{Amount: &amount}, later)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if updated.Amount().Kopecks() != 45000 || updated.Category() != domain.CategoryFood || updated.ID() != 42 ||
		!updated.UpdatedAt().Equal(later) || !updated.CreatedAt().Equal(created) {
		t.Errorf("updated = %+v", updated)
	}
	if original.Amount().Kopecks() != 35000 || !original.UpdatedAt().Equal(created) {
		t.Errorf("original was mutated: %+v", original)
	}
}

// Проверяет: Apply отвергает изменение, ломающее инвариант.
func TestApplyRejectsInvalidResult(t *testing.T) {
	original, _ := domain.NewExpense(validInput(t, 100))
	zero := domain.Money{}
	bad := domain.Category("pets")

	if _, err := original.Apply(domain.Changes{Amount: &zero}, created); !errors.Is(err, domain.ErrInvalidAmount) {
		t.Errorf("zero amount error = %v", err)
	}
	if _, err := original.Apply(domain.Changes{Category: &bad}, created); !errors.Is(err, domain.ErrUnknownCategory) {
		t.Errorf("unknown category error = %v", err)
	}
}

// Проверяет: OwnedBy различает владельца и чужого.
func TestOwnedBy(t *testing.T) {
	expense, _ := domain.NewExpense(validInput(t, 100))
	same, _ := domain.NewUserID(7)
	other, _ := domain.NewUserID(8)

	if !expense.OwnedBy(same) || expense.OwnedBy(other) {
		t.Error("OwnedBy() is wrong")
	}
}

// Проверяет: Restore принимает корректный снимок и отклоняет повреждённый.
func TestRestoreValidatesStoredState(t *testing.T) {
	input := validInput(t, 100)
	snapshot := domain.Snapshot{
		ID: 5, Owner: input.Owner, Amount: input.Amount, Category: input.Category, SpentOn: input.SpentOn,
		Comment: input.Comment, CreatedAt: created, UpdatedAt: created,
	}

	restored, err := domain.Restore(snapshot)
	if err != nil || restored.ID() != 5 {
		t.Fatalf("Restore() = %+v, %v", restored, err)
	}
	snapshot.Amount = domain.Money{}
	if _, err := domain.Restore(snapshot); !errors.Is(err, domain.ErrInvalidAmount) {
		t.Errorf("corrupted snapshot error = %v, want ErrInvalidAmount", err)
	}
}
