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
	"time"

	"github.com/ValievM/family-expense-bot/internal/adapters/httpapi"
	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

// stubService records the arguments the adapter passes and returns canned results.
type stubService struct {
	created    domain.Expense
	filter     expense.ListFilter
	patchedID  int64
	patch      expense.Patch
	deletedID  int64
	listResult []domain.Expense
	userIDs    []int64
	err        error
}

func (stub *stubService) Create(_ context.Context, item domain.Expense) (domain.Expense, error) {
	stub.created = item
	item.ID = 1
	item.SpentAt = time.Date(2026, time.March, 5, 0, 0, 0, 0, time.UTC)
	return item, stub.err
}

func (stub *stubService) List(_ context.Context, filter expense.ListFilter) ([]domain.Expense, error) {
	stub.filter = filter
	return stub.listResult, stub.err
}

func (stub *stubService) ListUserIDs(context.Context) ([]int64, error) { return stub.userIDs, stub.err }

func (stub *stubService) Update(_ context.Context, id int64, patch expense.Patch) (domain.Expense, error) {
	stub.patchedID, stub.patch = id, patch
	return domain.Expense{ID: id, UserID: 7, Amount: 450, Category: domain.CategoryFood, SpentAt: time.Date(2026, time.March, 5, 0, 0, 0, 0, time.UTC)}, stub.err
}

func (stub *stubService) Delete(_ context.Context, id int64) error {
	stub.deletedID = id
	return stub.err
}

func do(t *testing.T, stub *stubService, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	handler := httpapi.NewServer(stub, time.UTC, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		t.Fatalf("response %q is not valid JSON: %v", recorder.Body.String(), err)
	}
	return value
}

func TestCreateExpense(t *testing.T) {
	stub := &stubService{}
	recorder := do(t, stub, http.MethodPost, "/api/expenses",
		`{"user_id":7,"amount":35000,"category":"food","spent_at":"2026-03-05","comment":"coffee"}`)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", recorder.Code, recorder.Body)
	}
	if stub.created.UserID != 7 || stub.created.Amount != 35000 || stub.created.Category != domain.CategoryFood || stub.created.Comment != "coffee" {
		t.Errorf("service received %+v", stub.created)
	}
	body := decode[map[string]any](t, recorder)
	if body["spent_at"] != "2026-03-05" || body["category_label"] != "еда" {
		t.Errorf("response = %v", body)
	}
}

func TestCreateExpenseWithoutDateLeavesDefaultingToService(t *testing.T) {
	stub := &stubService{}
	recorder := do(t, stub, http.MethodPost, "/api/expenses", `{"user_id":7,"amount":100,"category":"other"}`)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", recorder.Code, recorder.Body)
	}
	if !stub.created.SpentAt.IsZero() {
		t.Errorf("SpentAt = %v, want zero so the service applies today", stub.created.SpentAt)
	}
}

func TestListExpensesPassesFilters(t *testing.T) {
	stub := &stubService{listResult: []domain.Expense{{ID: 3, UserID: 7, Amount: 100, Category: domain.CategoryHealth, SpentAt: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)}}}
	recorder := do(t, stub, http.MethodGet, "/api/expenses?user_id=7&category=health&from=2026-03-01&to=2026-03-31", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", recorder.Code, recorder.Body)
	}
	if stub.filter.UserID == nil || *stub.filter.UserID != 7 {
		t.Errorf("user filter = %v, want 7", stub.filter.UserID)
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

func TestListExpensesReturnsEmptyArrayNotNull(t *testing.T) {
	recorder := do(t, &stubService{}, http.MethodGet, "/api/expenses", "")

	if !strings.Contains(recorder.Body.String(), `"items":[]`) {
		t.Errorf("body = %s, want an empty items array", recorder.Body)
	}
}

func TestUpdateExpenseBuildsPartialPatch(t *testing.T) {
	stub := &stubService{}
	recorder := do(t, stub, http.MethodPatch, "/api/expenses/12", `{"amount":450,"comment":""}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", recorder.Code, recorder.Body)
	}
	if stub.patchedID != 12 {
		t.Errorf("id = %d, want 12", stub.patchedID)
	}
	if stub.patch.Amount == nil || *stub.patch.Amount != 450 {
		t.Errorf("amount patch = %v, want 450", stub.patch.Amount)
	}
	if stub.patch.Comment == nil || *stub.patch.Comment != "" {
		t.Errorf("comment patch = %v, want explicit empty string", stub.patch.Comment)
	}
	if stub.patch.Category != nil || stub.patch.SpentAt != nil {
		t.Errorf("absent fields must stay nil: %+v", stub.patch)
	}
}

func TestDeleteExpense(t *testing.T) {
	stub := &stubService{}
	recorder := do(t, stub, http.MethodDelete, "/api/expenses/9", "")

	if recorder.Code != http.StatusNoContent || stub.deletedID != 9 {
		t.Errorf("status = %d, deleted id = %d, want 204 and 9", recorder.Code, stub.deletedID)
	}
}

func TestListUsersAndCategories(t *testing.T) {
	users := do(t, &stubService{userIDs: []int64{1, 2}}, http.MethodGet, "/api/users", "")
	if got := strings.TrimSpace(users.Body.String()); got != `{"items":[{"user_id":1},{"user_id":2}]}` {
		t.Errorf("users body = %s", got)
	}

	categories := decode[struct{ Items []map[string]string }](t, do(t, &stubService{}, http.MethodGet, "/api/categories", "")).Items
	if len(categories) != len(domain.Categories()) || categories[0]["key"] != "food" || categories[0]["label"] != "еда" {
		t.Errorf("categories = %v", categories)
	}
}

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
		{"domain validation", http.MethodPost, "/api/expenses", `{"user_id":1,"amount":0,"category":"food"}`, domain.ErrInvalidAmount, 400, "invalid_amount"},
		{"unknown category", http.MethodPost, "/api/expenses", `{"user_id":1,"amount":1,"category":"x"}`, domain.ErrUnknownCategory, 400, "unknown_category"},
		{"not found on update", http.MethodPatch, "/api/expenses/5", `{"amount":1}`, expense.ErrNotFound, 404, "not_found"},
		{"not found on delete", http.MethodDelete, "/api/expenses/5", "", expense.ErrNotFound, 404, "not_found"},
		{"invalid period", http.MethodGet, "/api/expenses", "", expense.ErrInvalidPeriod, 400, "invalid_period"},
		{"unexpected failure hides details", http.MethodGet, "/api/users", "", io.ErrUnexpectedEOF, 500, "internal"},
		{"malformed json", http.MethodPost, "/api/expenses", `{"user_id":`, nil, 400, "invalid_json"},
		{"unknown json field", http.MethodPost, "/api/expenses", `{"user_id":1,"amount":1,"category":"food","owner":2}`, nil, 400, "invalid_json"},
		{"trailing json", http.MethodPost, "/api/expenses", `{"user_id":1} {}`, nil, 400, "invalid_json"},
		{"bad date", http.MethodPost, "/api/expenses", `{"user_id":1,"amount":1,"category":"food","spent_at":"05.03.2026"}`, nil, 400, "invalid_date"},
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

func TestHealth(t *testing.T) {
	recorder := do(t, &stubService{}, http.MethodGet, "/healthz", "")

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", recorder.Code)
	}
}
