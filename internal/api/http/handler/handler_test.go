package handler_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dtroode/gophermart/internal/api/http/handler"
	"github.com/dtroode/gophermart/internal/api/http/handler/mocks"
	"github.com/dtroode/gophermart/internal/application"
	"github.com/dtroode/gophermart/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_RegisterUser(t *testing.T) {
	dummyLogger := &logger.Logger{
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	tests := map[string]struct {
		serviceMock        *mocks.Service
		requestBody        string
		wantError          bool
		expectedStatusCode int
		expectedAuthHeader string
	}{
		"failed to decode body": {
			requestBody:        `s`,
			wantError:          true,
			expectedStatusCode: http.StatusBadRequest,
		},
		"service error conflict": {
			requestBody: `{"login": "testuser", "password": "testpassword"}`,
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("RegisterUser", mock.Anything, mock.Anything).Once().Return("", application.ErrConflict)
				return service
			}(),
			wantError:          true,
			expectedStatusCode: http.StatusConflict,
		},
		"service error internal": {
			requestBody: `{"login": "testuser", "password": "testpassword"}`,
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("RegisterUser", mock.Anything, mock.Anything).Once().Return("", errors.New("service error"))
				return service
			}(),
			wantError:          true,
			expectedStatusCode: http.StatusInternalServerError,
		},
		"success": {
			requestBody: `{"login": "testuser", "password": "testpassword"}`,
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("RegisterUser", mock.Anything, mock.Anything).Once().Return("testtoken", nil)
				return service
			}(),
			expectedStatusCode: http.StatusOK,
			expectedAuthHeader: "Bearer testtoken",
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/user/register", strings.NewReader(tt.requestBody))

			handler := handler.New(tt.serviceMock, dummyLogger)

			handler.RegisterUser(w, r)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if !tt.wantError {
				assert.Equal(t, tt.expectedAuthHeader, w.Result().Header.Get("authorization"))
			}
		})
	}
}

// func TestHandler_Login(t *testing.T) {

// }
