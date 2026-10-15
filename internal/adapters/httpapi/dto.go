package httpapi

import (
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// expenseResponse is the wire form of an expense; Amount is expressed in kopecks.
type expenseResponse struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Amount        int64     `json:"amount"`
	Category      string    `json:"category"`
	CategoryLabel string    `json:"category_label"`
	SpentAt       string    `json:"spent_at"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type categoryResponse struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type userResponse struct {
	UserID int64 `json:"user_id"`
}

type listResponse[T any] struct {
	Items []T `json:"items"`
}

type createExpenseRequest struct {
	UserID   int64   `json:"user_id"`
	Amount   int64   `json:"amount"`
	Category string  `json:"category"`
	SpentAt  *string `json:"spent_at"`
	Comment  string  `json:"comment"`
}

// updateExpenseRequest uses pointers so an absent field differs from an explicit zero value.
type updateExpenseRequest struct {
	Amount   *int64  `json:"amount"`
	Category *string `json:"category"`
	SpentAt  *string `json:"spent_at"`
	Comment  *string `json:"comment"`
}

func (server *Server) toResponse(item domain.Expense) expenseResponse {
	return expenseResponse{
		ID:            item.ID,
		UserID:        item.UserID,
		Amount:        item.Amount,
		Category:      string(item.Category),
		CategoryLabel: item.Category.Label(),
		SpentAt:       domain.FormatDate(item.SpentAt, server.location),
		Comment:       item.Comment,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}
