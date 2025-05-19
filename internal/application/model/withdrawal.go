package model

import (
	"time"

	"github.com/google/uuid"
)

type Withdrawal struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	OrderID   uuid.UUID
	CreatedAt time.Time
	Amount    float32
}

type WithdrawalOrder struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	OrderID     uuid.UUID
	OrderNumber string
	CreatedAt   time.Time
	Amount      float32
}
