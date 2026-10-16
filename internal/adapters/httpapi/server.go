// Package httpapi — REST API семейной панели поверх сервиса расходов.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

// ExpenseService — use case'ы, нужные панели; контракт объявляет потребитель (адаптер).
type ExpenseService interface {
	// Принимает: ctx — контекст запроса, input — данные нового расхода.
	// Делает: создаёт расход; пустая дата означает «сегодня».
	// Возвращает: сохранённый расход или ошибку.
	Create(ctx context.Context, input expense.CreateInput) (domain.Expense, error)
	// Принимает: ctx — контекст запроса, filter — фильтр списка.
	// Делает: выбирает расходы по фильтру.
	// Возвращает: список расходов или ошибку.
	List(ctx context.Context, filter expense.ListFilter) ([]domain.Expense, error)
	// Принимает: ctx — контекст запроса.
	// Делает: выбирает идентификаторы пользователей.
	// Возвращает: список идентификаторов или ошибку.
	ListUserIDs(ctx context.Context) ([]domain.UserID, error)
	// Принимает: ctx — контекст запроса, id — идентификатор расхода, changes — изменяемые поля.
	// Делает: частично обновляет расход.
	// Возвращает: обновлённый расход или ошибку (в том числе expense.ErrNotFound).
	Update(ctx context.Context, id domain.ExpenseID, changes domain.Changes) (domain.Expense, error)
	// Принимает: ctx — контекст запроса, id — идентификатор расхода.
	// Делает: удаляет расход.
	// Возвращает: nil при успехе или ошибку (в том числе expense.ErrNotFound).
	Delete(ctx context.Context, id domain.ExpenseID) error
}

// Server переводит HTTP-запросы в use case'ы расходов.
type Server struct {
	expenses ExpenseService
	logger   *slog.Logger
}

// Принимает: expenses — сервис расходов, logger — логгер.
// Делает: создаёт Server; nil logger заменяется на slog.Default().
// Возвращает: указатель на Server.
func NewServer(expenses ExpenseService, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{expenses: expenses, logger: logger}
}

// Принимает: ничего.
// Делает: регистрирует маршруты API в ServeMux и оборачивает их в requestLogger и recoverer.
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
