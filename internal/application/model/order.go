package model

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatus represents the current status of an order
type OrderStatus string

// List of possible order statuses
const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)

// AccrualOrderStatus represents the status of an order in the accrual system
type AccrualOrderStatus string

// List of possible accrual order statuses
const (
	AccrualOrderStatusRegistered AccrualOrderStatus = "REGISTERED"
	AccrualOrderStatusInvalid    AccrualOrderStatus = "INVALID"
	AccrualOrderStatusProcessing AccrualOrderStatus = "PROCESSING"
	AccrualOrderStatusProcessed  AccrualOrderStatus = "PROCESSED"
)

// Order represents a user's order in the system
type Order struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
	Number    string
	Accrual   int32
	Status    OrderStatus
}

func NewOrder(
	userID uuid.UUID,
	number string,
) *Order {
	return &Order{
		UserID: userID,
		Number: number,
		Status: OrderStatusNew,
	}
}

type AccrualOrder struct {
	Number  string             `json:"order" binding:"required"`
	Status  AccrualOrderStatus `json:"status" binding:"required"`
	Accrual float32            `json:"accrual,omitempty"`
}
