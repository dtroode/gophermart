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

func (s *Service) isRetryableError(err error) bool {
	return errors.Is(err, application.ErrAccrualTooManyRequests) || errors.Is(err, application.ErrAccrualOrderNotRegistered)
}

func (s *Service) updateOrderStatus(ctx context.Context, id uuid.UUID, status model.OrderStatus) (*model.Order, error) {
	params := &storage.SetOrderStatus{
		ID:     id,
		Status: status,
	}
	order, err := s.storage.SetOrderStatus(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update order in storage: %w", err)
	}

	return order, nil
}

func (s *Service) finalizeInvalidOrder(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	params := &storage.SetOrderStatus{
		ID:     id,
		Status: model.OrderStatusInvalid,
	}
	updatedOrder, err := s.storage.SetOrderStatus(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update order in storage: %w", err)
	}

	return updatedOrder, nil
}

func (s *Service) finalizeProcessedOrder(ctx context.Context, orderID uuid.UUID, accrual float32) (any, error) {
	params := &storage.SetOrderStatusAndAccrual{
		ID:      orderID,
		Status:  model.OrderStatusProcessed,
		Accrual: int32(accrual * 100.0),
	}
	order, err := s.storage.SetOrderStatusAndAccrual(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update order in storage: %w", err)
	}
	return order, nil
}

func (s *Service) checkOrderJob(orderID uuid.UUID, orderNumber string, duration time.Duration) func(ctx context.Context) (any, error) {
	return func(ctx context.Context) (any, error) {
		c := time.Tick(duration)
		currentStatus := model.AccrualOrderStatusRegistered
		for {
			select {
			case <-c:
				order, err := s.accrualAdapter.GetOrder(ctx, orderNumber)
				if err != nil {
					if s.isRetryableError(err) {
						continue
					}

					return nil, fmt.Errorf("failed to check order: %w", err)
				}

				switch order.Status {
				case model.AccrualOrderStatusProcessing:
					if currentStatus != order.Status {
						if _, err := s.updateOrderStatus(ctx, orderID, model.OrderStatusProcessing); err != nil {
							return nil, err
						}
						currentStatus = order.Status
					}
					continue
				case model.AccrualOrderStatusInvalid:
					return s.finalizeInvalidOrder(ctx, orderID)
				case model.AccrualOrderStatusProcessed:
					return s.finalizeProcessedOrder(ctx, orderID, order.Accrual)
				default:
					continue
				}
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					return nil, fmt.Errorf("ctx is closed: %w", err)
				}
				return nil, fmt.Errorf("failed to get reason of context close")
			}
		}
	}
}
