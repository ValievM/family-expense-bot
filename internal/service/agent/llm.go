package agent

import (
	"context"
	"encoding/json"
)

// Role — автор сообщения в диалоге.
type Role string

// Роли диалога, понятные циклу агента и всем LLM-адаптерам.
const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall — вызов функции, запрошенный моделью; Arguments — сырой JSON, который инструмент обязан проверить.
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// Message — одно сообщение диалога в формате, не зависящем от провайдера.
type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall // set on assistant messages that request tools
	ToolCallID string     // set on tool messages to answer a specific call
}

// ToolSpec — описание функции для модели; Parameters — объект JSON Schema.
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

// LLM — единственное, что агенту нужно от провайдера языковой модели.
type LLM interface {
	// Принимает: ctx — контекст запроса; request — сообщения диалога и доступные инструменты.
	// Делает: отправляет диалог модели и получает следующее сообщение ассистента.
	// Возвращает: сообщение ассистента (текст и/или вызовы инструментов) или ошибку провайдера.
	Complete(ctx context.Context, request Request) (Message, error)
}
