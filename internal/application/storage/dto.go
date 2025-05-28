package storage

import (
	"github.com/dtroode/gophermart/internal/application/model"
	"github.com/google/uuid"
)

type SetOrderAccrual struct {
	ID      uuid.UUID
	Accrual float32
}

type SetOrderStatus struct {
	ID     uuid.UUID
	Status model.OrderStatus
}

type SetOrderStatusAndAccrual struct {
	ID      uuid.UUID
	Status  model.OrderStatus
	Accrual float32
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
