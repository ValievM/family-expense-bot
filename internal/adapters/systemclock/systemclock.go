// Package systemclock реализует порт ports.Clock на системных часах.
package systemclock

import (
	"time"

	"github.com/ValievM/family-expense-bot/internal/ports"
)

// Clock — ports.Clock на системном времени.
type Clock struct{}

var _ ports.Clock = Clock{}

// Принимает: ничего.
// Делает: читает системные часы.
// Возвращает: текущее время.
func (Clock) Now() time.Time {
	return time.Now()
}
