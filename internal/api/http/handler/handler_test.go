package handler_test

import (
	"context"
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
	"github.com/dtroode/gophermart/internal/application/model"
	dto "github.com/dtroode/gophermart/internal/application/request"
	"github.com/dtroode/gophermart/internal/application/response"
	"github.com/dtroode/gophermart/internal/auth"
	"github.com/dtroode/gophermart/internal/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type failReader struct {
}

func (m *failReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("failed to read")
}

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
				service.On("RegisterUser", mock.Anything, &dto.RegisterUser{
					Login:    "testuser",
					Password: "testpassword",
				}).Once().Return("", application.ErrConflict)
				return service
			}(),
			wantError:          true,
			expectedStatusCode: http.StatusConflict,
		},
		"service error internal": {
			requestBody: `{"login": "testuser", "password": "testpassword"}`,
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("RegisterUser", mock.Anything, &dto.RegisterUser{
					Login:    "testuser",
					Password: "testpassword",
				}).Once().Return("", errors.New("service error"))
				return service
			}(),
			wantError:          true,
			expectedStatusCode: http.StatusInternalServerError,
		},
		"success": {
			requestBody: `{"login": "testuser", "password": "testpassword"}`,
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("RegisterUser", mock.Anything, &dto.RegisterUser{
					Login:    "testuser",
					Password: "testpassword",
				}).Once().Return("testtoken", nil)
				return service
			}(),
			expectedStatusCode: http.StatusOK,
			expectedAuthHeader: "Bearer testtoken",
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/user/register", strings.NewReader(tt.requestBody))

			h := handler.New(tt.serviceMock, dummyLogger)

			h.RegisterUser(w, r)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if !tt.wantError {
				assert.Equal(t, tt.expectedAuthHeader, w.Result().Header.Get("authorization"))
			}
		})
	}
}

func TestHandler_Login(t *testing.T) {
	dummyLogger := &logger.Logger{
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	tests := map[string]struct {
		requestBody        string
		serviceMock        *mocks.Service
		expectedStatusCode int
		wantError          bool
		expectedAuthHeader string
	}{
		"failed to decode body": {
			requestBody:        `s`,
			wantError:          true,
			expectedStatusCode: http.StatusBadRequest,
		},
		"service error unauthorized": {
			requestBody: `{"login": "testuser", "password": "testpassword"}`,
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("Login", mock.Anything, &dto.Login{
					Login:    "testuser",
					Password: "testpassword",
				}).Once().Return("", application.ErrUnauthorized)
				return service
			}(),
			wantError:          true,
			expectedStatusCode: http.StatusUnauthorized,
		},
		"service error internal": {
			requestBody: `{"login": "testuser", "password": "testpassword"}`,
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("Login", mock.Anything, &dto.Login{
					Login:    "testuser",
					Password: "testpassword",
				}).Once().Return("", errors.New("service error"))
				return service
			}(),
			wantError:          true,
			expectedStatusCode: http.StatusInternalServerError,
		},
		"success": {
			requestBody: `{"login": "testuser", "password": "testpassword"}`,
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("Login", mock.Anything, &dto.Login{
					Login:    "testuser",
					Password: "testpassword",
				}).Once().Return("testtoken", nil)
				return service
			}(),
			expectedStatusCode: http.StatusOK,
			expectedAuthHeader: "Bearer testtoken",
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/user/login", strings.NewReader(tt.requestBody))

			h := handler.New(tt.serviceMock, dummyLogger)

			h.Login(w, r)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if !tt.wantError {
				assert.Equal(t, tt.expectedAuthHeader, w.Result().Header.Get("authorization"))
			}
		})
	}
}

func TestHandler_UploadOrder(t *testing.T) {
	dummyLogger := &logger.Logger{
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	userID := uuid.New()

	tests := map[string]struct {
		ctx                context.Context
		requestBody        io.Reader
		serviceMock        *mocks.Service
		expectedStatusCode int
	}{
		"failed to get user id from context": {
			ctx:                context.Background(),
			expectedStatusCode: http.StatusInternalServerError,
		},
		"failed to read body": {
			ctx:                auth.SetUserIDToContext(context.Background(), userID),
			requestBody:        &failReader{},
			expectedStatusCode: http.StatusBadRequest,
		},
		"service error already exists": {
			ctx:         auth.SetUserIDToContext(context.Background(), userID),
			requestBody: strings.NewReader("1234"),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("UploadOrder", mock.Anything, &dto.UploadOrder{
					UserID:      userID,
					OrderNumber: "1234",
				}).Once().Return(nil, application.ErrAlreadyExist)
				return service
			}(),
			expectedStatusCode: http.StatusOK,
		},
		"service error conflict": {
			ctx:         auth.SetUserIDToContext(context.Background(), userID),
			requestBody: strings.NewReader("1234"),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("UploadOrder", mock.Anything, &dto.UploadOrder{
					UserID:      userID,
					OrderNumber: "1234",
				}).Once().Return(nil, application.ErrConflict)
				return service
			}(),
			expectedStatusCode: http.StatusConflict,
		},
		"service error unprocessable": {
			ctx:         auth.SetUserIDToContext(context.Background(), userID),
			requestBody: strings.NewReader("1234"),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("UploadOrder", mock.Anything, &dto.UploadOrder{
					UserID:      userID,
					OrderNumber: "1234",
				}).Once().Return(nil, application.ErrUnprocessable)
				return service
			}(),
			expectedStatusCode: http.StatusUnprocessableEntity,
		},
		"service error internal": {
			ctx:         auth.SetUserIDToContext(context.Background(), userID),
			requestBody: strings.NewReader("1234"),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("UploadOrder", mock.Anything, &dto.UploadOrder{
					UserID:      userID,
					OrderNumber: "1234",
				}).Once().Return(nil, errors.New("service error"))
				return service
			}(),
			expectedStatusCode: http.StatusInternalServerError,
		},
		"success": {
			ctx:         auth.SetUserIDToContext(context.Background(), userID),
			requestBody: strings.NewReader("1234"),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("UploadOrder", mock.Anything, &dto.UploadOrder{
					UserID:      userID,
					OrderNumber: "1234",
				}).Once().Return(&model.Order{}, nil)
				return service
			}(),
			expectedStatusCode: http.StatusAccepted,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/user/orders", tt.requestBody)
			r = r.WithContext(tt.ctx)

			h := handler.New(tt.serviceMock, dummyLogger)

			h.UploadOrder(w, r)

			assert.Equal(t, tt.expectedStatusCode, w.Code)
		})
	}
}

func TestHandler_ListUserOrders(t *testing.T) {
	dummyLogger := &logger.Logger{
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	userID := uuid.New()

	tests := map[string]struct {
		ctx                context.Context
		serviceMock        *mocks.Service
		wantError          bool
		expectedStatusCode int
		expectedResponse   string
	}{
		"failed to get user id from context": {
			ctx:                context.Background(),
			wantError:          true,
			expectedStatusCode: http.StatusInternalServerError,
		},
		"service error no data": {
			ctx: auth.SetUserIDToContext(context.Background(), userID),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("ListUserOrders", mock.Anything, userID).Once().Return(nil, application.ErrNoData)
				return service
			}(),
			wantError:          true,
			expectedStatusCode: http.StatusNoContent,
		},
		"service error internal": {
			ctx: auth.SetUserIDToContext(context.Background(), userID),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("ListUserOrders", mock.Anything, userID).Once().Return(nil, errors.New("service error"))
				return service
			}(),
			wantError:          true,
			expectedStatusCode: http.StatusInternalServerError,
		},
		"success": {
			ctx: auth.SetUserIDToContext(context.Background(), userID),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("ListUserOrders", mock.Anything, userID).Once().
					Return([]*response.UserOrder{
						{
							Number:     "1234",
							Status:     "PROCESSING",
							Accrual:    0,
							UploadedAt: "some-time",
						},
						{
							Number:     "5678",
							Status:     "PROCESSED",
							Accrual:    5,
							UploadedAt: "diff-time",
						},
					}, nil)
				return service
			}(),
			expectedStatusCode: http.StatusOK,
			expectedResponse: `[
			{"number": "1234", "status": "PROCESSING", "uploaded_at": "some-time"},
			{"number": "5678", "status": "PROCESSED", "accrual": 5, "uploaded_at": "diff-time"}]`,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/user/orders", nil)
			r = r.WithContext(tt.ctx)

			h := handler.New(tt.serviceMock, dummyLogger)

			h.ListUserOrders(w, r)

			res := w.Result()
			defer res.Body.Close()

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if !tt.wantError {
				resBody, err := io.ReadAll(res.Body)
				require.NoError(t, err)

				assert.Equal(t, "application/json", res.Header.Get("content-type"))
				assert.JSONEq(t, tt.expectedResponse, string(resBody))
			}
		})
	}
}

func TestHandler_GetUserBalance(t *testing.T) {
	dummyLogger := &logger.Logger{
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	userID := uuid.New()

	tests := map[string]struct {
		ctx                context.Context
		serviceMock        *mocks.Service
		wantError          bool
		expectedStatusCode int
		expectedResponse   string
	}{
		"failed to get user id from context": {
			ctx:                context.Background(),
			wantError:          true,
			expectedStatusCode: http.StatusInternalServerError,
		},
		"service error intenal": {
			ctx: auth.SetUserIDToContext(context.Background(), userID),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("GetUserBalance", mock.Anything, userID).Once().
					Return(nil, errors.New("service error"))
				return service
			}(),
			wantError:          true,
			expectedStatusCode: http.StatusInternalServerError,
		},
		"success": {
			ctx: auth.SetUserIDToContext(context.Background(), userID),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("GetUserBalance", mock.Anything, userID).Once().
					Return(&response.UserBalance{
						Current:   5,
						Withdrawn: 55.8,
					}, nil)
				return service
			}(),
			expectedStatusCode: http.StatusOK,
			expectedResponse:   `{"current": 5, "withdrawn": 55.8}`,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/user/balance", nil)
			r = r.WithContext(tt.ctx)

			h := handler.New(tt.serviceMock, dummyLogger)

			h.GetUserBalance(w, r)

			res := w.Result()
			defer res.Body.Close()

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if !tt.wantError {
				resBody, err := io.ReadAll(res.Body)
				require.NoError(t, err)

				assert.Equal(t, "application/json", res.Header.Get("content-type"))
				assert.JSONEq(t, tt.expectedResponse, string(resBody))
			}
		})
	}
}

func TestHandler_WithdrawUserBonuses(t *testing.T) {
	dummyLogger := &logger.Logger{
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	userID := uuid.New()

	tests := map[string]struct {
		ctx                context.Context
		requestBody        io.Reader
		serviceMock        *mocks.Service
		expectedStatusCode int
	}{
		"failed to get user id from context": {
			ctx:                context.Background(),
			expectedStatusCode: http.StatusInternalServerError,
		},
		"failed to decode request body": {
			ctx:                auth.SetUserIDToContext(context.Background(), userID),
			requestBody:        &failReader{},
			expectedStatusCode: http.StatusBadRequest,
		},
		"service error not enough bonuses": {
			ctx:         auth.SetUserIDToContext(context.Background(), userID),
			requestBody: strings.NewReader(`{"order": "1234", "sum": 50}`),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("WithdrawUserBonuses", mock.Anything, &dto.WithdrawBonuses{
					UserID:      userID,
					OrderNumber: "1234",
					Sum:         50,
				}).Once().Return(application.ErrNotEnoughBonuses)
				return service
			}(),
			expectedStatusCode: http.StatusPaymentRequired,
		},
		"service error unprocessable": {
			ctx:         auth.SetUserIDToContext(context.Background(), userID),
			requestBody: strings.NewReader(`{"order": "1234", "sum": 50}`),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("WithdrawUserBonuses", mock.Anything, &dto.WithdrawBonuses{
					UserID:      userID,
					OrderNumber: "1234",
					Sum:         50,
				}).Once().Return(application.ErrUnprocessable)
				return service
			}(),
			expectedStatusCode: http.StatusUnprocessableEntity,
		},
		"service error internal": {
			ctx:         auth.SetUserIDToContext(context.Background(), userID),
			requestBody: strings.NewReader(`{"order": "1234", "sum": 50}`),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("WithdrawUserBonuses", mock.Anything, &dto.WithdrawBonuses{
					UserID:      userID,
					OrderNumber: "1234",
					Sum:         50,
				}).Once().Return(errors.New("service error"))
				return service
			}(),
			expectedStatusCode: http.StatusInternalServerError,
		},
		"success": {
			ctx:         auth.SetUserIDToContext(context.Background(), userID),
			requestBody: strings.NewReader(`{"order": "1234", "sum": 50}`),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("WithdrawUserBonuses", mock.Anything, &dto.WithdrawBonuses{
					UserID:      userID,
					OrderNumber: "1234",
					Sum:         50,
				}).Once().Return(nil)
				return service
			}(),
			expectedStatusCode: http.StatusOK,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/user/balance/withdraw", tt.requestBody)
			r = r.WithContext(tt.ctx)

			h := handler.New(tt.serviceMock, dummyLogger)

			h.WithdrawUserBonuses(w, r)

			assert.Equal(t, tt.expectedStatusCode, w.Code)
		})
	}
}

func TestHandler_ListUserWithdrawals(t *testing.T) {
	dummyLogger := &logger.Logger{
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	userID := uuid.New()

	tests := map[string]struct {
		ctx                context.Context
		serviceMock        *mocks.Service
		wantError          bool
		expectedStatusCode int
		expectedResponse   string
	}{
		"failed to get user id from context": {
			ctx:                context.Background(),
			wantError:          true,
			expectedStatusCode: http.StatusInternalServerError,
		},
		"service error no data": {
			ctx: auth.SetUserIDToContext(context.Background(), userID),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("ListUserWithdrawals", mock.Anything, userID).Once().
					Return(nil, application.ErrNoData)
				return service
			}(),
			wantError:          true,
			expectedStatusCode: http.StatusNoContent,
		},
		"service error internal": {
			ctx: auth.SetUserIDToContext(context.Background(), userID),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("ListUserWithdrawals", mock.Anything, userID).Once().
					Return(nil, errors.New("service error"))
				return service
			}(),
			wantError:          true,
			expectedStatusCode: http.StatusInternalServerError,
		},
		"success": {
			ctx: auth.SetUserIDToContext(context.Background(), userID),
			serviceMock: func() *mocks.Service {
				service := mocks.NewService(t)
				service.On("ListUserWithdrawals", mock.Anything, userID).Once().
					Return([]*response.UserWithdrawal{
						{
							OrderNumber: "1234",
							Sum:         50.8,
							ProcessedAt: "some-time",
						},
						{
							OrderNumber: "5678",
							Sum:         3,
							ProcessedAt: "diff-time",
						},
					}, nil)
				return service
			}(),
			expectedStatusCode: http.StatusOK,
			expectedResponse: `[
			{"order": "1234", "sum": 50.8, "processed_at": "some-time"},
			{"order": "5678", "sum": 3, "processed_at": "diff-time"}]`,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/user/withdrawals", nil)
			r = r.WithContext(tt.ctx)

			h := handler.New(tt.serviceMock, dummyLogger)

			h.ListUserWithdrawals(w, r)

			res := w.Result()
			defer res.Body.Close()

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if !tt.wantError {
				resBody, err := io.ReadAll(res.Body)
				require.NoError(t, err)

				assert.Equal(t, "application/json", res.Header.Get("content-type"))
				assert.JSONEq(t, tt.expectedResponse, string(resBody))
			}
		})
	}
}
