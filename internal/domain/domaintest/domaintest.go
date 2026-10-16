// Package domaintest содержит тестовые конструкторы объектов-значений: они падают в тесте при ошибке, чтобы не дублировать хелперы по пакетам.
package domaintest

import (
	"testing"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// Принимает: t — тестовый контекст; value — Telegram ID.
// Делает: создаёт UserID через конструктор и останавливает тест при ошибке.
// Возвращает: UserID.
func UserID(t testing.TB, value int64) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(value)
	if err != nil {
		t.Fatalf("NewUserID(%d): %v", value, err)
	}
	return id
}

// Принимает: t — тестовый контекст; kopecks — сумма в копейках.
// Делает: создаёт Money через конструктор и останавливает тест при ошибке.
// Возвращает: Money.
func Money(t testing.TB, kopecks int64) domain.Money {
	t.Helper()
	money, err := domain.NewMoney(kopecks)
	if err != nil {
		t.Fatalf("NewMoney(%d): %v", kopecks, err)
	}
	return money
}

// Принимает: t — тестовый контекст; value — день YYYY-MM-DD.
// Делает: разбирает день через конструктор и останавливает тест при ошибке.
// Возвращает: Day.
func Day(t testing.TB, value string) domain.Day {
	t.Helper()
	day, err := domain.ParseDay(value)
	if err != nil {
		t.Fatalf("ParseDay(%q): %v", value, err)
	}
	return day
}

// Принимает: t — тестовый контекст; text — текст комментария.
// Делает: создаёт Comment через конструктор и останавливает тест при ошибке.
// Возвращает: Comment.
func Comment(t testing.TB, text string) domain.Comment {
	t.Helper()
	comment, err := domain.NewComment(text)
	if err != nil {
		t.Fatalf("NewComment(%q): %v", text, err)
	}
	return comment
}
