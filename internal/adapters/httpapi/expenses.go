package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

func (server *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (server *Server) handleListCategories(w http.ResponseWriter, _ *http.Request) {
	categories := domain.Categories()
	items := make([]categoryResponse, 0, len(categories))
	for _, category := range categories {
		items = append(items, categoryResponse{Key: string(category), Label: category.Label()})
	}
	writeJSON(w, http.StatusOK, listResponse[categoryResponse]{Items: items})
}

func (server *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	userIDs, err := server.expenses.ListUserIDs(r.Context())
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	items := make([]userResponse, 0, len(userIDs))
	for _, userID := range userIDs {
		items = append(items, userResponse{UserID: userID})
	}
	writeJSON(w, http.StatusOK, listResponse[userResponse]{Items: items})
}

func (server *Server) handleListExpenses(w http.ResponseWriter, r *http.Request) {
	filter, err := server.parseListFilter(r)
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	expenses, err := server.expenses.List(r.Context(), filter)
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	items := make([]expenseResponse, 0, len(expenses))
	for _, item := range expenses {
		items = append(items, server.toResponse(item))
	}
	writeJSON(w, http.StatusOK, listResponse[expenseResponse]{Items: items})
}

func (server *Server) handleCreateExpense(w http.ResponseWriter, r *http.Request) {
	var request createExpenseRequest
	if err := decodeJSON(w, r, &request); err != nil {
		server.writeError(w, r, err)
		return
	}
	spentAt, err := server.parseOptionalDate(request.SpentAt)
	if err != nil {
		server.writeError(w, r, err)
		return
	}

	item := domain.Expense{
		UserID:   request.UserID,
		Amount:   request.Amount,
		Category: domain.Category(request.Category),
		Comment:  request.Comment,
	}
	if spentAt != nil {
		item.SpentAt = *spentAt
	}
	created, err := server.expenses.Create(r.Context(), item)
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, server.toResponse(created))
}

func (server *Server) handleUpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	var request updateExpenseRequest
	if err := decodeJSON(w, r, &request); err != nil {
		server.writeError(w, r, err)
		return
	}
	spentAt, err := server.parseOptionalDate(request.SpentAt)
	if err != nil {
		server.writeError(w, r, err)
		return
	}

	patch := expense.Patch{Amount: request.Amount, SpentAt: spentAt, Comment: request.Comment}
	if request.Category != nil {
		category := domain.Category(*request.Category)
		patch.Category = &category
	}
	updated, err := server.expenses.Update(r.Context(), id, patch)
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, server.toResponse(updated))
}

func (server *Server) handleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	if err := server.expenses.Delete(r.Context(), id); err != nil {
		server.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (server *Server) parseListFilter(r *http.Request) (expense.ListFilter, error) {
	query := r.URL.Query()
	var filter expense.ListFilter

	if raw := query.Get("user_id"); raw != "" {
		userID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return filter, requestError{code: "invalid_user", message: "user_id must be an integer"}
		}
		filter.UserID = &userID
	}
	if raw := query.Get("category"); raw != "" {
		category := domain.Category(raw)
		filter.Category = &category
	}
	from, err := server.parseOptionalDate(optionalString(query.Get("from")))
	if err != nil {
		return filter, err
	}
	to, err := server.parseOptionalDate(optionalString(query.Get("to")))
	if err != nil {
		return filter, err
	}
	filter.From, filter.To = from, to
	return filter, nil
}

func (server *Server) parseOptionalDate(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	parsed, err := domain.ParseDate(*raw, server.location)
	if err != nil {
		return nil, requestError{code: "invalid_date", message: "date must use the YYYY-MM-DD format"}
	}
	return &parsed, nil
}

func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, requestError{code: "invalid_id", message: "expense id must be a positive integer"}
	}
	return id, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
