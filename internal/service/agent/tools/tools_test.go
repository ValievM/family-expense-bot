// Package tools_test проверяет инструменты агента на поддельной области трат без обращения к хранилищу.
package tools_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/domain/domaintest"
	"github.com/ValievM/family-expense-bot/internal/ports"
	"github.com/ValievM/family-expense-bot/internal/service/agent/tools"
)

type stubExpenses struct {
	items   []domain.Expense
	changes domain.Changes
	updated bool
}

func (stub *stubExpenses) Create(context.Context, ports.UserCreateInput) (domain.Expense, error) {
	return domain.Expense{}, nil
}

func (stub *stubExpenses) Get(_ context.Context, id domain.ExpenseID) (domain.Expense, error) {
	for _, item := range stub.items {
		if item.ID() == id {
			return item, nil
		}
	}
	return domain.Expense{}, ports.ErrNotFound
}

func (stub *stubExpenses) List(context.Context, ports.ListFilter) ([]domain.Expense, error) {
	return stub.items, nil
}

func (stub *stubExpenses) Update(ctx context.Context, id domain.ExpenseID, changes domain.Changes) (domain.Expense, error) {
	stub.changes, stub.updated = changes, true
	return stub.Get(ctx, id)
}

func (stub *stubExpenses) Delete(context.Context, domain.ExpenseID) error { return nil }

var _ ports.UserExpenses = (*stubExpenses)(nil)

func manyExpenses(t *testing.T, count int) []domain.Expense {
	t.Helper()
	owner := domaintest.UserID(t, 7)
	amount := domaintest.Money(t, 100)
	spent := domaintest.Day(t, "2026-10-16")
	items := make([]domain.Expense, 0, count)
	for index := 1; index <= count; index++ {
		item, err := domain.Restore(domain.Snapshot{
			ID: domain.ExpenseID(index), Owner: owner, Amount: amount, Category: domain.CategoryFood,
			SpentOn: spent, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, item)
	}
	return items
}

func tool(t *testing.T, name string) ports.Tool {
	t.Helper()
	found, ok := tools.Default().Get(name)
	if !ok {
		t.Fatalf("tool %q is not registered", name)
	}
	return found
}

// Проверяет: список обрезается до 50 записей, а итог и последний id считаются по всем найденным.
func TestListTruncatesButTotalsEverything(t *testing.T) {
	stub := &stubExpenses{items: manyExpenses(t, 60)}

	result, err := tool(t, "list_expenses").Execute(context.Background(), stub, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	encoded, _ := json.Marshal(result)
	var decoded struct {
		Count     int               `json:"count"`
		TotalRub  string            `json:"total_rub"`
		LatestID  int64             `json:"latest_id"`
		Truncated bool              `json:"truncated"`
		Expenses  []json.RawMessage `json:"expenses"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Count != 60 || decoded.TotalRub != "60.00" || decoded.LatestID != 60 || !decoded.Truncated || len(decoded.Expenses) != 50 {
		t.Errorf("decoded = %+v", decoded)
	}
}

// Проверяет: обновление без единого поля отклоняется до обращения к области трат.
func TestUpdateRequiresAtLeastOneField(t *testing.T) {
	stub := &stubExpenses{items: manyExpenses(t, 1)}

	_, err := tool(t, "update_expense").Execute(context.Background(), stub, json.RawMessage(`{"id":1}`))

	var argErr ports.ArgumentError
	if !errors.As(err, &argErr) || !strings.Contains(argErr.Message, "nothing to update") || stub.updated {
		t.Errorf("error = %v, updated = %v", err, stub.updated)
	}
}

// Проверяет: сборку частичного изменения: комментарий-пустая-строка очищает, неуказанные поля остаются nil.
func TestUpdateBuildsPartialChanges(t *testing.T) {
	stub := &stubExpenses{items: manyExpenses(t, 1)}

	if _, err := tool(t, "update_expense").Execute(context.Background(), stub, json.RawMessage(`{"id":1,"amount":450,"comment":""}`)); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if stub.changes.Amount == nil || stub.changes.Amount.Kopecks() != 45000 || stub.changes.Comment == nil || stub.changes.Comment.String() != "" {
		t.Errorf("changes = %+v", stub.changes)
	}
	if stub.changes.Category != nil || stub.changes.SpentOn != nil {
		t.Errorf("absent fields must stay nil: %+v", stub.changes)
	}
}

// Проверяет: таблицу некорректных аргументов: все должны давать ошибку, которую модель способна исправить.
func TestInvalidArgumentsAreRecoverable(t *testing.T) {
	tests := []struct {
		tool      string
		arguments string
	}{
		{"add_expense", `{"amount":350,"category":"food","user_id":7}`},
		{"add_expense", `{"amount":-5,"category":"food"}`},
		{"add_expense", `{"amount":5,"category":"pets"}`},
		{"add_expense", `{"amount":5,"category":"food","date":"вчера"}`},
		{"list_expenses", `{"from":"2026-13-01"}`},
		{"list_expenses", `{"category":"pets"}`},
		{"delete_expense", `{"id":0}`},
		{"delete_expense", `not json`},
	}

	for _, tt := range tests {
		t.Run(tt.tool+" "+tt.arguments, func(t *testing.T) {
			_, err := tool(t, tt.tool).Execute(context.Background(), &stubExpenses{}, json.RawMessage(tt.arguments))

			if err == nil {
				t.Fatal("Execute() error = nil, want a validation error")
			}
		})
	}
}
