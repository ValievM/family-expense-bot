package expense

import "time"

// Clock supplies timestamps so service behavior remains deterministic in tests.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}
