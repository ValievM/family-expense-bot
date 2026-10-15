package domain

import "time"

// dateLayout — формат даты YYYY-MM-DD в стиле time.Format.
const dateLayout = "2006-01-02"

// Принимает: value — момент времени; location — часовой пояс (nil означает UTC).
// Делает: переводит момент в указанный пояс и обнуляет время суток.
// Возвращает: локальную полночь того же календарного дня в location.
func DateOnly(value time.Time, location *time.Location) time.Time {
	location = effectiveLocation(location)
	local := value.In(location)
	year, month, day := local.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}

// Принимает: value — строка даты YYYY-MM-DD с ведущими нулями; location — часовой пояс (nil означает UTC).
// Делает: разбирает строку в location и отвергает невозможные даты, год меньше 1 и формат без нулевого заполнения.
// Возвращает: локальную полночь указанной даты или ErrInvalidDate при неверном значении.
func ParseDate(value string, location *time.Location) (time.Time, error) {
	location = effectiveLocation(location)
	parsed, err := time.ParseInLocation(dateLayout, value, location)
	if err != nil || parsed.Year() < 1 || parsed.Format(dateLayout) != value {
		return time.Time{}, ErrInvalidDate
	}
	return DateOnly(parsed, location), nil
}

// Принимает: value — момент времени; location — часовой пояс (nil означает UTC).
// Делает: приводит момент к локальному календарному дню в location и форматирует его.
// Возвращает: строку вида YYYY-MM-DD.
func FormatDate(value time.Time, location *time.Location) string {
	return DateOnly(value, location).Format(dateLayout)
}

// Принимает: left, right — сравниваемые моменты; location — часовой пояс (nil означает UTC).
// Делает: сравнивает только календарные дни в location, игнорируя время суток.
// Возвращает: -1, если left раньше right; 1, если позже; 0, если дни совпадают.
func CompareDates(left, right time.Time, location *time.Location) int {
	location = effectiveLocation(location)
	leftDay := DateOnly(left, location)
	rightDay := DateOnly(right, location)
	switch {
	case leftDay.Before(rightDay):
		return -1
	case leftDay.After(rightDay):
		return 1
	default:
		return 0
	}
}

// Принимает: value — проверяемое время.
// Делает: проверяет, что значение не нулевое и время суток равно 00:00:00.000000000.
// Возвращает: true, если значение является датой без времени, иначе false.
func isDateOnly(value time.Time) bool {
	if value.IsZero() {
		return false
	}
	hour, minute, second := value.Clock()
	return hour == 0 && minute == 0 && second == 0 && value.Nanosecond() == 0
}

// Принимает: location — часовой пояс или nil.
// Делает: подставляет UTC вместо nil.
// Возвращает: переданный пояс либо time.UTC.
func effectiveLocation(location *time.Location) *time.Location {
	if location == nil {
		return time.UTC
	}
	return location
}
