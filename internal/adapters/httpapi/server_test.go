// Package httpapi_test — тесты REST API семейной панели через публичный интерфейс пакета.
package httpapi_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ValievM/family-expense-bot/internal/adapters/httpapi"
	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/domain/domaintest"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

// stubService запоминает переданные адаптером аргументы и возвращает заготовленные результаты.
type stubService struct {
	created    ports.CreateInput
	filter     ports.ListFilter
	patchedID  domain.ExpenseID
	changes    domain.Changes
	deletedID  domain.ExpenseID
	listResult []domain.Expense
	userIDs    []domain.UserID
	err        error
}

// Принимает: id — идентификатор, owner — владелец, kopecks — сумма, category — категория, day — день в формате YYYY-MM-DD.
// Делает: собирает валидный доменный расход через Restore, при ошибке валит тест.
// Возвращает: domain.Expense.
func makeExpense(t *testing.T, id int64, owner int64, kopecks int64, category domain.Category, day string) domain.Expense {
	t.Helper()
	item, err := domain.Restore(domain.Snapshot{ID: domain.ExpenseID(id), Owner: domaintest.UserID(t, owner), Amount: domaintest.Money(t, kopecks), Category: category, SpentOn: domaintest.Day(t, day)})
	if err != nil {
		t.Fatal(err)
	}
	return item
}

// Принимает: второй параметр (контекст) не используется, input — данные создаваемого расхода.
// Делает: сохраняет input в stub.created; для неположительной суммы возвращает ErrInvalidAmount, как сервис.
// Возвращает: расход с ID 1 и фиксированным днём 2026-03-05, а также stub.err.
func (stub *stubService) Create(_ context.Context, input ports.CreateInput) (domain.Expense, error) {
	stub.created = input
	if !input.Amount.IsPositive() {
		return domain.Expense{}, domain.ErrInvalidAmount
	}
	day, _ := domain.ParseDay("2026-03-05")
	item, err := domain.Restore(domain.Snapshot{ID: 1, Owner: input.Owner, Amount: input.Amount, Category: input.Category, SpentOn: day, Comment: input.Comment})
	if err != nil {
		return domain.Expense{}, err
	}
	return item, stub.err
}

// Принимает: второй параметр (контекст) не используется, filter — фильтр списка.
// Делает: сохраняет filter в stub.filter.
// Возвращает: stub.listResult и stub.err.
func (stub *stubService) List(_ context.Context, filter ports.ListFilter) ([]domain.Expense, error) {
	stub.filter = filter
	return stub.listResult, stub.err
}

// Принимает: контекст не используется.
// Делает: отдаёт заготовленный список пользователей.
// Возвращает: stub.userIDs и stub.err.
func (stub *stubService) ListUserIDs(context.Context) ([]domain.UserID, error) {
	return stub.userIDs, stub.err
}

// Принимает: второй параметр (контекст) не используется, id — идентификатор, changes — изменения.
// Делает: запоминает id и changes в stub.
// Возвращает: фиксированный расход с этим id и stub.err.
func (stub *stubService) Update(_ context.Context, id domain.ExpenseID, changes domain.Changes) (domain.Expense, error) {
	stub.patchedID, stub.changes = id, changes
	owner, _ := domain.NewUserID(7)
	amount, _ := domain.NewMoney(450)
	day, _ := domain.ParseDay("2026-03-05")
	item, err := domain.Restore(domain.Snapshot{ID: id, Owner: owner, Amount: amount, Category: domain.CategoryFood, SpentOn: day})
	if err != nil {
		return domain.Expense{}, err
	}
	return item, stub.err
}

// Принимает: второй параметр (контекст) не используется, id — идентификатор.
// Делает: запоминает id в stub.deletedID.
// Возвращает: stub.err.
func (stub *stubService) Delete(_ context.Context, id domain.ExpenseID) error {
	stub.deletedID = id
	return stub.err
}

// Принимает: t — тестовый контекст, stub — подставной сервис, method, target, body — метод, URL и тело запроса.
// Делает: собирает Server поверх stub и прогоняет через его Handler запрос с httptest.
// Возвращает: записанный ответ *httptest.ResponseRecorder.
func do(t *testing.T, stub *stubService, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	handler := httpapi.NewServer(stub, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

// Принимает: t — тестовый контекст, recorder — записанный ответ.
// Делает: разбирает тело ответа как JSON в тип T, при ошибке валит тест через t.Fatalf.
// Возвращает: значение типа T.
func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		t.Fatalf("response %q is not valid JSON: %v", recorder.Body.String(), err)
	}
	return value
}

// Принимает: t — тестовый контекст.
// Делает: что POST /api/expenses передаёт поля в сервис и возвращает 201 с датой и подписью категории.
// Возвращает: ничего.
func TestCreateExpense(t *testing.T) {
	stub := &stubService{}
	recorder := do(t, stub, http.MethodPost, "/api/expenses",
		`{"user_id":7,"amount":35000,"category":"food","spent_at":"2026-03-05","comment":"coffee"}`)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", recorder.Code, recorder.Body)
	}
	if stub.created.Owner.Int64() != 7 || stub.created.Amount.Kopecks() != 35000 || stub.created.Category != domain.CategoryFood || stub.created.Comment.String() != "coffee" {
		t.Errorf("service received %+v", stub.created)
	}
	body := decode[map[string]any](t, recorder)
	if body["spent_at"] != "2026-03-05" || body["category_label"] != "еда" {
		t.Errorf("response = %v", body)
	}
}

// Принимает: t — тестовый контекст.
// Делает: что без spent_at адаптер передаёт nil-день, чтобы сервис подставил сегодняшний.
// Возвращает: ничего.
func TestCreateExpenseWithoutDateLeavesDefaultingToService(t *testing.T) {
	stub := &stubService{}
	recorder := do(t, stub, http.MethodPost, "/api/expenses", `{"user_id":7,"amount":100,"category":"other"}`)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", recorder.Code, recorder.Body)
	}
	if stub.created.SpentOn != nil {
		t.Errorf("SpentOn = %v, want nil so the service applies today", stub.created.SpentOn)
	}
}

// Принимает: t — тестовый контекст.
// Делает: что query-параметры user_id, category, from и to попадают в фильтр сервиса, а ответ содержит элементы.
// Возвращает: ничего.
func TestListExpensesPassesFilters(t *testing.T) {
	stub := &stubService{listResult: []domain.Expense{makeExpense(t, 3, 7, 100, domain.CategoryHealth, "2026-03-01")}}
	recorder := do(t, stub, http.MethodGet, "/api/expenses?user_id=7&category=health&from=2026-03-01&to=2026-03-31", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", recorder.Code, recorder.Body)
	}
	if stub.filter.Owner == nil || stub.filter.Owner.Int64() != 7 {
		t.Errorf("user filter = %v, want 7", stub.filter.Owner)
	}
	if stub.filter.Category == nil || *stub.filter.Category != domain.CategoryHealth {
		t.Errorf("category filter = %v, want health", stub.filter.Category)
	}
	if stub.filter.From == nil || stub.filter.To == nil {
		t.Errorf("period filter = (%v, %v), want both set", stub.filter.From, stub.filter.To)
	}
	items := decode[struct{ Items []map[string]any }](t, recorder).Items
	if len(items) != 1 || items[0]["id"].(float64) != 3 {
		t.Errorf("items = %v", items)
	}
}

// Принимает: t — тестовый контекст.
// Делает: что пустой результат сериализуется как items:[], а не null.
// Возвращает: ничего.
func TestListExpensesReturnsEmptyArrayNotNull(t *testing.T) {
	recorder := do(t, &stubService{}, http.MethodGet, "/api/expenses", "")

	if !strings.Contains(recorder.Body.String(), `"items":[]`) {
		t.Errorf("body = %s, want an empty items array", recorder.Body)
	}
}

// Принимает: t — тестовый контекст.
// Делает: что PATCH строит patch только из переданных полей, включая явную пустую строку комментария.
// Возвращает: ничего.
func TestUpdateExpenseBuildsPartialPatch(t *testing.T) {
	stub := &stubService{}
	recorder := do(t, stub, http.MethodPatch, "/api/expenses/12", `{"amount":450,"comment":""}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", recorder.Code, recorder.Body)
	}
	if stub.patchedID != 12 {
		t.Errorf("id = %d, want 12", stub.patchedID)
	}
	if stub.changes.Amount == nil || stub.changes.Amount.Kopecks() != 450 {
		t.Errorf("amount change = %v, want 450", stub.changes.Amount)
	}
	if stub.changes.Comment == nil || stub.changes.Comment.String() != "" {
		t.Errorf("comment change = %v, want explicit empty string", stub.changes.Comment)
	}
	if stub.changes.Category != nil || stub.changes.SpentOn != nil {
		t.Errorf("absent fields must stay nil: %+v", stub.changes)
	}
}

// Принимает: t — тестовый контекст.
// Делает: что DELETE возвращает 204 и передаёт в сервис верный id.
// Возвращает: ничего.
func TestDeleteExpense(t *testing.T) {
	stub := &stubService{}
	recorder := do(t, stub, http.MethodDelete, "/api/expenses/9", "")

	if recorder.Code != http.StatusNoContent || stub.deletedID != 9 {
		t.Errorf("status = %d, deleted id = %d, want 204 and 9", recorder.Code, stub.deletedID)
	}
}

// Принимает: t — тестовый контекст.
// Делает: форму ответов /api/users и /api/categories.
// Возвращает: ничего.
func TestListUsersAndCategories(t *testing.T) {
	users := do(t, &stubService{userIDs: []domain.UserID{domaintest.UserID(t, 1), domaintest.UserID(t, 2)}}, http.MethodGet, "/api/users", "")
	if got := strings.TrimSpace(users.Body.String()); got != `{"items":[{"user_id":1},{"user_id":2}]}` {
		t.Errorf("users body = %s", got)
	}

	categories := decode[struct{ Items []map[string]string }](t, do(t, &stubService{}, http.MethodGet, "/api/categories", "")).Items
	if len(categories) != len(domain.Categories()) || categories[0]["key"] != "food" || categories[0]["label"] != "еда" {
		t.Errorf("categories = %v", categories)
	}
}

// Принимает: t — тестовый контекст.
// Делает: таблицей, что ошибки сервиса и разбора запроса превращаются в нужные статус и код, а внутренние детали не утекают в 500.
// Возвращает: ничего.
func TestErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		body       string
		serviceErr error
		wantStatus int
		wantCode   string
	}{
		{"domain validation", http.MethodPost, "/api/expenses", `{"user_id":1,"amount":0,"category":"food"}`, nil, 400, "invalid_amount"},
		{"unknown category", http.MethodPost, "/api/expenses", `{"user_id":1,"amount":1,"category":"x"}`, domain.ErrUnknownCategory, 400, "unknown_category"},
		{"not found on update", http.MethodPatch, "/api/expenses/5", `{"amount":1}`, ports.ErrNotFound, 404, "not_found"},
		{"not found on delete", http.MethodDelete, "/api/expenses/5", "", ports.ErrNotFound, 404, "not_found"},
		{"invalid period", http.MethodGet, "/api/expenses", "", ports.ErrInvalidPeriod, 400, "invalid_period"},
		{"unexpected failure hides details", http.MethodGet, "/api/users", "", io.ErrUnexpectedEOF, 500, "internal"},
		{"malformed json", http.MethodPost, "/api/expenses", `{"user_id":`, nil, 400, "invalid_json"},
		{"unknown json field", http.MethodPost, "/api/expenses", `{"user_id":1,"amount":1,"category":"food","owner":2}`, nil, 400, "invalid_json"},
		{"trailing json", http.MethodPost, "/api/expenses", `{"user_id":1} {}`, nil, 400, "invalid_json"},
		{"bad date", http.MethodPost, "/api/expenses", `{"user_id":1,"amount":1,"category":"food","spent_at":"05.03.2026"}`, nil, 400, "invalid_date"},
		{"comment too long", http.MethodPost, "/api/expenses", `{"user_id":1,"amount":1,"category":"food","comment":"` + strings.Repeat("я", 501) + `"}`, nil, 400, "comment_too_long"},
		{"comment too long on update", http.MethodPatch, "/api/expenses/5", `{"comment":"` + strings.Repeat("я", 501) + `"}`, nil, 400, "comment_too_long"},
		{"zero user on create", http.MethodPost, "/api/expenses", `{"user_id":0,"amount":1,"category":"food"}`, nil, 400, "invalid_user"},
		{"negative user on create", http.MethodPost, "/api/expenses", `{"user_id":-5,"amount":1,"category":"food"}`, nil, 400, "invalid_user"},
		{"zero user filter", http.MethodGet, "/api/expenses?user_id=0", "", nil, 400, "invalid_user"},
		{"negative amount", http.MethodPost, "/api/expenses", `{"user_id":1,"amount":-1,"category":"food"}`, nil, 400, "invalid_amount"},
		{"unknown category filter", http.MethodGet, "/api/expenses?category=x", "", nil, 400, "unknown_category"},
		{"bad id", http.MethodDelete, "/api/expenses/abc", "", nil, 400, "invalid_id"},
		{"bad user filter", http.MethodGet, "/api/expenses?user_id=abc", "", nil, 400, "invalid_user"},
		{"bad period date", http.MethodGet, "/api/expenses?from=yesterday", "", nil, 400, "invalid_date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := do(t, &stubService{err: tt.serviceErr}, tt.method, tt.target, tt.body)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", recorder.Code, tt.wantStatus, recorder.Body)
			}
			body := decode[struct {
				Error struct{ Code, Message string }
			}](t, recorder)
			if body.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Error.Code, tt.wantCode)
			}
			if tt.wantStatus == 500 && strings.Contains(body.Error.Message, "EOF") {
				t.Errorf("internal error leaked: %q", body.Error.Message)
			}
		})
	}
}

// Принимает: t — тестовый контекст.
// Делает: что /healthz отвечает 200.
// Возвращает: ничего.
func TestHealth(t *testing.T) {
	recorder := do(t, &stubService{}, http.MethodGet, "/healthz", "")

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", recorder.Code)
	}
}
