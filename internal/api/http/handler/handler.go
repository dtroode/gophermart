package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/dtroode/gophermart/internal/api/http/request"
	"github.com/dtroode/gophermart/internal/application"
	"github.com/dtroode/gophermart/internal/application/model"
	dto "github.com/dtroode/gophermart/internal/application/request"
	"github.com/dtroode/gophermart/internal/application/response"
	"github.com/dtroode/gophermart/internal/auth"
	"github.com/dtroode/gophermart/internal/logger"
	"github.com/google/uuid"
)

type Service interface {
	RegisterUser(ctx context.Context, dto *dto.RegisterUser) (string, error)
	Login(ctx context.Context, dto *dto.Login) (string, error)
	UploadOrder(ctx context.Context, dto *dto.UploadOrder) (*model.Order, error)
	ListUserOrders(ctx context.Context, id uuid.UUID) ([]*response.UserOrder, error)
	GetUserBalance(ctx context.Context, id uuid.UUID) (*response.UserBalance, error)
	WithdrawUserBonuses(ctx context.Context, dto *dto.WithdrawBonuses) error
	ListUserWithdrawals(ctx context.Context, id uuid.UUID) ([]*response.UserWithdrawal, error)
}

type Handler struct {
	service Service
	logger  *logger.Logger
}

func New(s Service, l *logger.Logger) *Handler {
	return &Handler{
		service: s,
		logger:  l,
	}
}

// RegisterUser godoc
// @Summary Register new user
// @Description Register a new user in the system
// @Tags auth
// @Accept json
// @Produce json
// @Param request body request.RegisterUser true "User registration details"
// @Success 200 {string} string "Bearer token in Authorization header"
// @Failure 400 {string} string "Invalid input"
// @Failure 409 {string} string "User already exists"
// @Failure 500 {string} string "Internal server error"
// @Router /user/register [post]
func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &request.RegisterUser{}

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := h.service.RegisterUser(ctx, &dto.RegisterUser{
		Login:    req.Login,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, application.ErrConflict) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		h.logger.Error("failed to register user", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("authorization", fmt.Sprintf("Bearer %s", token))
	w.WriteHeader(http.StatusOK)
}

// Login godoc
// @Summary Login user
// @Description Authenticate existing user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body request.Login true "User login credentials"
// @Success 200 {string} string "Bearer token in Authorization header"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /user/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &request.Login{}

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(ctx, &dto.Login{
		Login:    req.Login,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, application.ErrUnauthorized) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.logger.Error("failed to login user", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("authorization", fmt.Sprintf("Bearer %s", token))
	w.WriteHeader(http.StatusOK)
}

// UploadOrder godoc
// @Summary Upload order
// @Description Upload a new order for the authenticated user
// @Tags orders
// @Accept text/plain
// @Produce json
// @Security Bearer
// @Param order body string true "Order number"
// @Success 202 {string} string "Order accepted"
// @Success 200 {string} string "Order already exists"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 409 {string} string "Order registered by another user"
// @Failure 422 {string} string "Invalid order number"
// @Failure 500 {string} string "Internal server error"
// @Router /user/orders [post]
func (h *Handler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		h.logger.Error("failed to get user id from context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	_, err = h.service.UploadOrder(ctx, &dto.UploadOrder{
		UserID:      userID,
		OrderNumber: string(body),
	})
	if err != nil {
		if errors.Is(err, application.ErrAlreadyExist) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, application.ErrConflict) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		if errors.Is(err, application.ErrUnprocessable) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		h.logger.Error("failed to upload order", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// ListUserOrders godoc
// @Summary List user orders
// @Description Get all orders for the authenticated user
// @Tags orders
// @Produce json
// @Security Bearer
// @Success 200 {array} response.UserOrder
// @Success 204 {string} string "No orders found"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /user/orders [get]
func (h *Handler) ListUserOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		h.logger.Error("failed to get user id from context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	orders, err := h.service.ListUserOrders(ctx, userID)
	if err != nil {
		if errors.Is(err, application.ErrNoData) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.logger.Error("failed to get user orders", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(orders); err != nil {
		h.logger.Error("failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// GetUserBalance godoc
// @Summary Get user balance
// @Description Get current balance for the authenticated user
// @Tags balance
// @Produce json
// @Security Bearer
// @Success 200 {object} response.UserBalance
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /user/balance [get]
func (h *Handler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		h.logger.Error("failed to get user id from context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	balance, err := h.service.GetUserBalance(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get user balance", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(balance); err != nil {
		h.logger.Error("failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// WithdrawUserBonuses godoc
// @Summary Withdraw user bonuses
// @Description Withdraw bonuses for the authenticated user
// @Tags balance
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body request.WithdrawBonuses true "Withdrawal details"
// @Success 200 {string} string "Withdrawal successful"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 402 {string} string "Not enough bonuses"
// @Failure 422 {string} string "Invalid order number"
// @Failure 500 {string} string "Internal server error"
// @Router /user/balance/withdraw [post]
func (h *Handler) WithdrawUserBonuses(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		h.logger.Error("failed to get user id from context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	req := &request.WithdrawBonuses{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.service.WithdrawUserBonuses(ctx, &dto.WithdrawBonuses{
		UserID:      userID,
		OrderNumber: req.Order,
		Sum:         req.Sum,
	}); err != nil {
		if errors.Is(err, application.ErrNotEnoughBonuses) {
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		if errors.Is(err, application.ErrUnprocessable) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		h.logger.Error("failed to withdraw bonuses", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ListUserWithdrawals godoc
// @Summary List user withdrawals
// @Description Get all withdrawals for the authenticated user
// @Tags balance
// @Produce json
// @Security Bearer
// @Success 200 {array} response.UserWithdrawal
// @Success 204 {string} string "No withdrawals found"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Router /user/withdrawals [get]
func (h *Handler) ListUserWithdrawals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok {
		h.logger.Error("failed to get user id from context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	withdrawals, err := h.service.ListUserWithdrawals(ctx, userID)
	if err != nil {
		if errors.Is(err, application.ErrNoData) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.logger.Error("failed to list user withdrawals", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(withdrawals); err != nil {
		h.logger.Error("failed to encode response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
