package ports

import (
	"context"
	"encoding/json"
)

// Role — автор сообщения разговора.
type Role string

// Роли разговора, понятные циклу агента и любому адаптеру модели.
const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall — вызов функции, запрошенный моделью; Arguments — сырой JSON, который инструмент обязан проверить сам.
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// Message — нейтральная к провайдеру запись разговора.
type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall // заполнено у сообщений ассистента, запросивших инструменты
	ToolCallID string     // заполнено у сообщений роли tool: на какой вызов это ответ
}

// ToolSpec — описание функции для модели; Parameters — JSON Schema объекта аргументов.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// Request — один раунд запроса к модели.
type Request struct {
	Messages []Message
	Tools    []ToolSpec
}

type LLM interface {
	Complete(ctx context.Context, request Request) (Message, error)
}
