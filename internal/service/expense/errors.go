package expense

import "errors"

var (
	// ErrNotFound — расход не найден или принадлежит другому пользователю (наличие не раскрывается).
	ErrNotFound = errors.New("expense not found")
	// ErrInvalidPeriod — дата начала фильтра позже даты окончания.
	ErrInvalidPeriod = errors.New("from date is after to date")
)
