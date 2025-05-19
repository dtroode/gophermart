package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	Login     string
	Password  string
	CreatedAt time.Time
	Balance   float32
}
