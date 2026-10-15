// Package httpapi exposes the expense service as the REST API consumed by the family panel.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

// ExpenseService lists the use cases the panel needs; the adapter owns the contract it consumes.
type ExpenseService interface {
	Create(ctx context.Context, item domain.Expense) (domain.Expense, error)
	List(ctx context.Context, filter expense.ListFilter) ([]domain.Expense, error)
	ListUserIDs(ctx context.Context) ([]int64, error)
	Update(ctx context.Context, id int64, patch expense.Patch) (domain.Expense, error)
	Delete(ctx context.Context, id int64) error
}

// Server translates HTTP requests into expense use cases.
type Server struct {
	expenses ExpenseService
	location *time.Location
	logger   *slog.Logger
}

// NewServer wires the service with the calendar location used for date fields; nil values use defaults.
func NewServer(expenses ExpenseService, location *time.Location, logger *slog.Logger) *Server {
	if location == nil {
		location = time.UTC
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{expenses: expenses, location: location, logger: logger}
}

// Handler returns the routed API wrapped with recovery and request logging.
func (server *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.handleHealth)
	mux.HandleFunc("GET /api/categories", server.handleListCategories)
	mux.HandleFunc("GET /api/users", server.handleListUsers)
	mux.HandleFunc("GET /api/expenses", server.handleListExpenses)
	mux.HandleFunc("POST /api/expenses", server.handleCreateExpense)
	mux.HandleFunc("PATCH /api/expenses/{id}", server.handleUpdateExpense)
	mux.HandleFunc("DELETE /api/expenses/{id}", server.handleDeleteExpense)

	return server.recoverer(server.requestLogger(mux))
}
