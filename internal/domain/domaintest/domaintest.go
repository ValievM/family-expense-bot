// Package domaintest содержит тестовые конструкторы объектов-значений: они падают в тесте при ошибке, чтобы не дублировать хелперы по пакетам.
package domaintest

import (
	"testing"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

func UserID(t testing.TB, value int64) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(value)
	if err != nil {
		t.Fatalf("NewUserID(%d): %v", value, err)
	}
	return id
}

func Money(t testing.TB, kopecks int64) domain.Money {
	t.Helper()
	money, err := domain.NewMoney(kopecks)
	if err != nil {
		t.Fatalf("NewMoney(%d): %v", kopecks, err)
	}
	return money
}

func Day(t testing.TB, value string) domain.Day {
	t.Helper()
	day, err := domain.ParseDay(value)
	if err != nil {
		t.Fatalf("ParseDay(%q): %v", value, err)
	}
	return day
}

func Comment(t testing.TB, text string) domain.Comment {
	t.Helper()
	comment, err := domain.NewComment(text)
	if err != nil {
		t.Fatalf("NewComment(%q): %v", text, err)
	}
	return comment
}
