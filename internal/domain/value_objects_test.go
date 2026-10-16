package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// Проверяет: NewMoney: границы и невозможность отрицательной суммы.
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

// Проверяет: точный разбор рублей десятичной строкой без float и отказ на лишних знаках и мусоре.
func TestParseRubles(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    int64
		wantErr bool
	}{
		{name: "whole rubles", text: "350", want: 35000},
		{name: "one decimal", text: "99.9", want: 9990},
		{name: "two decimals", text: "99.90", want: 9990},
		{name: "comma separator", text: "350,5", want: 35050},
		{name: "surrounding spaces", text: " 12 ", want: 1200},
		{name: "one kopeck", text: "0.01", want: 1},
		{name: "upper bound", text: "1000000000", want: 1_000_000_000 * 100},
		{name: "empty", text: "", wantErr: true},
		{name: "zero", text: "0", wantErr: true},
		{name: "zero with decimals", text: "0.00", wantErr: true},
		{name: "negative", text: "-1", wantErr: true},
		{name: "plus sign", text: "+5", wantErr: true},
		{name: "three decimals", text: "1.005", wantErr: true},
		{name: "empty fraction", text: "5.", wantErr: true},
		{name: "empty whole", text: ".5", wantErr: true},
		{name: "exponent", text: "1e2", wantErr: true},
		{name: "letters", text: "abc", wantErr: true},
		{name: "above bound", text: "1000000001", wantErr: true},
		{name: "overflowing digits", text: "99999999999999999999", wantErr: true},
		{name: "two separators", text: "1.2.3", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := domain.ParseRubles(tt.text)

			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidAmount) {
					t.Errorf("ParseRubles(%q) error = %v, want ErrInvalidAmount", tt.text, err)
				}
				return
			}
			if err != nil || money.Kopecks() != tt.want {
				t.Errorf("ParseRubles(%q) = %d, %v; want %d", tt.text, money.Kopecks(), err, tt.want)
			}
		})
	}
}

// Проверяет: NewDay и ParseDay: существующие и несуществующие даты, строгий формат разбора.
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

// Проверяет: один момент времени даёт разные дни в разных часовых поясах; nil означает UTC.
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

// Проверяет: сравнение дней и нулевое значение.
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

// Проверяет: NewUserID принимает положительные ID и отклоняет остальные.
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

// Проверяет: NewComment: обрезка пробелов и предел длины по символам.
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

// Проверяет: ParseCategory: известный ключ проходит, неизвестный отклоняется.
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
