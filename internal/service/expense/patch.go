package expense

import (
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// Patch contains only the expense fields that may be changed.
type Patch struct {
	Amount   *int64
	Category *domain.Category
	SpentAt  *time.Time
	Comment  *string
}
