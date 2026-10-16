package llm

import (
	"context"
	"time"
)

// Принимает: ctx; backoff — пауза по номеру попытки; shouldRetry — повторять ли; call — одна попытка.
// Делает: зовёт call до maxRetries+1 раз с паузами backoff; останавливается на первой неповторяемой ошибке.
// Возвращает: результат успешной попытки, последнюю ошибку или ctx.Err() при отмене паузы.
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
