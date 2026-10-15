package httpapi

import (
	"errors"
	"net/http"

	"github.com/ValievM/family-expense-bot/internal/domain"
	"github.com/ValievM/family-expense-bot/internal/service/expense"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// requestError reports malformed input detected before any use case runs.
type requestError struct {
	code    string
	message string
}

func (err requestError) Error() string { return err.message }

// errorRule maps a known domain or service error to its HTTP representation.
// New business errors are supported by appending a rule, without touching the handlers.
type errorRule struct {
	target error
	status int
	code   string
}

var errorRules = []errorRule{
	{domain.ErrInvalidAmount, http.StatusBadRequest, "invalid_amount"},
	{domain.ErrInvalidUser, http.StatusBadRequest, "invalid_user"},
	{domain.ErrUnknownCategory, http.StatusBadRequest, "unknown_category"},
	{domain.ErrInvalidDate, http.StatusBadRequest, "invalid_date"},
	{expense.ErrInvalidPeriod, http.StatusBadRequest, "invalid_period"},
	{expense.ErrNotFound, http.StatusNotFound, "not_found"},
}

func (server *Server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var badRequest requestError
	if errors.As(err, &badRequest) {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: errorDetail{Code: badRequest.code, Message: badRequest.message}})
		return
	}
	for _, rule := range errorRules {
		if errors.Is(err, rule.target) {
			writeJSON(w, rule.status, errorBody{Error: errorDetail{Code: rule.code, Message: rule.target.Error()}})
			return
		}
	}

	server.logger.ErrorContext(r.Context(), "unhandled request error", "method", r.Method, "path", r.URL.Path, "error", err)
	writeJSON(w, http.StatusInternalServerError, errorBody{Error: errorDetail{Code: "internal", Message: "internal server error"}})
}
