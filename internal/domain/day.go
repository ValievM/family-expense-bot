package domain

import "time"

const dayLayout = "2006-01-02"

type Day struct {
	date time.Time // полночь UTC, поэтому сравнение и равенство однозначны
}

// Принимает: year, month, day — компоненты даты по григорианскому календарю.
// Делает: проверяет, что такая дата существует.
// Возвращает: Day либо ErrInvalidDate (нулевой год, 31 февраля).
func NewDay(year int, month time.Month, day int) (Day, error) {
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if year < 1 || date.Year() != year || date.Month() != month || date.Day() != day {
		return Day{}, ErrInvalidDate
	}
	return Day{date: date}, nil
}

// Принимает: value — строка YYYY-MM-DD.
// Делает: строго разбирает день, "2026-1-5" отклоняется.
// Возвращает: Day либо ErrInvalidDate.
func ParseDay(value string) (Day, error) {
	parsed, err := time.Parse(dayLayout, value)
	if err != nil || parsed.Format(dayLayout) != value {
		return Day{}, ErrInvalidDate
	}
	return NewDay(parsed.Year(), parsed.Month(), parsed.Day())
}

// Принимает: instant — момент времени; location — пояс пользователя, nil означает UTC.
// Делает: определяет календарный день момента в этом поясе.
// Возвращает: Day; нулевой Day для некорректного года.
func DayOf(instant time.Time, location *time.Location) Day {
	if location == nil {
		location = time.UTC
	}
	year, month, day := instant.In(location).Date()
	result, err := NewDay(year, month, day)
	if err != nil {
		return Day{}
	}
	return result
}

func (day Day) IsZero() bool {
	return day.date.IsZero()
}

func (day Day) String() string {
	return day.date.Format(dayLayout)
}

// Принимает: ничего.
// Делает: отдаёт полночь UTC этого дня для сохранения в хранилище (тип DATE).
// Возвращает: time.Time.
func (day Day) Time() time.Time {
	return day.date
}

// Принимает: other — другой день.
// Делает: сравнивает дни.
// Возвращает: -1, если день раньше, 0, если равны, 1, если позже.
func (day Day) Compare(other Day) int {
	switch {
	case day.date.Before(other.date):
		return -1
	case day.date.After(other.date):
		return 1
	default:
		return 0
	}
}
