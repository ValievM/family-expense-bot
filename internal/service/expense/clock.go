package expense

import "time"

// Clock — источник времени, делающий поведение сервиса детерминированным в тестах.
type Clock interface {
	// Принимает: ничего.
	// Делает: объявляет получение текущего времени.
	// Возвращает: текущее время time.Time.
	Now() time.Time
}

type systemClock struct{}

// Принимает: ничего.
// Делает: берёт текущее системное время через time.Now.
// Возвращает: текущее время time.Time.
func (systemClock) Now() time.Time {
	return time.Now()
}
