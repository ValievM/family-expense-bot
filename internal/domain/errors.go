package domain

import "errors"

var (
	// ErrInvalidAmount — сумма не может быть положительным расходом.
	ErrInvalidAmount = errors.New("amount must be greater than zero")
	// ErrInvalidUser — владелец расхода не задан или его ID не положителен.
	ErrInvalidUser = errors.New("user ID must be greater than zero")
	// ErrUnknownCategory — категория не входит в закрытый набор домена.
	ErrUnknownCategory = errors.New("category is unknown")
	// ErrInvalidDate — календарная дата отсутствует или не разбирается.
	ErrInvalidDate = errors.New("date is invalid")
	// ErrCommentTooLong — комментарий длиннее допустимого.
	ErrCommentTooLong = errors.New("comment is too long")
)
