package ports

import (
	"context"
	"encoding/json"
)

type Tool interface {
	Spec() ToolSpec
	Execute(ctx context.Context, expenses UserExpenses, arguments json.RawMessage) (any, error)
}

type History interface {
	Recent(userID int64) []Message
	Remember(userID int64, messages ...Message)
}
