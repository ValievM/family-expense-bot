package domain_test

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// Принимает: t — тестовый контекст.
// Делает: проверяет конструктор NewMoney: границы и невозможность отрицательной суммы.
// Возвращает: ничего.
func TestNewMoney(t *testing.T) {
	tests := []struct {
		name    string
		kopecks int64
		wantErr bool
	}{
		{"zero is a valid amount of money", 0, false},
		{"positive", 35000, false},
		{"upper bound", 1_000_000_000 * 100, false},
		{"above bound", 1_000_000_000*100 + 1, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := domain.NewMoney(tt.kopecks)

			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidAmount) {
					t.Errorf("error = %v, want ErrInvalidAmount", err)
				}
				return
			}
			if err != nil || money.Kopecks() != tt.kopecks {
				t.Errorf("NewMoney(%d) = %v, %v", tt.kopecks, money, err)
			}
		})
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет перевод рублей в копейки: округление и недопустимые значения.
// Возвращает: ничего.
func TestMoneyFromRubles(t *testing.T) {
	tests := []struct {
		name    string
		rubles  float64
		want    int64
		wantErr bool
	}{
		{name: "whole rubles", rubles: 350, want: 35000},
		{name: "fraction", rubles: 99.9, want: 9990},
		{name: "float noise is rounded", rubles: 0.1 + 0.2, want: 30},
		{name: "one kopeck", rubles: 0.01, want: 1},
		{name: "below one kopeck", rubles: 0.004, wantErr: true},
		{name: "zero", rubles: 0, wantErr: true},
		{name: "negative", rubles: -1, wantErr: true},
		{name: "NaN", rubles: math.NaN(), wantErr: true},
		{name: "infinity", rubles: math.Inf(1), wantErr: true},
		{name: "too large", rubles: 2e9, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := domain.MoneyFromRubles(tt.rubles)

			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidAmount) {
					t.Errorf("error = %v, want ErrInvalidAmount", err)
				}
				return
			}
			if err != nil || money.Kopecks() != tt.want {
				t.Errorf("MoneyFromRubles(%v) = %d, %v; want %d", tt.rubles, money.Kopecks(), err, tt.want)
			}
		})
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет форматирование, сложение и положительность суммы.
// Возвращает: ничего.
func TestMoneyStringAddAndPositivity(t *testing.T) {
	cases := map[int64]string{0: "0.00", 5: "0.05", 35000: "350.00", 9990: "99.90", 123456789: "1234567.89"}
	for kopecks, want := range cases {
		money, _ := domain.NewMoney(kopecks)
		if money.String() != want {
			t.Errorf("Money(%d).String() = %q, want %q", kopecks, money.String(), want)
		}
	}

	left, _ := domain.NewMoney(150)
	right, _ := domain.NewMoney(250)
	if sum := left.Add(right); sum.Kopecks() != 400 || left.Kopecks() != 150 {
		t.Errorf("Add() = %d, operands must stay unchanged (left = %d)", sum.Kopecks(), left.Kopecks())
	}
	if (domain.Money{}).IsPositive() || !left.IsPositive() {
		t.Error("IsPositive() is wrong for zero or positive money")
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет конструкторы Day: существующие и несуществующие даты, строгий формат разбора.
// Возвращает: ничего.
func TestDayConstructors(t *testing.T) {
	valid := []string{"2026-01-02", "2024-02-29", "0001-01-01"}
	for _, value := range valid {
		day, err := domain.ParseDay(value)
		if err != nil || day.String() != value {
			t.Errorf("ParseDay(%q) = %v, %v", value, day, err)
		}
	}

	invalid := []string{"", "2026-1-2", "2026-02-30", "2025-02-29", "2026-13-01", "2026-00-10", "05.03.2026", "2026-01-02T00:00:00Z", "0000-01-01", " 2026-01-02"}
	for _, value := range invalid {
		if _, err := domain.ParseDay(value); !errors.Is(err, domain.ErrInvalidDate) {
			t.Errorf("ParseDay(%q) error = %v, want ErrInvalidDate", value, err)
		}
	}
	if _, err := domain.NewDay(2026, time.February, 31); !errors.Is(err, domain.ErrInvalidDate) {
		t.Errorf("NewDay(2026, 2, 31) error = %v, want ErrInvalidDate", err)
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет DayOf: один момент времени даёт разные дни в разных часовых поясах; nil означает UTC.
// Возвращает: ничего.
func TestDayOfUsesTheGivenLocation(t *testing.T) {
	moscow := time.FixedZone("MSK", 3*60*60)
	instant := time.Date(2026, time.January, 1, 22, 30, 0, 0, time.UTC)

	if got := domain.DayOf(instant, moscow).String(); got != "2026-01-02" {
		t.Errorf("DayOf(Moscow) = %s, want 2026-01-02", got)
	}
	if got := domain.DayOf(instant, nil).String(); got != "2026-01-01" {
		t.Errorf("DayOf(nil) = %s, want 2026-01-01", got)
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет сравнение дней и нулевое значение.
// Возвращает: ничего.
func TestDayCompareAndZero(t *testing.T) {
	earlier, _ := domain.ParseDay("2026-01-01")
	sameDay, _ := domain.ParseDay("2026-01-01")
	later, _ := domain.ParseDay("2026-01-02")

	if earlier.Compare(later) != -1 || later.Compare(earlier) != 1 || earlier.Compare(sameDay) != 0 {
		t.Error("Compare() returns wrong ordering")
	}
	if !(domain.Day{}).IsZero() || earlier.IsZero() {
		t.Error("IsZero() is wrong")
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет конструктор UserID.
// Возвращает: ничего.
func TestNewUserID(t *testing.T) {
	for _, bad := range []int64{0, -5} {
		if _, err := domain.NewUserID(bad); !errors.Is(err, domain.ErrInvalidUser) {
			t.Errorf("NewUserID(%d) error = %v, want ErrInvalidUser", bad, err)
		}
	}
	id, err := domain.NewUserID(6612214276)
	if err != nil || id.Int64() != 6612214276 || id.IsZero() {
		t.Errorf("NewUserID() = %v, %v", id, err)
	}
	if !(domain.UserID{}).IsZero() {
		t.Error("zero UserID must report IsZero")
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет конструктор Comment: обрезка пробелов и предел длины по символам.
// Возвращает: ничего.
func TestNewComment(t *testing.T) {
	comment, err := domain.NewComment("  кофе  ")
	if err != nil || comment.String() != "кофе" {
		t.Errorf("NewComment() = %q, %v", comment.String(), err)
	}
	if _, err := domain.NewComment(strings.Repeat("я", 500)); err != nil {
		t.Errorf("500 runes must be accepted, got %v", err)
	}
	if _, err := domain.NewComment(strings.Repeat("я", 501)); !errors.Is(err, domain.ErrCommentTooLong) {
		t.Errorf("501 runes error = %v, want ErrCommentTooLong", err)
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет ParseCategory: известный ключ проходит, неизвестный отклоняется.
// Возвращает: ничего.
func TestParseCategory(t *testing.T) {
	if category, err := domain.ParseCategory("food"); err != nil || category != domain.CategoryFood {
		t.Errorf("ParseCategory(food) = %v, %v", category, err)
	}
	for _, bad := range []string{"", "Food", "pets"} {
		if _, err := domain.ParseCategory(bad); !errors.Is(err, domain.ErrUnknownCategory) {
			t.Errorf("ParseCategory(%q) error = %v, want ErrUnknownCategory", bad, err)
		}
	}
}
