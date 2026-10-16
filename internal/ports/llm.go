package ports

import (
	"context"
	"encoding/json"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall — Arguments — сырой JSON, который tool обязан проверить сам.
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall // заполнено у сообщений ассистента, запросивших инструменты
	ToolCallID string     // заполнено у сообщений роли tool: на какой вызов это ответ
}

// ToolSpec — Parameters — JSON Schema аргументов.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

type Request struct {
	Messages []Message
	Tools    []ToolSpec
}

type LLM interface {
	Complete(ctx context.Context, request Request) (Message, error)
}
