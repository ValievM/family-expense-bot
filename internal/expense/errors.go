package expense

import "errors"

// ErrNotFound hides whether an expense exists when the caller does not own it.
var ErrNotFound = errors.New("expense not found")
