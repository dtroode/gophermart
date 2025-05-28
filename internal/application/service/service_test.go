package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dtroode/gophermart/internal/application"
	"github.com/dtroode/gophermart/internal/application/model"
	"github.com/dtroode/gophermart/internal/application/request"
	"github.com/dtroode/gophermart/internal/application/response"
	"github.com/dtroode/gophermart/internal/application/service"
	mocks "github.com/dtroode/gophermart/internal/application/service/mocks"
	"github.com/dtroode/gophermart/internal/application/storage"
	"github.com/dtroode/gophermart/internal/workerpool"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testNameKey string

const tnk testNameKey = "test"

func TestService_RegisterUser(t *testing.T) {
	ctx := context.WithValue(context.Background(), tnk, "RegisterUser")

	params := &request.RegisterUser{
		Login:    "test-login",
		Password: "test-password",
	}

	tests := map[string]struct {
		storageMock      *mocks.Storage
		hasherMock       *mocks.Hasher
		tokenManagerMock *mocks.TokenManager
		workerPoolMock   *mocks.WorkerPool
		expectedResp     string
		expectedErr      error
	}{
		"failed to get user by login": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(nil, errors.New("storage error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to check user with login: %w", errors.New("storage error")),
		},
		"login is taken": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(&model.User{}, nil)
				return mock
			}(),
			expectedErr: application.ErrConflict,
		},
		"failed to hash password": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(nil, application.ErrNotFound)
				return mock
			}(),
			hasherMock: func() *mocks.Hasher {
				mock := mocks.NewHasher(t)
				mock.On("Hash", ctx, []byte(params.Password)).Once().Return("", errors.New("hasher error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to hash password: %w", errors.New("hasher error")),
		},
		"failed to save user": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(nil, application.ErrNotFound)
				mock.On("SaveUser", ctx, &model.User{Login: params.Login, Password: "hash"}).Once().Return(nil, errors.New("storage error"))
				return mock
			}(),
			hasherMock: func() *mocks.Hasher {
				mock := mocks.NewHasher(t)
				mock.On("Hash", ctx, []byte(params.Password)).Once().Return("hash", nil)
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to save user: %w", errors.New("storage error")),
		},
		"failed to create token": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(nil, application.ErrNotFound)
				mock.On("SaveUser", ctx, &model.User{Login: params.Login, Password: "hash"}).Once().Return(&model.User{Login: params.Login, ID: uuid.Max}, nil)
				return mock
			}(),
			hasherMock: func() *mocks.Hasher {
				mock := mocks.NewHasher(t)
				mock.On("Hash", ctx, []byte(params.Password)).Once().Return("hash", nil)
				return mock
			}(),
			tokenManagerMock: func() *mocks.TokenManager {
				mock := mocks.NewTokenManager(t)
				mock.On("CreateToken", uuid.Max).Once().Return("", errors.New("token manager error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to create token: %w", errors.New("token manager error")),
		},
		"success": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(nil, application.ErrNotFound)
				mock.On("SaveUser", ctx, &model.User{Login: params.Login, Password: "hash"}).Once().Return(&model.User{Login: params.Login, ID: uuid.Max}, nil)
				return mock
			}(),
			hasherMock: func() *mocks.Hasher {
				mock := mocks.NewHasher(t)
				mock.On("Hash", ctx, []byte(params.Password)).Once().Return("hash", nil)
				return mock
			}(),
			tokenManagerMock: func() *mocks.TokenManager {
				mock := mocks.NewTokenManager(t)
				mock.On("CreateToken", uuid.Max).Once().Return("token", nil)
				return mock
			}(),
			expectedResp: "token",
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			s := service.NewService(
				tt.storageMock,
				tt.hasherMock,
				tt.tokenManagerMock,
				nil,
				nil)

			resp, err := s.RegisterUser(ctx, params)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				assert.Empty(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

func TestService_Login(t *testing.T) {
	ctx := context.WithValue(context.Background(), tnk, "Login")

	params := &request.Login{
		Login:    "test-login",
		Password: "test-password",
	}

	tests := map[string]struct {
		storageMock      *mocks.Storage
		hasherMock       *mocks.Hasher
		tokenManagerMock *mocks.TokenManager
		expectedResp     string
		expectedErr      error
	}{
		"failed to get user": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(nil, errors.New("storage error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to get user: %w", errors.New("storage error")),
		},
		"user not found": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(nil, application.ErrNotFound)
				return mock
			}(),
			expectedErr: application.ErrUnauthorized,
		},
		"failed to hash password": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(&model.User{}, nil)
				return mock
			}(),
			hasherMock: func() *mocks.Hasher {
				mock := mocks.NewHasher(t)
				mock.On("Hash", ctx, []byte(params.Password)).Once().Return("", errors.New("hasher error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to hash password: %w", errors.New("hasher error")),
		},
		"password doesn't match hash": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(&model.User{Password: "diff-hash"}, nil)
				return mock
			}(),
			hasherMock: func() *mocks.Hasher {
				mock := mocks.NewHasher(t)
				mock.On("Hash", ctx, []byte(params.Password)).Once().Return("hash", nil)
				return mock
			}(),
			expectedErr: application.ErrUnauthorized,
		},
		"failed to create token": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(&model.User{ID: uuid.Max, Password: "hash"}, nil)
				return mock
			}(),
			hasherMock: func() *mocks.Hasher {
				mock := mocks.NewHasher(t)
				mock.On("Hash", ctx, []byte(params.Password)).Once().Return("hash", nil)
				return mock
			}(),
			tokenManagerMock: func() *mocks.TokenManager {
				mock := mocks.NewTokenManager(t)
				mock.On("CreateToken", uuid.Max).Once().Return("", errors.New("token manager error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to create token: %w", errors.New("token manager error")),
		},
		"success": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserByLogin", ctx, params.Login).Once().Return(&model.User{ID: uuid.Max, Password: "hash"}, nil)
				return mock
			}(),
			hasherMock: func() *mocks.Hasher {
				mock := mocks.NewHasher(t)
				mock.On("Hash", ctx, []byte(params.Password)).Once().Return("hash", nil)
				return mock
			}(),
			tokenManagerMock: func() *mocks.TokenManager {
				mock := mocks.NewTokenManager(t)
				mock.On("CreateToken", uuid.Max).Once().Return("token", nil)
				return mock
			}(),
			expectedResp: "token",
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			s := service.NewService(
				tt.storageMock,
				tt.hasherMock,
				tt.tokenManagerMock,
				nil,
				nil)

			resp, err := s.Login(ctx, params)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				assert.Empty(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

func TestService_UploadOrder(t *testing.T) {
	ctx := context.WithValue(context.Background(), tnk, "UploadOrder")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	params := &request.UploadOrder{
		UserID: uuid.New(),
	}

	tests := map[string]struct {
		orderNumber  string
		storageMock  *mocks.Storage
		accrualMock  *mocks.AccrualAdapter
		poolMock     *mocks.WorkerPool
		expectedResp *model.Order
		expectedErr  error
	}{
		"order number isn't valid": {
			orderNumber: "4561261212345464",

			expectedErr: application.ErrUnprocessable,
		},
		"failed to check order exists": {
			orderNumber: "4561261212345467",
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetOrderByNumber", ctx, "4561261212345467").Once().Return(nil, errors.New("storage error"))
				return mock
			}(),

			expectedErr: fmt.Errorf("failed to check order exists: %w", errors.New("storage error")),
		},
		"order exists for same user": {
			orderNumber: "4561261212345467",
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetOrderByNumber", ctx, "4561261212345467").Once().Return(&model.Order{UserID: params.UserID}, nil)
				return mock
			}(),

			expectedErr: application.ErrAlreadyExist,
		},
		"order exists for different user": {
			orderNumber: "4561261212345467",
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetOrderByNumber", ctx, "4561261212345467").Once().Return(&model.Order{UserID: uuid.New()}, nil)
				return mock
			}(),

			expectedErr: application.ErrConflict,
		},
		"failed to save order": {
			orderNumber: "4561261212345467",
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetOrderByNumber", ctx, "4561261212345467").Once().Return(nil, application.ErrNotFound)
				mock.On("SaveOrder", ctx, &model.Order{UserID: params.UserID, Number: "4561261212345467", Status: model.OrderStatusNew}).Once().Return(nil, errors.New("storage error"))
				return mock
			}(),

			expectedErr: fmt.Errorf("failed to save order: %w", errors.New("storage error")),
		},
		"success": {
			orderNumber: "66465778752",
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetOrderByNumber", ctx, "66465778752").Once().Return(nil, application.ErrNotFound)
				mock.On("SaveOrder", ctx, &model.Order{UserID: params.UserID, Number: "66465778752", Status: model.OrderStatusNew}).Once().Return(&model.Order{ID: uuid.Max, UserID: params.UserID, Number: "66465778752", Status: model.OrderStatusNew}, nil)
				return mock
			}(),
			accrualMock: func() *mocks.AccrualAdapter {
				return mocks.NewAccrualAdapter(t)
			}(),
			poolMock: func() *mocks.WorkerPool {
				poolMock := mocks.NewWorkerPool(t)
				resultCh := make(chan *workerpool.Result)
				poolMock.On("Submit", context.Background(), 1*time.Hour, mock.Anything, false).Maybe().Return(resultCh)
				return poolMock
			}(),
			expectedResp: &model.Order{ID: uuid.Max, UserID: params.UserID, Number: "66465778752", Status: model.OrderStatusNew},
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			params.OrderNumber = tt.orderNumber

			s := service.NewService(
				tt.storageMock,
				nil,
				nil,
				tt.accrualMock,
				tt.poolMock)

			resp, err := s.UploadOrder(ctx, params)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				assert.Empty(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

func TestService_ListUserOrders(t *testing.T) {
	ctx := context.WithValue(context.Background(), tnk, "ListUserOrders")

	userID := uuid.New()

	tests := map[string]struct {
		storageMock  *mocks.Storage
		expectedResp []*response.UserOrder
		expectedErr  error
	}{
		"failed to get orders": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserOrdersNewestFirst", ctx, userID).Once().Return(nil, errors.New("storage error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to get user orders: %w", errors.New("storage error")),
		},
		"no orders found": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserOrdersNewestFirst", ctx, userID).Once().Return([]*model.Order{}, nil)
				return mock
			}(),
			expectedErr: application.ErrNoData,
		},
		"success": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserOrdersNewestFirst", ctx, userID).Once().Return([]*model.Order{
					{
						ID:        uuid.New(),
						UserID:    userID,
						Number:    "4561261212345467",
						Status:    model.OrderStatusProcessed,
						Accrual:   100,
						CreatedAt: time.Now(),
					},
				}, nil)
				return mock
			}(),
			expectedResp: []*response.UserOrder{
				{
					Number:     "4561261212345467",
					Status:     string(model.OrderStatusProcessed),
					Accrual:    100,
					UploadedAt: time.Now().Format(time.RFC3339),
				},
			},
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			s := service.NewService(
				tt.storageMock,
				nil,
				nil,
				nil,
				nil)

			resp, err := s.ListUserOrders(ctx, userID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedResp), len(resp))
				for i, order := range resp {
					assert.Equal(t, tt.expectedResp[i].Number, order.Number)
					assert.Equal(t, tt.expectedResp[i].Status, order.Status)
					assert.Equal(t, tt.expectedResp[i].Accrual, order.Accrual)
				}
			}
		})
	}
}

func TestService_GetUserBalance(t *testing.T) {
	ctx := context.WithValue(context.Background(), tnk, "GetUserBalance")

	userID := uuid.New()

	tests := map[string]struct {
		storageMock  *mocks.Storage
		expectedResp *response.UserBalance
		expectedErr  error
	}{
		"failed to get user": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUser", ctx, userID).Once().Return(nil, errors.New("storage error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to get user: %w", errors.New("storage error")),
		},
		"user not found": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUser", ctx, userID).Once().Return(nil, application.ErrNotFound)
				return mock
			}(),
			expectedErr: application.ErrNotFound,
		},
		"failed to get user withdrawal sum": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUser", ctx, userID).Once().Return(&model.User{Balance: float32(100)}, nil)
				mock.On("GetUserWithdrawalSum", ctx, userID).Once().Return(float32(0), errors.New("storage error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to get user withdrawal sum: %w", errors.New("storage error")),
		},
		"success": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUser", ctx, userID).Once().Return(&model.User{Balance: float32(100)}, nil)
				mock.On("GetUserWithdrawalSum", ctx, userID).Once().Return(float32(50), nil)
				return mock
			}(),
			expectedResp: &response.UserBalance{
				Current:   100,
				Withdrawn: 50,
			},
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			s := service.NewService(
				tt.storageMock,
				nil,
				nil,
				nil,
				nil)

			resp, err := s.GetUserBalance(ctx, userID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

func TestService_WithdrawUserBonuses(t *testing.T) {
	ctx := context.WithValue(context.Background(), tnk, "WithdrawUserBonuses")

	params := &request.WithdrawBonuses{
		UserID: uuid.New(),
		Sum:    100,
	}

	tests := map[string]struct {
		orderNumber string
		storageMock *mocks.Storage
		expectedErr error
	}{
		"order number isn't valid": {
			orderNumber: "4561261212345464",
			expectedErr: application.ErrUnprocessable,
		},
		"user not found": {
			orderNumber: "4561261212345467",
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("WithdrawUserBonuses", ctx, &storage.WithdrawUserBonuses{
					UserID:   params.UserID,
					OrderNum: "4561261212345467",
					Sum:      params.Sum,
				}).Once().Return(nil, application.ErrNotFound)
				return mock
			}(),
			expectedErr: application.ErrNotFound,
		},
		"not enough bonuses": {
			orderNumber: "4561261212345467",
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("WithdrawUserBonuses", ctx, &storage.WithdrawUserBonuses{
					UserID:   params.UserID,
					OrderNum: "4561261212345467",
					Sum:      params.Sum,
				}).Once().Return(nil, application.ErrNotEnoughBonuses)
				return mock
			}(),
			expectedErr: application.ErrNotEnoughBonuses,
		},
		"failed to withdraw user bonuses": {
			orderNumber: "4561261212345467",
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("WithdrawUserBonuses", ctx, &storage.WithdrawUserBonuses{
					UserID:   params.UserID,
					OrderNum: "4561261212345467",
					Sum:      params.Sum,
				}).Once().Return(nil, errors.New("storage error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to withdraw user bonuses: %w", errors.New("storage error")),
		},
		"success": {
			orderNumber: "4561261212345467",
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("WithdrawUserBonuses", ctx, &storage.WithdrawUserBonuses{
					UserID:   params.UserID,
					OrderNum: "4561261212345467",
					Sum:      params.Sum,
				}).Once().Return(&model.User{}, nil)
				return mock
			}(),
			expectedErr: nil,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			params.OrderNumber = tt.orderNumber

			s := service.NewService(
				tt.storageMock,
				nil,
				nil,
				nil,
				nil)

			err := s.WithdrawUserBonuses(ctx, params)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_ListUserWithdrawals(t *testing.T) {
	ctx := context.WithValue(context.Background(), tnk, "ListUserWithdrawals")

	userID := uuid.New()

	tests := map[string]struct {
		storageMock  *mocks.Storage
		expectedResp []*response.UserWithdrawal
		expectedErr  error
	}{
		"failed to get user withdrawals": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserWithdrawals", ctx, userID).Once().Return(nil, errors.New("storage error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to get user withdrawals: %w", errors.New("storage error")),
		},
		"no withdrawals": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserWithdrawals", ctx, userID).Once().Return([]*model.WithdrawalOrder{}, nil)
				return mock
			}(),
			expectedErr: application.ErrNoData,
		},
		"success": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUserWithdrawals", ctx, userID).Once().Return([]*model.WithdrawalOrder{
					{OrderNumber: "1234567890", Amount: 100, CreatedAt: time.Now()},
				}, nil)
				return mock
			}(),
			expectedResp: []*response.UserWithdrawal{
				{OrderNumber: "1234567890", Sum: 100, ProcessedAt: time.Now().Format(time.RFC3339)},
			},
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			s := service.NewService(tt.storageMock, nil, nil, nil, nil)
			resp, err := s.ListUserWithdrawals(ctx, userID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}
