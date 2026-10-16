package httpapi

import (
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

// expenseResponse — Amount в копейках.
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

// createExpenseRequest — Amount в копейках.
type createExpenseRequest struct {
	UserID   int64   `json:"user_id"`
	Amount   int64   `json:"amount"`
	Category string  `json:"category"`
	SpentAt  *string `json:"spent_at"`
	Comment  string  `json:"comment"`
}

// updateExpenseRequest: указатели отличают отсутствующее поле от нулевого значения.
type updateExpenseRequest struct {
	Amount   *int64  `json:"amount"`
	Category *string `json:"category"`
	SpentAt  *string `json:"spent_at"`
	Comment  *string `json:"comment"`
}

// Принимает: item — доменный расход.
// Делает: собирает JSON-представление; категорию отдаёт ключом и подписью, день — как YYYY-MM-DD.
// Возвращает: expenseResponse.
func toResponse(item domain.Expense) expenseResponse {
	return expenseResponse{
		ID:            int64(item.ID()),
		UserID:        item.Owner().Int64(),
		Amount:        item.Amount().Kopecks(),
		Category:      string(item.Category()),
		CategoryLabel: item.Category().Label(),
		SpentAt:       item.SpentOn().String(),
		Comment:       item.Comment().String(),
		CreatedAt:     item.CreatedAt(),
		UpdatedAt:     item.UpdatedAt(),
	}
}

// Принимает: ничего.
// Делает: переводит поля запроса в доменные значения валидирующими конструкторами.
// Возвращает: ports.CreateInput, доменную ошибку или requestError invalid_date.
func (request createExpenseRequest) toInput() (ports.CreateInput, error) {
	var input ports.CreateInput
	owner, err := domain.NewUserID(request.UserID)
	if err != nil {
		return input, err
	}
	amount, err := domain.NewMoney(request.Amount)
	if err != nil {
		return input, err
	}
	category, err := domain.ParseCategory(request.Category)
	if err != nil {
		return input, err
	}
	spentOn, err := parseOptionalDay(request.SpentAt)
	if err != nil {
		return input, err
	}
	comment, err := domain.NewComment(request.Comment)
	if err != nil {
		return input, err
	}
	return ports.CreateInput{Owner: owner, Amount: amount, Category: category, SpentOn: spentOn, Comment: comment}, nil
}

// Принимает: ничего.
// Делает: переводит в domain.Changes только переданные поля; пустой комментарий — явное значение.
// Возвращает: domain.Changes, доменную ошибку или requestError invalid_date.
func (request updateExpenseRequest) toChanges() (domain.Changes, error) {
	var changes domain.Changes
	if request.Amount != nil {
		amount, err := domain.NewMoney(*request.Amount)
		if err != nil {
			return changes, err
		}
		changes.Amount = &amount
	}
	if request.Category != nil {
		category, err := domain.ParseCategory(*request.Category)
		if err != nil {
			return changes, err
		}
		changes.Category = &category
	}
	spentOn, err := parseOptionalDay(request.SpentAt)
	if err != nil {
		return changes, err
	}
	changes.SpentOn = spentOn
	if request.Comment != nil {
		comment, err := domain.NewComment(*request.Comment)
		if err != nil {
			return changes, err
		}
		changes.Comment = &comment
	}
	return changes, nil
}
