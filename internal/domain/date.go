package domain

import "time"

const dateLayout = "2006-01-02"

// DateOnly returns local midnight for the instant in location; nil uses UTC.
func DateOnly(value time.Time, location *time.Location) time.Time {
	location = effectiveLocation(location)
	local := value.In(location)
	year, month, day := local.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}

// ParseDate parses a zero-padded YYYY-MM-DD date in location; nil uses UTC.
func ParseDate(value string, location *time.Location) (time.Time, error) {
	location = effectiveLocation(location)
	parsed, err := time.ParseInLocation(dateLayout, value, location)
	if err != nil || parsed.Year() < 1 || parsed.Format(dateLayout) != value {
		return time.Time{}, ErrInvalidDate
	}
	return DateOnly(parsed, location), nil
}

// FormatDate renders the local calendar day as YYYY-MM-DD; nil uses UTC.
func FormatDate(value time.Time, location *time.Location) string {
	return DateOnly(value, location).Format(dateLayout)
}

// CompareDates compares calendar days in location and returns -1, 0, or 1.
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

func isDateOnly(value time.Time) bool {
	if value.IsZero() {
		return false
	}
	hour, minute, second := value.Clock()
	return hour == 0 && minute == 0 && second == 0 && value.Nanosecond() == 0
}

func effectiveLocation(location *time.Location) *time.Location {
	if location == nil {
		return time.UTC
	}
	return location
}
