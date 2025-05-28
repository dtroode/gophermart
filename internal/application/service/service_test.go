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
	now := time.Now()

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
		"success with orders": {
			storageMock: func() *mocks.Storage {
				mockStorage := mocks.NewStorage(t)
				mockStorage.On("GetUserOrdersNewestFirst", ctx, userID).Once().Return([]*model.Order{
					{
						ID:        uuid.New(),
						UserID:    userID,
						Number:    "4561261212345467",
						Status:    model.OrderStatusProcessed,
						Accrual:   10000,
						CreatedAt: now,
					},
					{
						ID:        uuid.New(),
						UserID:    userID,
						Number:    "1234567890123456",
						Status:    model.OrderStatusNew,
						Accrual:   0,
						CreatedAt: now.Add(-time.Hour),
					},
				}, nil)
				return mockStorage
			}(),
			expectedResp: []*response.UserOrder{
				{
					Number:     "4561261212345467",
					Status:     string(model.OrderStatusProcessed),
					Accrual:    100.00,
					UploadedAt: now.Format(time.RFC3339),
				},
				{
					Number:     "1234567890123456",
					Status:     string(model.OrderStatusNew),
					Accrual:    0.00,
					UploadedAt: now.Add(-time.Hour).Format(time.RFC3339),
				},
			},
		},
		"success with zero accrual": {
			storageMock: func() *mocks.Storage {
				mockStorage := mocks.NewStorage(t)
				mockStorage.On("GetUserOrdersNewestFirst", ctx, userID).Once().Return([]*model.Order{
					{
						ID:        uuid.New(),
						UserID:    userID,
						Number:    "ORDER_ZERO_ACCRUAL",
						Status:    model.OrderStatusProcessed,
						Accrual:   0,
						CreatedAt: now,
					},
				}, nil)
				return mockStorage
			}(),
			expectedResp: []*response.UserOrder{
				{
					Number:     "ORDER_ZERO_ACCRUAL",
					Status:     string(model.OrderStatusProcessed),
					Accrual:    0.0,
					UploadedAt: now.Format(time.RFC3339),
				},
			},
		},
		"success with specific accrual": {
			storageMock: func() *mocks.Storage {
				mockStorage := mocks.NewStorage(t)
				mockStorage.On("GetUserOrdersNewestFirst", ctx, userID).Once().Return([]*model.Order{
					{
						ID:        uuid.New(),
						UserID:    userID,
						Number:    "ORDER_12345_ACCRUAL",
						Status:    model.OrderStatusProcessed,
						Accrual:   12345,
						CreatedAt: now,
					},
				}, nil)
				return mockStorage
			}(),
			expectedResp: []*response.UserOrder{
				{
					Number:     "ORDER_12345_ACCRUAL",
					Status:     string(model.OrderStatusProcessed),
					Accrual:    123.45,
					UploadedAt: now.Format(time.RFC3339),
				},
			},
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			s := service.NewService(tt.storageMock, nil, nil, nil, nil)
			resp, err := s.ListUserOrders(ctx, userID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedResp), len(resp))
				for i, expectedOrder := range tt.expectedResp {
					actualOrder := resp[i]
					assert.Equal(t, expectedOrder.Number, actualOrder.Number)
					assert.Equal(t, expectedOrder.Status, actualOrder.Status)
					assert.InDelta(t, expectedOrder.Accrual, actualOrder.Accrual, 0.001)
					assert.Equal(t, expectedOrder.UploadedAt, actualOrder.UploadedAt)
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
				mock.On("GetUser", ctx, userID).Once().Return(&model.User{Balance: 10000}, nil)
				mock.On("GetUserWithdrawalSum", ctx, userID).Once().Return(int32(0), errors.New("storage error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to get user withdrawal sum: %w", errors.New("storage error")),
		},
		"success general case": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUser", ctx, userID).Once().Return(&model.User{Balance: 10000}, nil)
				mock.On("GetUserWithdrawalSum", ctx, userID).Once().Return(int32(5000), nil)
				return mock
			}(),
			expectedResp: &response.UserBalance{
				Current:   100.00,
				Withdrawn: 50.00,
			},
		},
		"balance is zero": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUser", ctx, userID).Once().Return(&model.User{Balance: 0}, nil)
				mock.On("GetUserWithdrawalSum", ctx, userID).Once().Return(int32(0), nil)
				return mock
			}(),
			expectedResp: &response.UserBalance{
				Current:   0.0,
				Withdrawn: 0.0,
			},
		},
		"balance is 20560 cents": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUser", ctx, userID).Once().Return(&model.User{Balance: 20560}, nil)
				mock.On("GetUserWithdrawalSum", ctx, userID).Once().Return(int32(0), nil)
				return mock
			}(),
			expectedResp: &response.UserBalance{
				Current:   205.60,
				Withdrawn: 0.0,
			},
		},
		"withdrawn is 10025 cents": {
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("GetUser", ctx, userID).Once().Return(&model.User{Balance: 30000}, nil)
				mock.On("GetUserWithdrawalSum", ctx, userID).Once().Return(int32(10025), nil)
				return mock
			}(),
			expectedResp: &response.UserBalance{
				Current:   300.00,
				Withdrawn: 100.25,
			},
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			s := service.NewService(tt.storageMock, nil, nil, nil, nil)
			resp, err := s.GetUserBalance(ctx, userID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.InDelta(t, tt.expectedResp.Current, resp.Current, 0.001)
				assert.InDelta(t, tt.expectedResp.Withdrawn, resp.Withdrawn, 0.001)
			}
		})
	}
}

func TestService_WithdrawUserBonuses(t *testing.T) {
	ctx := context.WithValue(context.Background(), tnk, "WithdrawUserBonuses")
	userID := uuid.New()

	tests := map[string]struct {
		params      *request.WithdrawBonuses
		orderNumber string
		storageMock *mocks.Storage
		expectedErr error
	}{
		"order number isn't valid": {
			params:      &request.WithdrawBonuses{UserID: userID, OrderNumber: "4561261212345464", Sum: 10.0},
			expectedErr: application.ErrUnprocessable,
		},
		"user not found": {
			params: &request.WithdrawBonuses{UserID: userID, OrderNumber: "4561261212345467", Sum: 10.0},
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("WithdrawUserBonuses", ctx, &storage.WithdrawUserBonuses{
					UserID:   userID,
					OrderNum: "4561261212345467",
					Sum:      1000,
				}).Once().Return(nil, application.ErrNotFound)
				return mock
			}(),
			expectedErr: application.ErrNotFound,
		},
		"not enough bonuses": {
			params: &request.WithdrawBonuses{UserID: userID, OrderNumber: "4561261212345467", Sum: 100.0},
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("WithdrawUserBonuses", ctx, &storage.WithdrawUserBonuses{
					UserID:   userID,
					OrderNum: "4561261212345467",
					Sum:      10000,
				}).Once().Return(nil, application.ErrNotEnoughBonuses)
				return mock
			}(),
			expectedErr: application.ErrNotEnoughBonuses,
		},
		"failed to withdraw user bonuses": {
			params: &request.WithdrawBonuses{UserID: userID, OrderNumber: "4561261212345467", Sum: 20.0},
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)
				mock.On("WithdrawUserBonuses", ctx, &storage.WithdrawUserBonuses{
					UserID:   userID,
					OrderNum: "4561261212345467",
					Sum:      2000,
				}).Once().Return(nil, errors.New("storage error"))
				return mock
			}(),
			expectedErr: fmt.Errorf("failed to withdraw user bonuses: %w", errors.New("storage error")),
		},
		"success": {
			params: &request.WithdrawBonuses{UserID: userID, OrderNumber: "4561261212345467", Sum: 10.50},
			storageMock: func() *mocks.Storage {
				mock := mocks.NewStorage(t)

				expectedStorageDTO := &storage.WithdrawUserBonuses{
					UserID:   userID,
					OrderNum: "4561261212345467",
					Sum:      1050,
				}
				mock.On("WithdrawUserBonuses", ctx, expectedStorageDTO).Once().Return(&model.User{}, nil)
				return mock
			}(),
			expectedErr: nil,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			s := service.NewService(tt.storageMock, nil, nil, nil, nil)
			err := s.WithdrawUserBonuses(ctx, tt.params)

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
	now := time.Now()

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
		"success with withdrawals": {
			storageMock: func() *mocks.Storage {
				mockStorage := mocks.NewStorage(t)
				mockStorage.On("GetUserWithdrawals", ctx, userID).Once().Return([]*model.WithdrawalOrder{
					{OrderNumber: "1234567890", Amount: 10000, CreatedAt: now},
					{OrderNumber: "0987654321", Amount: 7890, CreatedAt: now.Add(-time.Hour)},
				}, nil)
				return mockStorage
			}(),
			expectedResp: []*response.UserWithdrawal{
				{OrderNumber: "1234567890", Sum: 100.00, ProcessedAt: now.Format(time.RFC3339)},
				{OrderNumber: "0987654321", Sum: 78.90, ProcessedAt: now.Add(-time.Hour).Format(time.RFC3339)},
			},
		},
		"success with specific withdrawal sum": {
			storageMock: func() *mocks.Storage {
				mockStorage := mocks.NewStorage(t)
				mockStorage.On("GetUserWithdrawals", ctx, userID).Once().Return([]*model.WithdrawalOrder{
					{OrderNumber: "WITHDRAW_7890", Amount: 7890, CreatedAt: now},
				}, nil)
				return mockStorage
			}(),
			expectedResp: []*response.UserWithdrawal{
				{OrderNumber: "WITHDRAW_7890", Sum: 78.90, ProcessedAt: now.Format(time.RFC3339)},
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
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedResp), len(resp))
				for i, expectedWithdrawal := range tt.expectedResp {
					actualWithdrawal := resp[i]
					assert.Equal(t, expectedWithdrawal.OrderNumber, actualWithdrawal.OrderNumber)
					assert.InDelta(t, expectedWithdrawal.Sum, actualWithdrawal.Sum, 0.001)
					assert.Equal(t, expectedWithdrawal.ProcessedAt, actualWithdrawal.ProcessedAt)
				}
			}
		})
	}
}
