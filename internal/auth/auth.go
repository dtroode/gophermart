package auth

import (
	"context"

	"github.com/google/uuid"
)

type contextKey int

const (
	userIDKey contextKey = iota + 1
)

func SetUserIDToContext(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	u, ok := ctx.Value(userIDKey).(uuid.UUID)
	return u, ok
}
