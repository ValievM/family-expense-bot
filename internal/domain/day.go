package domain

import "time"

const dayLayout = "2006-01-02"

// Day — неизменяемое значение «календарный день» без часового пояса и времени суток.
type Day struct {
	date time.Time // полночь UTC, поэтому сравнение и равенство однозначны
}

// Принимает: year, month, day — компоненты даты по григорианскому календарю.
// Делает: создаёт день и проверяет, что такой даты существует.
// Возвращает: Day либо ErrInvalidDate (нулевой год, 31 февраля и так далее).
func NewDay(year int, month time.Month, day int) (Day, error) {
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if year < 1 || date.Year() != year || date.Month() != month || date.Day() != day {
		return Day{}, ErrInvalidDate
	}
	return Day{date: date}, nil
}

// Принимает: value — строка YYYY-MM-DD с нулями.
// Делает: строго разбирает день; нестрогие формы вроде "2026-1-5" отклоняются.
// Возвращает: Day либо ErrInvalidDate.
func ParseDay(value string) (Day, error) {
	parsed, err := time.Parse(dayLayout, value)
	if err != nil || parsed.Format(dayLayout) != value {
		return Day{}, ErrInvalidDate
	}
	return NewDay(parsed.Year(), parsed.Month(), parsed.Day())
}

// Принимает: instant — момент времени; location — часовой пояс пользователя, nil означает UTC.
// Делает: определяет, какому календарному дню принадлежит момент в этом поясе («сегодня» для пользователя).
// Возвращает: Day; для некорректных годов нулевой Day.
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

// Принимает: ничего.
// Делает: проверяет, что день не задан (нулевое значение).
// Возвращает: true для нулевого Day.
func (day Day) IsZero() bool {
	return day.date.IsZero()
}

// Принимает: ничего.
// Делает: форматирует день.
// Возвращает: строку YYYY-MM-DD.
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
