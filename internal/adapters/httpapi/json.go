package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// maxBodyBytes — максимальный размер тела запроса (1 МиБ).
const maxBodyBytes = 1 << 20

// Принимает: w — writer ответа, status — HTTP-статус, body — значение для JSON (nil — без тела).
// Делает: выставляет Content-Type и статус, затем кодирует body; ошибка кодирования игнорируется, так как статус уже отправлен.
// Возвращает: ничего.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	// The status line is already sent, so an encoding failure cannot change the response.
	_ = json.NewEncoder(w).Encode(body)
}

// Принимает: w — writer ответа (для ограничения размера тела), r — входящий запрос, destination — указатель на целевую структуру.
// Делает: читает ровно один JSON-объект с лимитом maxBodyBytes и отвергает неизвестные поля и лишние данные после объекта.
// Возвращает: nil при успехе; requestError invalid_json при ошибке.
func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return requestError{code: "invalid_json", message: "request body must be a valid JSON object"}
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return requestError{code: "invalid_json", message: "request body must contain a single JSON object"}
	}
	return nil
}
