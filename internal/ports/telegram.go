package ports

import "context"

type Responder interface {
	Handle(ctx context.Context, userID int64, text string) (string, error)
}

type AccessPolicy interface {
	IsUserAllowed(userID int64) bool
}
