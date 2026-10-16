package llm

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errTransient = errors.New("transient")

// Принимает: error — ошибка попытки.
// Делает: считает любую ошибку повторяемой.
// Возвращает: всегда true.
func alwaysRetry(error) bool { return true }

// Принимает: int — номер попытки (не используется).
// Делает: задаёт нулевую паузу.
// Возвращает: ноль.
func zeroBackoff(int) time.Duration { return 0 }

// Проверяет: после временных сбоев возвращается успешный результат.
func TestWithRetriesSucceedsAfterRetries(t *testing.T) {
	calls := 0
	got, err := withRetries(context.Background(), zeroBackoff, alwaysRetry, func() (string, error) {
		calls++
		if calls < 3 {
			return "", errTransient
		}
		return "ok", nil
	})
	if err != nil || got != "ok" || calls != 3 {
		t.Errorf("got = %q, err = %v, calls = %d, want ok after 3 calls", got, err, calls)
	}
}

// Проверяет: неповторяемая ошибка сразу останавливает цикл.
func TestWithRetriesStopsOnNonRetryable(t *testing.T) {
	calls := 0
	_, err := withRetries(context.Background(), zeroBackoff, func(error) bool { return false }, func() (int, error) {
		calls++
		return 0, errTransient
	})
	if !errors.Is(err, errTransient) || calls != 1 {
		t.Errorf("err = %v, calls = %d, want one call", err, calls)
	}
}

// Проверяет: отмена контекста во время паузы прерывает повторы.
func TestWithRetriesStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, err := withRetries(ctx, func(int) time.Duration {
		cancel()
		return time.Hour
	}, alwaysRetry, func() (int, error) {
		calls++
		return 0, errTransient
	})
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Errorf("err = %v, calls = %d, want context.Canceled after one call", err, calls)
	}
}

// Проверяет: при исчерпании попыток возвращается последняя ошибка.
func TestWithRetriesReturnsLastErrorWhenExhausted(t *testing.T) {
	calls := 0
	_, err := withRetries(context.Background(), zeroBackoff, alwaysRetry, func() (int, error) {
		calls++
		return 0, errors.New("attempt " + string(rune('0'+calls)))
	})
	if err == nil || err.Error() != "attempt 3" || calls != maxRetries+1 {
		t.Errorf("err = %v, calls = %d, want last error after %d calls", err, calls, maxRetries+1)
	}
}
