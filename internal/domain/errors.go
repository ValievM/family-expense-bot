package domain

import "errors"

var (
	// ErrInvalidAmount identifies amounts that cannot represent a positive expense.
	ErrInvalidAmount = errors.New("amount must be greater than zero")
	// ErrInvalidUser identifies missing or non-positive owners.
	ErrInvalidUser = errors.New("user ID must be greater than zero")
	// ErrUnknownCategory identifies categories outside the closed domain set.
	ErrUnknownCategory = errors.New("category is unknown")
	// ErrInvalidDate identifies missing or unparseable calendar dates.
	ErrInvalidDate = errors.New("date is invalid")
)
