package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dtroode/gophermart/internal/application"
	"github.com/dtroode/gophermart/internal/application/model"
	"github.com/dtroode/gophermart/internal/application/storage"
	"github.com/google/uuid"
)

func (s *Service) checkOrderJob(orderID uuid.UUID, orderNumber string) func(ctx context.Context) (any, error) {
	return func(ctx context.Context) (any, error) {
		c := time.Tick(1 * time.Second)
		currentStatus := model.AccrualOrderStatusRegistered
		for {
			select {
			case <-c:
				order, err := s.accrualAdapter.GetOrder(ctx, orderNumber)
				if err != nil {
					if errors.Is(err, application.ErrAccrualTooManyRequests) {
						continue
					}
					if errors.Is(err, application.ErrAccrualOrderNotRegistered) {
						continue
					}

					return nil, fmt.Errorf("failed to check order: %w", err)
				}

				if order.Status == model.AccrualOrderStatusProcessing {
					if currentStatus != order.Status {
						params := &storage.SetOrderStatus{
							ID:     orderID,
							Status: model.OrderStatusProcessing,
						}
						_, err := s.storage.SetOrderStatus(ctx, params)
						if err != nil {
							return nil, fmt.Errorf("failed to update order in storage: %w", err)
						}
						currentStatus = order.Status
					}

					continue
				}

				if order.Status == model.AccrualOrderStatusInvalid {
					params := &storage.SetOrderStatus{
						ID:     orderID,
						Status: model.OrderStatusInvalid,
					}
					updatedOrder, err := s.storage.SetOrderStatus(ctx, params)
					if err != nil {
						return nil, fmt.Errorf("failed to update order in storage: %w", err)
					}

					return updatedOrder, nil
				}

				if order.Status == model.AccrualOrderStatusProcessed {
					params := &storage.SetOrderStatusAndAccrual{
						OrderID:     orderID,
						OrderStatus: model.OrderStatusProcessed,
						Accrual:     order.Accrual,
					}
					updatedOrder, err := s.storage.SetOrderStatusAndAccrual(ctx, params)
					if err != nil {
						return nil, fmt.Errorf("failed to update order in storage: %w", err)
					}

					return updatedOrder, nil
				}
			case <-ctx.Done():
				return nil, fmt.Errorf("failed to check order, context deadline exceeded")
			}
		}
	}
}
