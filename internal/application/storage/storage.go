package storage

import (
	"context"

	"github.com/dtroode/gophermart/internal/application/model"
	"github.com/google/uuid"
)

type Storage interface {
	GetUser(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
	SaveUser(ctx context.Context, user *model.User) (*model.User, error)
	WithdrawUserBonuses(ctx context.Context, dto *WithdrawUserBonuses) (*model.User, error)
	GetOrderByNumber(ctx context.Context, number string) (*model.Order, error)
	SaveOrder(ctx context.Context, order *model.Order) (*model.Order, error)
	SetOrderStatus(ctx context.Context, dto *SetOrderStatus) (*model.Order, error)
	SetOrderStatusAndAccrual(ctx context.Context, dto *SetOrderStatusAndAccrual) (*model.Order, error)
	IncrementUserBalance(ctx context.Context, dto *IncrementUserBalance) (*model.User, error)
	GetUserWithdrawalSum(ctx context.Context, userID uuid.UUID) (float32, error)
	GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]*model.WithdrawalOrder, error)
	GetUserOrdersNewestFirst(ctx context.Context, userID uuid.UUID) ([]*model.Order, error)
}

type SetOrderAccrual struct {
	ID      uuid.UUID
	Accrual float32
}

type SetOrderStatus struct {
	ID     uuid.UUID
	Status model.OrderStatus
}

type SetOrderStatusAndAccrual struct {
	OrderID     uuid.UUID
	OrderStatus model.OrderStatus
	Accrual     float32
}

type SetUserBalance struct {
	ID      uuid.UUID
	Balance float32
}

type IncrementUserBalance struct {
	ID  uuid.UUID
	Sum float32
}

type WithdrawUserBonuses struct {
	UserID   uuid.UUID
	OrderNum string
	Sum      float32
}
