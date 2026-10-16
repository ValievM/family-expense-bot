// Package httpapi — REST API семейной панели поверх сервиса расходов.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/ValievM/family-expense-bot/internal/ports"
)

type Server struct {
	expenses ports.ExpenseService
	logger   *slog.Logger
}

// Принимает: expenses — сервис расходов, logger — логгер (nil заменяется на slog.Default()).
// Делает: создаёт Server.
// Возвращает: *Server.
func NewServer(expenses ports.ExpenseService, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{expenses: expenses, logger: logger}
}

// Принимает: ничего.
// Делает: регистрирует маршруты API и оборачивает их в requestLogger и recoverer.
// Возвращает: готовый http.Handler.
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
