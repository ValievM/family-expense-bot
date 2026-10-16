package domain

import "errors"

var (
	ErrInvalidAmount   = errors.New("amount must be a positive number of rubles with at most two decimals")
	ErrInvalidUser     = errors.New("user ID must be greater than zero")
	ErrUnknownCategory = errors.New("category is unknown")
	ErrInvalidDate     = errors.New("date is invalid")
	ErrCommentTooLong  = errors.New("comment is too long")
)
