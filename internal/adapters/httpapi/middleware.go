package httpapi

import (
	"net/http"
	"time"
)

// statusRecorder запоминает HTTP-статус ответа для логирования.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// Принимает: status — HTTP-статус ответа.
// Делает: сохраняет статус и передаёт его вложенному ResponseWriter.
// Возвращает: ничего.
func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

// Принимает: next — следующий обработчик в цепочке.
// Делает: вызывает next и после него логирует метод, путь, статус и длительность запроса.
// Возвращает: обёрнутый http.Handler.
func (server *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		server.logger.InfoContext(r.Context(), "request handled",
			"method", r.Method, "path", r.URL.Path, "status", recorder.status, "duration", time.Since(started))
	})
}

// Принимает: next — следующий обработчик в цепочке.
// Делает: вызывает next, перехватывает panic, логирует её и отвечает 500 internal.
// Возвращает: обёрнутый http.Handler.
func (server *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				server.logger.ErrorContext(r.Context(), "panic in handler", "panic", recovered)
				writeJSON(w, http.StatusInternalServerError, errorBody{Error: errorDetail{Code: "internal", Message: "internal server error"}})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
