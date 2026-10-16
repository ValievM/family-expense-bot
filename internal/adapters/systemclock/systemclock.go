// Package systemclock реализует порт ports.Clock на системных часах.
package systemclock

import (
	"time"

	"github.com/ValievM/family-expense-bot/internal/ports"
)

type Clock struct{}

var _ ports.Clock = Clock{}

func (Clock) Now() time.Time {
	return time.Now()
}
