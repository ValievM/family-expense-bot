// Package systemclock реализует порт ports.Clock на системных часах.
package systemclock

import (
	"time"

	"github.com/ValievM/family-expense-bot/internal/ports"
)

type Clock struct{}

func (Clock) Now() time.Time {
	return time.Now()
}

var _ ports.Clock = Clock{}
