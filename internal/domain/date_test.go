package domain

import (
	"errors"
	"testing"
	"time"
)

// Принимает: t — тестовый контекст.
// Делает: проверяет, что DateOnly возвращает полночь в запрошенном часовом поясе.
// Возвращает: ничего.
func TestDateOnlyUsesRequestedLocation(t *testing.T) {
	location := time.FixedZone("UTC+03", 3*60*60)
	instant := time.Date(2026, time.January, 1, 22, 30, 0, 0, time.UTC)

	got := DateOnly(instant, location)
	want := time.Date(2026, time.January, 2, 0, 0, 0, 0, location)
	if !got.Equal(want) || got.Location() != location {
		t.Errorf("DateOnly() = %v (%v), want %v (%v)", got, got.Location(), want, location)
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет ParseDate на корректных и некорректных строках дат.
// Возвращает: ничего.
func TestParseDate(t *testing.T) {
	location := time.FixedZone("UTC+03", 3*60*60)
	tests := []struct {
		name    string
		value   string
		want    time.Time
		wantErr error
	}{
		{
			name:  "valid leap day",
			value: "2024-02-29",
			want:  time.Date(2024, time.February, 29, 0, 0, 0, 0, location),
		},
		{name: "impossible date", value: "2025-02-29", wantErr: ErrInvalidDate},
		{name: "not zero padded", value: "2024-2-9", wantErr: ErrInvalidDate},
		{name: "year zero", value: "0000-01-01", wantErr: ErrInvalidDate},
		{name: "wrong separator", value: "2024/02/29", wantErr: ErrInvalidDate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDate(tt.value, location)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ParseDate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDate() error = %v, want nil", err)
			}
			if !got.Equal(tt.want) || got.Location() != location {
				t.Errorf("ParseDate() = %v (%v), want %v (%v)", got, got.Location(), tt.want, location)
			}
		})
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет, что FormatDate использует календарный день запрошенного пояса.
// Возвращает: ничего.
func TestFormatDateUsesRequestedLocation(t *testing.T) {
	location := time.FixedZone("UTC+03", 3*60*60)
	instant := time.Date(2026, time.January, 1, 22, 30, 0, 0, time.UTC)

	if got := FormatDate(instant, location); got != "2026-01-02" {
		t.Errorf("FormatDate() = %q, want %q", got, "2026-01-02")
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет, что CompareDates сравнивает локальные календарные дни.
// Возвращает: ничего.
func TestCompareDatesComparesLocalCalendarDays(t *testing.T) {
	location := time.FixedZone("UTC+03", 3*60*60)
	day := time.Date(2026, time.January, 2, 0, 0, 0, 0, location)
	tests := []struct {
		name  string
		left  time.Time
		right time.Time
		want  int
	}{
		{
			name:  "same local date despite different times",
			left:  day,
			right: day.Add(23 * time.Hour),
		},
		{
			name:  "earlier local date",
			left:  day.Add(-time.Minute),
			right: day,
			want:  -1,
		},
		{
			name:  "later local date",
			left:  day.Add(24 * time.Hour),
			right: day,
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareDates(tt.left, tt.right, location); got != tt.want {
				t.Errorf("CompareDates() = %d, want %d", got, tt.want)
			}
		})
	}
}
