package expense

import (
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// Patch — набор полей расхода, которые разрешено изменить (nil — поле не трогаем).
type Patch struct {
	Amount   *int64
	Category *domain.Category
	SpentAt  *time.Time
	Comment  *string
}
