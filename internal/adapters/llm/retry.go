package llm

import (
	"context"
	"time"
)

// Принимает: ctx — контекст; backoff — пауза перед попыткой по её номеру; shouldRetry — решает, повторять ли после ошибки; call — одна попытка запроса.
// Делает: вызывает call до maxRetries+1 раз, между попытками ждёт backoff с учётом отмены контекста; останавливается на первой неповторяемой ошибке.
// Возвращает: результат успешной попытки, последнюю ошибку или ctx.Err() при отмене во время паузы.
func withRetries[T any](ctx context.Context, backoff func(attempt int) time.Duration, shouldRetry func(error) bool, call func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleep(ctx, backoff(attempt)); err != nil {
				return zero, err
			}
		}
		result, err := call()
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !shouldRetry(err) {
			break
		}
	}
	return zero, lastErr
}
