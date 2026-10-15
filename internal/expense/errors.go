package expense

import "errors"

var (
	// ErrNotFound hides whether an expense exists when the caller does not own it.
	ErrNotFound = errors.New("expense not found")
	// ErrInvalidPeriod identifies filters whose start date is after their end date.
	ErrInvalidPeriod = errors.New("from date is after to date")
)
