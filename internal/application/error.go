package application

import "errors"

var ErrNoData = errors.New("no data")
var ErrNotFound = errors.New("not found")
var ErrUnauthorized = errors.New("unauthorized")
var ErrConflict = errors.New("conflict")
var ErrAlreadyExist = errors.New("already exist")
var ErrUnprocessable = errors.New("unprocessable entity")
var ErrNotEnoughBonuses = errors.New("not enough bonuses")

var ErrAccrualOrderNotRegistered = errors.New("order is not registered")
var ErrAccrualTooManyRequests = errors.New("too many requests")
var ErrAccrualInternal = errors.New("internal service error")
