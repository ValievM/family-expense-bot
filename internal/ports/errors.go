package ports

import "errors"

var (
	// ErrNotFound — расход не существует либо не принадлежит владельцу; чужой расход неотличим от несуществующего.
	ErrNotFound = errors.New("expense not found")
	// ErrInvalidPeriod — начало периода позже его конца.
	ErrInvalidPeriod = errors.New("from date is after to date")
)

// ArgumentError — ошибка аргументов инструмента, которую модель может исправить повторным вызовом.
type ArgumentError struct {
	Message string
}

// Принимает: ничего.
// Делает: отдаёт текст ошибки аргументов.
// Возвращает: сообщение для модели.
func (err ArgumentError) Error() string {
	return err.Message
}
