package middleware

import (
	"net/http"
	"strings"

	"github.com/dtroode/gophermart/internal/auth"
	"github.com/dtroode/gophermart/internal/logger"
	"github.com/google/uuid"
)

type TokenManager interface {
	GetUserID(tokenString string) (uuid.UUID, error)
}

type Authenticate struct {
	tokenManager TokenManager
	logger       *logger.Logger
}

func NewAuthenticate(tokenManager TokenManager, l *logger.Logger) *Authenticate {
	return &Authenticate{
		tokenManager: tokenManager,
		logger:       l,
	}
}

func (m *Authenticate) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		userID, err := m.tokenManager.GetUserID(tokenString)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if userID == uuid.Nil {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		ctx := auth.SetUserIDToContext(r.Context(), userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
