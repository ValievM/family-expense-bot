package domain

import (
	"errors"
	"testing"
)

// Принимает: t — тестовый контекст.
// Делает: проверяет, что ошибки валидации не совпадают друг с другом через errors.Is.
// Возвращает: ничего.
func TestValidationErrorsRemainDistinct(t *testing.T) {
	validationErrors := []error{
		ErrInvalidAmount,
		ErrInvalidUser,
		ErrUnknownCategory,
		ErrInvalidDate,
		ErrCommentTooLong,
	}

	for index, validationErr := range validationErrors {
		for otherIndex, otherErr := range validationErrors {
			if index != otherIndex && errors.Is(validationErr, otherErr) {
				t.Errorf("validation error %v incorrectly matches %v", validationErr, otherErr)
			}
		}
	}
}
