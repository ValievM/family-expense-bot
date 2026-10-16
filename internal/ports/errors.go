package ports

import "errors"

var (
	// ErrNotFound — также для чужого расхода: он неотличим от несуществующего.
	ErrNotFound      = errors.New("expense not found")
	ErrInvalidPeriod = errors.New("from date is after to date")
)

// ArgumentError — ошибка аргументов tool, которую модель может исправить повторным вызовом.
type ArgumentError struct {
	Message string
}

func (err ArgumentError) Error() string {
	return err.Message
}
