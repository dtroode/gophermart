package request

import "github.com/google/uuid"

type RegisterUser struct {
	Login    string
	Password string
}

type Login struct {
	Login    string
	Password string
}

type UploadOrder struct {
	UserID      uuid.UUID
	OrderNumber string
}

type WithdrawBonuses struct {
	UserID      uuid.UUID
	OrderNumber string
	Sum         float32
}
