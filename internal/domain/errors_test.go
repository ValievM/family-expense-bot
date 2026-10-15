package domain

import (
	"errors"
	"testing"
)

func TestValidationErrorsRemainDistinct(t *testing.T) {
	validationErrors := []error{
		ErrInvalidAmount,
		ErrInvalidUser,
		ErrUnknownCategory,
		ErrInvalidDate,
	}

	for index, validationErr := range validationErrors {
		for otherIndex, otherErr := range validationErrors {
			if index != otherIndex && errors.Is(validationErr, otherErr) {
				t.Errorf("validation error %v incorrectly matches %v", validationErr, otherErr)
			}
		}
	}
}
