package storage

import (
	"github.com/dtroode/gophermart/internal/application/model"
	"github.com/google/uuid"
)

type SetOrderAccrual struct {
	ID      uuid.UUID
	Accrual int32
}

type SetOrderStatus struct {
	ID     uuid.UUID
	Status model.OrderStatus
}

type SetOrderStatusAndAccrual struct {
	ID      uuid.UUID
	Status  model.OrderStatus
	Accrual int32
}

type SetUserBalance struct {
	ID      uuid.UUID
	Balance int32
}

type IncrementUserBalance struct {
	ID  uuid.UUID
	Sum int32
}

type WithdrawUserBonuses struct {
	UserID   uuid.UUID
	OrderNum string
	Sum      int32
}
