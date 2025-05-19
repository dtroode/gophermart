package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/dtroode/gophermart/internal/auth"
	"github.com/dtroode/gophermart/internal/logger"
	"github.com/google/uuid"
)

type Token interface {
	GetUserID(ctx context.Context, tokenString string) (uuid.UUID, error)
	CreateToken(ctx context.Context, userID uuid.UUID) (string, error)
}

type Authenticate struct {
	token  Token
	logger *logger.Logger
}

func NewAuthenticate(token Token, l *logger.Logger) *Authenticate {
	return &Authenticate{
		token:  token,
		logger: l,
	}
}

func (m *Authenticate) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.logger.Info("[AUTH] auth header is empty")
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			m.logger.Info("[AUTH] token doesn't start with Bearer")
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		userID, err := m.token.GetUserID(ctx, tokenString)
		if err != nil {
			m.logger.Info("[AUTH] failed to get user id", "error", err)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if userID == uuid.Nil {
			m.logger.Info("[AUTH] user id is nil")
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		ctx = auth.SetUserIDToContext(r.Context(), userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
