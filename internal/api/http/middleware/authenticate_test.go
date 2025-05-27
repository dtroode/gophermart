package middleware_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dtroode/gophermart/internal/api/http/middleware"
	"github.com/dtroode/gophermart/internal/api/http/middleware/mocks"
	"github.com/dtroode/gophermart/internal/auth"
	"github.com/dtroode/gophermart/internal/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticate_Handle(t *testing.T) {
	dummyLogger := &logger.Logger{
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	userID := uuid.New()

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, ok := auth.GetUserIDFromContext(r.Context())
		require.True(t, ok)

		assert.Equal(t, userID, ctxUserID)
		w.WriteHeader(http.StatusOK)
	})

	tests := map[string]struct {
		req                *http.Request
		tokenManagerMock   *mocks.TokenManager
		expectedStatusCode int
	}{
		"no auth header": {
			req:                httptest.NewRequest("GET", "/", nil),
			expectedStatusCode: http.StatusUnauthorized,
		},
		"auth header not in proper format": {
			req: func() *http.Request {
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Authorization", "some")
				return r
			}(),
			expectedStatusCode: http.StatusUnauthorized,
		},
		"failed to get user id from token": {
			req: func() *http.Request {
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Authorization", "Bearer some.jwt.token")
				return r
			}(),
			tokenManagerMock: func() *mocks.TokenManager {
				tokenManager := mocks.NewTokenManager(t)
				tokenManager.On("GetUserID", "some.jwt.token").Once().
					Return(uuid.Nil, errors.New("token manager error"))
				return tokenManager
			}(),
			expectedStatusCode: http.StatusUnauthorized,
		},
		"user id is nil": {
			req: func() *http.Request {
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Authorization", "Bearer some.jwt.token")
				return r
			}(),
			tokenManagerMock: func() *mocks.TokenManager {
				tokenManager := mocks.NewTokenManager(t)
				tokenManager.On("GetUserID", "some.jwt.token").Once().
					Return(uuid.Nil, nil)
				return tokenManager
			}(),
			expectedStatusCode: http.StatusUnauthorized,
		},
		"success": {
			req: func() *http.Request {
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Authorization", "Bearer some.jwt.token")
				return r
			}(),
			tokenManagerMock: func() *mocks.TokenManager {
				tokenManager := mocks.NewTokenManager(t)
				tokenManager.On("GetUserID", "some.jwt.token").Once().
					Return(userID, nil)
				return tokenManager
			}(),
			expectedStatusCode: http.StatusOK,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()

			a := middleware.NewAuthenticate(tt.tokenManagerMock, dummyLogger)

			a.Handle(dummyHandler).ServeHTTP(w, tt.req)

			require.Equal(t, tt.expectedStatusCode, w.Code)
		})
	}
}
