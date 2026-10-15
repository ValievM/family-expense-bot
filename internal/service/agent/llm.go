package agent

import (
	"context"
	"encoding/json"
)

// Role identifies the author of a conversation message.
type Role string

// Conversation roles understood by the agent loop and every LLM adapter.
const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall is a function invocation requested by the model. Arguments is raw JSON the tool must validate.
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// Message is one provider-neutral conversation entry.
type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall // set on assistant messages that request tools
	ToolCallID string     // set on tool messages to answer a specific call
}

// ToolSpec advertises a callable function to the model; Parameters is a JSON Schema object.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// Request is a single completion round.
type Request struct {
	Messages []Message
	Tools    []ToolSpec
}

// LLM is the only capability the agent needs from a language model provider.
type LLM interface {
	Complete(ctx context.Context, request Request) (Message, error)
}
