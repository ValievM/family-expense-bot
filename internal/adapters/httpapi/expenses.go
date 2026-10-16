package httpapi

import (
	"cmp"
	"context"
	"net/http"
	"slices"
	"strconv"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

// Принимает: w — ответ.
// Делает: отвечает 200 со статусом ok.
// Возвращает: ничего.
// Принимает: w, r — ответ и запрос.
// Делает: проверяет доступность базы за healthTimeout.
// Возвращает: 200 {"status":"ok"} либо 503 {"status":"unavailable"}, если база не отвечает.
func (server *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
	defer cancel()
	if err := server.health.Ping(ctx); err != nil {
		server.logger.WarnContext(r.Context(), "health check failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Принимает: w — ответ.
// Делает: отдаёт все категории домена с ключом и подписью.
// Возвращает: ничего.
func (server *Server) handleListCategories(w http.ResponseWriter, _ *http.Request) {
	categories := domain.Categories()
	items := make([]categoryResponse, 0, len(categories))
	for _, category := range categories {
		items = append(items, categoryResponse{Key: string(category), Label: category.Label()})
	}
	writeJSON(w, http.StatusOK, listResponse[categoryResponse]{Items: items})
}

// Принимает: w, r — ответ и запрос.
// Делает: объединяет владельцев расходов и известных участников без дублей; имя и ник берёт из справочника, пустые — "".
// Возвращает: ничего; ошибку любого источника пишет через writeError.
func (server *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	userIDs, err := server.expenses.ListUserIDs(r.Context())
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	members, err := server.members.List(r.Context())
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	byID := make(map[int64]userResponse, len(userIDs)+len(members))
	for _, userID := range userIDs {
		byID[userID.Int64()] = userResponse{UserID: userID.Int64()}
	}
	for _, member := range members {
		byID[member.ID().Int64()] = userResponse{UserID: member.ID().Int64(), Name: member.Name(), Username: member.Username()}
	}
	items := make([]userResponse, 0, len(byID))
	for _, item := range byID {
		items = append(items, item)
	}
	slices.SortFunc(items, func(a, b userResponse) int { return cmp.Compare(a.UserID, b.UserID) })
	writeJSON(w, http.StatusOK, listResponse[userResponse]{Items: items})
}

// Принимает: w, r — ответ и запрос с query-фильтрами.
// Делает: разбирает фильтр и отдаёт расходы списком (пустой список, не null).
// Возвращает: ничего; ошибки пишет через writeError.
func (server *Server) handleListExpenses(w http.ResponseWriter, r *http.Request) {
	filter, err := parseListFilter(r)
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
		items = append(items, toResponse(item))
	}
	writeJSON(w, http.StatusOK, listResponse[expenseResponse]{Items: items})
}

// Принимает: w, r — ответ и запрос с JSON-телом.
// Делает: создаёт расход через сервис; без даты её подставляет сервис.
// Возвращает: ничего; 201 с расходом или ошибка через writeError.
func (server *Server) handleCreateExpense(w http.ResponseWriter, r *http.Request) {
	var request createExpenseRequest
	if err := decodeJSON(w, r, &request); err != nil {
		server.writeError(w, r, err)
		return
	}
	input, err := request.toInput()
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	created, err := server.expenses.Create(r.Context(), input)
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(created))
}

// Принимает: w, r — ответ и запрос с id в пути и JSON-телом.
// Делает: строит patch из переданных полей и обновляет расход через сервис.
// Возвращает: ничего; 200 с расходом или ошибка через writeError.
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
	changes, err := request.toChanges()
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	updated, err := server.expenses.Update(r.Context(), id, changes)
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(updated))
}

// Принимает: w, r — ответ и запрос с id в пути.
// Делает: удаляет расход через сервис.
// Возвращает: ничего; 204 без тела или ошибка через writeError.
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

// Принимает: r — запрос с query user_id, category, from, to.
// Делает: строит ports.ListFilter из непустых параметров доменными конструкторами.
// Возвращает: фильтр; requestError при нечисловом user_id, доменную ошибку при неверных значениях.
func parseListFilter(r *http.Request) (ports.ListFilter, error) {
	query := r.URL.Query()
	var filter ports.ListFilter

	if raw := query.Get("user_id"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return filter, requestError{code: "invalid_user", message: "user_id must be an integer"}
		}
		owner, err := domain.NewUserID(value)
		if err != nil {
			return filter, err
		}
		filter.Owner = &owner
	}
	if raw := query.Get("category"); raw != "" {
		category, err := domain.ParseCategory(raw)
		if err != nil {
			return filter, err
		}
		filter.Category = &category
	}
	from, err := parseOptionalDay(optionalString(query.Get("from")))
	if err != nil {
		return filter, err
	}
	to, err := parseOptionalDay(optionalString(query.Get("to")))
	if err != nil {
		return filter, err
	}
	filter.From, filter.To = from, to
	return filter, nil
}

// Принимает: raw — необязательный день YYYY-MM-DD.
// Делает: разбирает день, если он передан.
// Возвращает: день или nil для nil; requestError invalid_date при неверном формате.
func parseOptionalDay(raw *string) (*domain.Day, error) {
	if raw == nil {
		return nil, nil
	}
	day, err := domain.ParseDay(*raw)
	if err != nil {
		return nil, requestError{code: "invalid_date", message: "date must use the YYYY-MM-DD format"}
	}
	return &day, nil
}

// Принимает: r — запрос с path-параметром id.
// Делает: читает id расхода и проверяет, что он положительный.
// Возвращает: id или requestError invalid_id.
func parseID(r *http.Request) (domain.ExpenseID, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, requestError{code: "invalid_id", message: "expense id must be a positive integer"}
	}
	return domain.ExpenseID(id), nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
