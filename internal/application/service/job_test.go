package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dtroode/gophermart/internal/application"
	"github.com/dtroode/gophermart/internal/application/model"
	"github.com/dtroode/gophermart/internal/application/service/mocks"
	"github.com/dtroode/gophermart/internal/application/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_checkOrderJob(t *testing.T) {
	orderID := uuid.New()
	orderNumber := "1234"

	tests := map[string]struct {
		accrualMock  *mocks.AccrualAdapter
		storageMock  *mocks.Storage
		expectedResp any
		expectedErr  error
	}{
		"order status invalid": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).Once().
					Return(&model.AccrualOrder{
						Number: orderNumber,
						Status: model.AccrualOrderStatusInvalid,
					}, nil)
				return accrual
			}(),
			storageMock: func() *mocks.Storage {
				storageMock := mocks.NewStorage(t)
				storageMock.On("SetOrderStatus", mock.Anything, &storage.SetOrderStatus{
					ID:     orderID,
					Status: model.OrderStatusInvalid,
				}).Once().Return(&model.Order{
					ID:     orderID,
					Number: orderNumber,
					Status: model.OrderStatusInvalid,
				}, nil)
				return storageMock
			}(),
			expectedResp: &model.Order{
				ID:     orderID,
				Number: orderNumber,
				Status: model.OrderStatusInvalid,
			},
		},
		"order status processed": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).Once().
					Return(&model.AccrualOrder{
						Status:  model.AccrualOrderStatusProcessed,
						Accrual: 50,
						Number:  orderNumber,
					}, nil)
				return accrual
			}(),
			storageMock: func() *mocks.Storage {
				storageMock := mocks.NewStorage(t)
				storageMock.On("SetOrderStatusAndAccrual", mock.Anything, &storage.SetOrderStatusAndAccrual{
					ID:      orderID,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}).Once().Return(&model.Order{
					ID:      orderID,
					Number:  orderNumber,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}, nil)
				return storageMock
			}(),
			expectedResp: &model.Order{
				ID:      orderID,
				Number:  orderNumber,
				Accrual: 50,
				Status:  model.OrderStatusProcessed,
			},
		},
		"order status processing and than processed": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).
					Return(&model.AccrualOrder{
						Status: model.AccrualOrderStatusProcessing,
						Number: orderNumber,
					}, nil).Once()
				accrual.On("GetOrder", mock.Anything, orderNumber).Return(&model.AccrualOrder{
					Status:  model.AccrualOrderStatusProcessed,
					Accrual: 50,
					Number:  orderNumber,
				}, nil).Once()
				return accrual
			}(),
			storageMock: func() *mocks.Storage {
				storageMock := mocks.NewStorage(t)
				storageMock.On("SetOrderStatus", mock.Anything, &storage.SetOrderStatus{
					ID:     orderID,
					Status: model.OrderStatusProcessing,
				}).Once().Return(&model.Order{
					ID:     orderID,
					Number: orderNumber,
					Status: model.OrderStatusProcessing,
				}, nil)
				storageMock.On("SetOrderStatusAndAccrual", mock.Anything, &storage.SetOrderStatusAndAccrual{
					ID:      orderID,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}).Once().Return(&model.Order{
					ID:      orderID,
					Number:  orderNumber,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}, nil)
				return storageMock
			}(),
			expectedResp: &model.Order{
				ID:      orderID,
				Number:  orderNumber,
				Accrual: 50,
				Status:  model.OrderStatusProcessed,
			},
		},
		"order status processing and than invalid": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).
					Return(&model.AccrualOrder{
						Status: model.AccrualOrderStatusProcessing,
						Number: orderNumber,
					}, nil).Once()
				accrual.On("GetOrder", mock.Anything, orderNumber).Return(&model.AccrualOrder{
					Status: model.AccrualOrderStatusInvalid,
					Number: orderNumber,
				}, nil).Once()
				return accrual
			}(),
			storageMock: func() *mocks.Storage {
				storageMock := mocks.NewStorage(t)
				storageMock.On("SetOrderStatus", mock.Anything, &storage.SetOrderStatus{
					ID:     orderID,
					Status: model.OrderStatusProcessing,
				}).Once().Return(&model.Order{
					ID:     orderID,
					Number: orderNumber,
					Status: model.OrderStatusProcessing,
				}, nil)
				storageMock.On("SetOrderStatus", mock.Anything, &storage.SetOrderStatus{
					ID:     orderID,
					Status: model.OrderStatusInvalid,
				}).Once().Return(&model.Order{
					ID:     orderID,
					Number: orderNumber,
					Status: model.OrderStatusInvalid,
				}, nil)
				return storageMock
			}(),
			expectedResp: &model.Order{
				ID:     orderID,
				Number: orderNumber,
				Status: model.OrderStatusInvalid,
			},
		},
		"order status registered and than processed": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).
					Return(&model.AccrualOrder{
						Status: model.AccrualOrderStatusRegistered,
						Number: orderNumber,
					}, nil).Once()
				accrual.On("GetOrder", mock.Anything, orderNumber).Return(&model.AccrualOrder{
					Status:  model.AccrualOrderStatusProcessed,
					Accrual: 50,
					Number:  orderNumber,
				}, nil).Once()
				return accrual
			}(),
			storageMock: func() *mocks.Storage {
				storageMock := mocks.NewStorage(t)
				storageMock.On("SetOrderStatusAndAccrual", mock.Anything, &storage.SetOrderStatusAndAccrual{
					ID:      orderID,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}).Once().Return(&model.Order{
					ID:      orderID,
					Number:  orderNumber,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}, nil)
				return storageMock
			}(),
			expectedResp: &model.Order{
				ID:      orderID,
				Number:  orderNumber,
				Accrual: 50,
				Status:  model.OrderStatusProcessed,
			},
		},
		"order status registered and than invalid": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).
					Return(&model.AccrualOrder{
						Status: model.AccrualOrderStatusRegistered,
						Number: orderNumber,
					}, nil).Once()
				accrual.On("GetOrder", mock.Anything, orderNumber).Return(&model.AccrualOrder{
					Status: model.AccrualOrderStatusInvalid,
					Number: orderNumber,
				}, nil).Once()
				return accrual
			}(),
			storageMock: func() *mocks.Storage {
				storageMock := mocks.NewStorage(t)
				storageMock.On("SetOrderStatus", mock.Anything, &storage.SetOrderStatus{
					ID:     orderID,
					Status: model.OrderStatusInvalid,
				}).Once().Return(&model.Order{
					ID:     orderID,
					Number: orderNumber,
					Status: model.OrderStatusInvalid,
				}, nil)
				return storageMock
			}(),
			expectedResp: &model.Order{
				ID:     orderID,
				Number: orderNumber,
				Status: model.OrderStatusInvalid,
			},
		},
		"order status processing 5 times and than processed": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).
					Return(&model.AccrualOrder{
						Status: model.AccrualOrderStatusProcessing,
						Number: orderNumber,
					}, nil).Times(5)
				accrual.On("GetOrder", mock.Anything, orderNumber).Return(&model.AccrualOrder{
					Status:  model.AccrualOrderStatusProcessed,
					Accrual: 50,
					Number:  orderNumber,
				}, nil).Once()
				return accrual
			}(),
			storageMock: func() *mocks.Storage {
				storageMock := mocks.NewStorage(t)
				storageMock.On("SetOrderStatus", mock.Anything, &storage.SetOrderStatus{
					ID:     orderID,
					Status: model.OrderStatusProcessing,
				}).Once().Return(&model.Order{
					ID:     orderID,
					Number: orderNumber,
					Status: model.OrderStatusProcessing,
				}, nil)
				storageMock.On("SetOrderStatusAndAccrual", mock.Anything, &storage.SetOrderStatusAndAccrual{
					ID:      orderID,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}).Once().Return(&model.Order{
					ID:      orderID,
					Number:  orderNumber,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}, nil)
				return storageMock
			}(),
			expectedResp: &model.Order{
				ID:      orderID,
				Number:  orderNumber,
				Accrual: 50,
				Status:  model.OrderStatusProcessed,
			},
		},
		"err too many requests and than processed": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).
					Return(nil, application.ErrAccrualTooManyRequests).Once()
				accrual.On("GetOrder", mock.Anything, orderNumber).Return(&model.AccrualOrder{
					Status:  model.AccrualOrderStatusProcessed,
					Accrual: 50,
					Number:  orderNumber,
				}, nil).Once()
				return accrual
			}(),
			storageMock: func() *mocks.Storage {
				storageMock := mocks.NewStorage(t)
				storageMock.On("SetOrderStatusAndAccrual", mock.Anything, &storage.SetOrderStatusAndAccrual{
					ID:      orderID,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}).Once().Return(&model.Order{
					ID:      orderID,
					Number:  orderNumber,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}, nil)
				return storageMock
			}(),
			expectedResp: &model.Order{
				ID:      orderID,
				Number:  orderNumber,
				Accrual: 50,
				Status:  model.OrderStatusProcessed,
			},
		},
		"err order not registerd and than processed": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).
					Return(nil, application.ErrAccrualOrderNotRegistered).Once()
				accrual.On("GetOrder", mock.Anything, orderNumber).Return(&model.AccrualOrder{
					Status:  model.AccrualOrderStatusProcessed,
					Accrual: 50,
					Number:  orderNumber,
				}, nil).Once()
				return accrual
			}(),
			storageMock: func() *mocks.Storage {
				storageMock := mocks.NewStorage(t)
				storageMock.On("SetOrderStatusAndAccrual", mock.Anything, &storage.SetOrderStatusAndAccrual{
					ID:      orderID,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}).Once().Return(&model.Order{
					ID:      orderID,
					Number:  orderNumber,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}, nil)
				return storageMock
			}(),
			expectedResp: &model.Order{
				ID:      orderID,
				Number:  orderNumber,
				Accrual: 50,
				Status:  model.OrderStatusProcessed,
			},
		},
		"failed to set order status": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).Return(&model.AccrualOrder{
					Status: model.AccrualOrderStatusProcessing,
					Number: orderNumber,
				}, nil).Once()
				return accrual
			}(),
			storageMock: func() *mocks.Storage {
				storageMock := mocks.NewStorage(t)
				storageMock.On("SetOrderStatus", mock.Anything, &storage.SetOrderStatus{
					ID:     orderID,
					Status: model.OrderStatusProcessing,
				}).Once().Return(nil, errors.New("storage error"))
				return storageMock
			}(),
			expectedErr: fmt.Errorf("failed to update order in storage: %w", errors.New("storage error")),
		},
		"failed to set order status and accrual": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).Return(&model.AccrualOrder{
					Status:  model.AccrualOrderStatusProcessed,
					Accrual: 50,
					Number:  orderNumber,
				}, nil).Once()
				return accrual
			}(),
			storageMock: func() *mocks.Storage {
				storageMock := mocks.NewStorage(t)
				storageMock.On("SetOrderStatusAndAccrual", mock.Anything, &storage.SetOrderStatusAndAccrual{
					ID:      orderID,
					Status:  model.OrderStatusProcessed,
					Accrual: 50,
				}).Once().Return(nil, errors.New("storage error"))
				return storageMock
			}(),
			expectedErr: fmt.Errorf("failed to update order in storage: %w", errors.New("storage error")),
		},
		"unexpected error": {
			accrualMock: func() *mocks.AccrualAdapter {
				accrual := mocks.NewAccrualAdapter(t)
				accrual.On("GetOrder", mock.Anything, orderNumber).
					Return(nil, errors.New("accrual error"))
				return accrual
			}(),
			expectedErr: fmt.Errorf("failed to check order: %w", errors.New("accrual error")),
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			t.Parallel()

			s := NewService(tt.storageMock, nil, nil, tt.accrualMock, nil)

			res, err := s.checkOrderJob(orderID, orderNumber, 1*time.Millisecond)(context.Background())

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedResp, res)
		})
	}
}
