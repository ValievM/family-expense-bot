package llm

import (
	"encoding/json"
	"errors"

	"github.com/ValievM/family-expense-bot/internal/ports"
)

// ErrEmptyResponse — провайдер ответил успешно, но без единого choice.
var ErrEmptyResponse = errors.New("llm provider returned no choices")

type chatRequest struct {
	Model      string        `json:"model"`
	Messages   []chatMessage `json:"messages"`
	Tools      []chatTool    `json:"tools,omitempty"`
	ToolChoice string        `json:"tool_choice,omitempty"`
}

type chatMessage struct {
	Role       string         `json:"role"`
	Content    *string        `json:"content"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatFunctionCall `json:"function"`
}

type chatFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatTool struct {
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Принимает: model — имя модели; request — запрос агента.
// Делает: переводит запрос в формат chat completions; при наличии инструментов ставит tool_choice=auto.
// Возвращает: chatRequest для провайдера.
func newChatRequest(model string, request ports.Request) chatRequest {
	wire := chatRequest{Model: model, Messages: make([]chatMessage, 0, len(request.Messages))}
	for _, message := range request.Messages {
		wire.Messages = append(wire.Messages, toChatMessage(message))
	}
	for _, spec := range request.Tools {
		wire.Tools = append(wire.Tools, chatTool{
			Type:     "function",
			Function: chatToolFunction{Name: spec.Name, Description: spec.Description, Parameters: spec.Parameters},
		})
	}
	if len(wire.Tools) > 0 {
		wire.ToolChoice = "auto"
	}
	return wire
}

// Принимает: message — сообщение агента.
// Делает: переводит его в формат провайдера.
// Возвращает: chatMessage.
func toChatMessage(message ports.Message) chatMessage {
	wire := chatMessage{Role: string(message.Role), ToolCallID: message.ToolCallID}
	// У ассистентского сообщения только с вызовами инструментов content должен быть null — так требуют многие провайдеры.
	if message.Content != "" || len(message.ToolCalls) == 0 {
		content := message.Content
		wire.Content = &content
	}
	for _, call := range message.ToolCalls {
		wire.ToolCalls = append(wire.ToolCalls, chatToolCall{
			ID:       call.ID,
			Type:     "function",
			Function: chatFunctionCall{Name: call.Name, Arguments: string(call.Arguments)},
		})
	}
	return wire
}

// Принимает: ничего.
// Делает: берёт первый choice и переводит его в сообщение ассистента с вызовами инструментов.
// Возвращает: ports.Message или ErrEmptyResponse, если choices пуст.
func (response chatResponse) assistantMessage() (ports.Message, error) {
	if len(response.Choices) == 0 {
		return ports.Message{}, ErrEmptyResponse
	}
	wire := response.Choices[0].Message
	message := ports.Message{Role: ports.RoleAssistant}
	if wire.Content != nil {
		message.Content = *wire.Content
	}
	for _, call := range wire.ToolCalls {
		message.ToolCalls = append(message.ToolCalls, ports.ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: json.RawMessage(call.Function.Arguments),
		})
	}
	return message, nil
}
