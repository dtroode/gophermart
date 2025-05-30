package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SetUserIDToContext(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	ctx = SetUserIDToContext(ctx, userID)
	contextUserID, ok := ctx.Value(userIDKey).(uuid.UUID)

	require.True(t, ok)
	assert.Equal(t, userID, contextUserID)
}

func Test_GetUserIDFromContext(t *testing.T) {
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), userIDKey, userID)

	contextUserID, ok := GetUserIDFromContext(ctx)

	require.True(t, ok)
	assert.Equal(t, userID, contextUserID)
}
