package httpapi

import (
	"net/http"
	"strconv"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

// Принимает: w — writer ответа, второй параметр (запрос) не используется.
// Делает: отвечает 200 со статусом ok для проверки живости.
// Возвращает: ничего.
func (server *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Принимает: w — writer ответа, второй параметр (запрос) не используется.
// Делает: отдаёт список всех категорий из домена с ключом и подписью.
// Возвращает: ничего.
func (server *Server) handleListCategories(w http.ResponseWriter, _ *http.Request) {
	categories := domain.Categories()
	items := make([]categoryResponse, 0, len(categories))
	for _, category := range categories {
		items = append(items, categoryResponse{Key: string(category), Label: category.Label()})
	}
	writeJSON(w, http.StatusOK, listResponse[categoryResponse]{Items: items})
}

// Принимает: w — writer ответа, r — входящий запрос.
// Делает: получает идентификаторы пользователей из сервиса и отдаёт их списком.
// Возвращает: ничего; при ошибке сервиса пишет её через writeError.
func (server *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	userIDs, err := server.expenses.ListUserIDs(r.Context())
	if err != nil {
		server.writeError(w, r, err)
		return
	}
	items := make([]userResponse, 0, len(userIDs))
	for _, userID := range userIDs {
		items = append(items, userResponse{UserID: userID.Int64()})
	}
	writeJSON(w, http.StatusOK, listResponse[userResponse]{Items: items})
}

// Принимает: w — writer ответа, r — входящий запрос с query-фильтрами.
// Делает: разбирает фильтр из query, запрашивает расходы у сервиса и отдаёт их списком (пустой список, не null).
// Возвращает: ничего; при ошибке разбора или сервиса пишет её через writeError.
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

// Принимает: w — writer ответа, r — входящий запрос с JSON-телом.
// Делает: декодирует тело, переводит поля в доменные значения и создаёт расход через сервис; при отсутствии даты оставляет её значение по умолчанию сервису.
// Возвращает: ничего; при успехе пишет 201 и созданный расход, иначе ошибку через writeError.
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

// Принимает: w — writer ответа, r — входящий запрос с id в пути и JSON-телом.
// Делает: собирает частичный patch только из переданных полей и обновляет расход через сервис.
// Возвращает: ничего; при успехе пишет 200 и обновлённый расход, иначе ошибку через writeError.
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

// Принимает: w — writer ответа, r — входящий запрос с id в пути.
// Делает: разбирает id и удаляет расход через сервис.
// Возвращает: ничего; при успехе пишет 204 без тела, иначе ошибку через writeError.
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

// Принимает: r — входящий запрос с query-параметрами user_id, category, from, to.
// Делает: строит expense.ListFilter из непустых параметров через доменные конструкторы.
// Возвращает: заполненный фильтр; requestError при нечисловом user_id, доменную ошибку при неверных user_id, category или дате.
func parseListFilter(r *http.Request) (expense.ListFilter, error) {
	query := r.URL.Query()
	var filter expense.ListFilter

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

// Принимает: raw — необязательная строка дня формата YYYY-MM-DD.
// Делает: разбирает день, если он передан.
// Возвращает: указатель на день или nil, если raw равен nil; requestError invalid_date при неверном формате.
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

// Принимает: r — входящий запрос с path-параметром id.
// Делает: читает id расхода из пути и проверяет, что это положительное целое.
// Возвращает: id; requestError invalid_id при ошибке разбора или значении не больше нуля.
func parseID(r *http.Request) (domain.ExpenseID, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, requestError{code: "invalid_id", message: "expense id must be a positive integer"}
	}
	return domain.ExpenseID(id), nil
}

// Принимает: value — строка из query.
// Делает: превращает пустую строку в отсутствие значения.
// Возвращает: nil для пустой строки, иначе указатель на копию value.
func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
