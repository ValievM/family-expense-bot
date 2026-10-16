package llm

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ValievM/family-expense-bot/internal/ports"
)

// responsesRequest — store=false и stream=true обязательны для оплаты подпиской.
type responsesRequest struct {
	Model  string          `json:"model"`
	Input  []any           `json:"input"`
	Tools  []responsesTool `json:"tools,omitempty"`
	Store  bool            `json:"store"`
	Stream bool            `json:"stream"`
}

type responsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type responsesMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responsesFunctionCall struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type responsesFunctionOutput struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

type streamEvent struct {
	Type  string `json:"type"`
	Delta string `json:"delta"`
	Item  struct {
		Type      string `json:"type"`
		CallID    string `json:"call_id"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"item"`
	Response struct {
		Error             *streamError `json:"error"`
		IncompleteDetails *struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
	} `json:"response"`
	Error   *streamError `json:"error"`
	Code    string       `json:"code"`
	Message string       `json:"message"`
	raw     string
}

type streamError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Принимает: model — slug модели; request — запрос агента.
// Делает: переводит разговор в элементы input: сообщения, вызовы функций и их результаты.
// Возвращает: тело запроса.
func newResponsesRequest(model string, request ports.Request) responsesRequest {
	wire := responsesRequest{Model: model, Input: make([]any, 0, len(request.Messages)), Store: false, Stream: true}
	for _, message := range request.Messages {
		wire.Input = append(wire.Input, toResponsesItems(message)...)
	}
	for _, spec := range request.Tools {
		wire.Tools = append(wire.Tools, responsesTool{
			Type: "function", Name: spec.Name, Description: spec.Description, Parameters: spec.Parameters,
		})
	}
	return wire
}

// Принимает: message — сообщение разговора.
// Делает: tool-сообщение даёт function_call_output, вызовы ассистента — function_call.
// Возвращает: один или несколько элементов input.
func toResponsesItems(message ports.Message) []any {
	if message.Role == ports.RoleTool {
		return []any{responsesFunctionOutput{Type: "function_call_output", CallID: message.ToolCallID, Output: message.Content}}
	}
	items := make([]any, 0, 1+len(message.ToolCalls))
	if message.Content != "" || len(message.ToolCalls) == 0 {
		items = append(items, responsesMessage{Role: string(message.Role), Content: message.Content})
	}
	for _, call := range message.ToolCalls {
		items = append(items, responsesFunctionCall{
			Type: "function_call", CallID: call.ID, Name: call.Name, Arguments: string(call.Arguments),
		})
	}
	return items
}

// Принимает: data — JSON одного события.
// Делает: разбирает событие потока.
// Возвращает: событие или ошибку; служебная строка [DONE] даёт пустое событие.
func parseEvent(data string) (streamEvent, error) {
	if data == "" || data == "[DONE]" {
		return streamEvent{}, nil
	}
	var event streamEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return streamEvent{}, fmt.Errorf("decode responses event: %w", err)
	}
	event.raw = data
	return event, nil
}

// Принимает: ничего.
// Делает: превращает событие-сбой в ошибку с кодом OpenAI.
// Возвращает: *APIError с кодом и текстом.
func (event streamEvent) failure() error {
	code, message := event.Code, event.Message
	if event.Error != nil {
		code, message = event.Error.Code, event.Error.Message
	}
	if event.Response.Error != nil {
		code, message = event.Response.Error.Code, event.Response.Error.Message
	}
	if event.Type == "response.incomplete" && event.Response.IncompleteDetails != nil {
		code = "incomplete"
		message = event.Response.IncompleteDetails.Reason
	}
	if code == "" {
		code = event.Type
	}
	if message == "" {
		// Провайдер прислал событие в неизвестной форме: оставляем кусок сырого текста, чтобы причину можно было найти по логу.
		message = truncateRaw(event.raw)
	}
	status := http.StatusOK // сбой потока приходит уже после HTTP 200 и по умолчанию не повторяется
	if isTransientStreamFailure(code) {
		status = http.StatusServiceUnavailable
	}
	return &APIError{Status: status, Message: code + ": " + message}
}

func truncateRaw(raw string) string {
	const limit = 300
	if len(raw) > limit {
		return raw[:limit] + "..."
	}
	return raw
}

// Принимает: code — код сбоя из потока.
// Делает: отличает временные сбои провайдера (пустой код события error, server_error и подобные) от окончательных отказов вроде лимита или неподдерживаемой возможности.
// Возвращает: true, если запрос стоит повторить.
func isTransientStreamFailure(code string) bool {
	switch code {
	case "error", "server_error", "internal_error", "service_unavailable", "overloaded", "subscription_sharing_usage_unavailable":
		return true
	}
	return false
}
