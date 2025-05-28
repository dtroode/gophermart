package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/dtroode/gophermart/internal/application"
	"github.com/dtroode/gophermart/internal/application/model"
	"github.com/dtroode/gophermart/internal/application/request"
	"github.com/dtroode/gophermart/internal/application/response"
	"github.com/dtroode/gophermart/internal/application/storage"
	"github.com/dtroode/gophermart/internal/workerpool"
	"github.com/google/uuid"
)

type Storage interface {
	GetUser(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
	SaveUser(ctx context.Context, user *model.User) (*model.User, error)
	WithdrawUserBonuses(ctx context.Context, dto *storage.WithdrawUserBonuses) (*model.User, error)
	GetOrderByNumber(ctx context.Context, number string) (*model.Order, error)
	SaveOrder(ctx context.Context, order *model.Order) (*model.Order, error)
	SetOrderStatus(ctx context.Context, dto *storage.SetOrderStatus) (*model.Order, error)
	SetOrderStatusAndAccrual(ctx context.Context, dto *storage.SetOrderStatusAndAccrual) (*model.Order, error)
	IncrementUserBalance(ctx context.Context, dto *storage.IncrementUserBalance) (*model.User, error)
	GetUserWithdrawalSum(ctx context.Context, userID uuid.UUID) (int32, error) // Changed
	GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]*model.WithdrawalOrder, error)
	GetUserOrdersNewestFirst(ctx context.Context, userID uuid.UUID) ([]*model.Order, error)
}

type Hasher interface {
	Hash(ctx context.Context, password []byte) (string, error)
}

type TokenManager interface {
	CreateToken(userID uuid.UUID) (string, error)
}

type AccrualAdapter interface {
	GetOrder(ctx context.Context, orderNumber string) (*model.AccrualOrder, error)
}

type WorkerPool interface {
	Submit(ctx context.Context, timeout time.Duration, fn func(ctx context.Context) (any, error), expectResult bool) chan *workerpool.Result
}

type Service struct {
	storage        Storage
	hasher         Hasher
	tokenManager   TokenManager
	accrualAdapter AccrualAdapter
	pool           WorkerPool
	sync.Mutex
}

func NewService(
	storage Storage,
	hasher Hasher,
	tokenManager TokenManager,
	accrualAdapter AccrualAdapter,
	pool WorkerPool,
) *Service {
	return &Service{
		storage:        storage,
		hasher:         hasher,
		tokenManager:   tokenManager,
		accrualAdapter: accrualAdapter,
		pool:           pool,
	}
}

func (s *Service) RegisterUser(ctx context.Context, params *request.RegisterUser) (string, error) {
	user, err := s.storage.GetUserByLogin(ctx, params.Login)
	if err != nil {
		if !errors.Is(err, application.ErrNotFound) {
			return "", fmt.Errorf("failed to check user with login: %w", err)
		}
	}

	if err == nil && user != nil {
		return "", application.ErrConflict
	}

	hash, err := s.hasher.Hash(ctx, []byte(params.Password))
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	user = &model.User{
		Login:    params.Login,
		Password: hash,
	}

	user, err = s.storage.SaveUser(ctx, user)
	if err != nil {
		return "", fmt.Errorf("failed to save user: %w", err)
	}

	token, err := s.tokenManager.CreateToken(user.ID)
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	return token, nil
}

func (s *Service) Login(ctx context.Context, params *request.Login) (string, error) {
	user, err := s.storage.GetUserByLogin(ctx, params.Login)
	if err != nil {
		if errors.Is(err, application.ErrNotFound) {
			return "", application.ErrUnauthorized
		}
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	hash, err := s.hasher.Hash(ctx, []byte(params.Password))
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	if user.Password != hash {
		return "", application.ErrUnauthorized
	}

	token, err := s.tokenManager.CreateToken(user.ID)
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	return token, nil
}

func (s *Service) checkByLuhn(number string) error {
	start := len(number) % 2
	control := make([]int, 0)

	for i, charRune := range number {
		sint, err := strconv.Atoi(string(charRune))
		if err != nil {
			return fmt.Errorf("failed to convert string character to number: %w", err)
		}
		if (start+i)%2 == 0 {
			sint = sint * 2
			if sint > 9 {
				sint = sint - 9 // Corrected Luhn algorithm for numbers > 9
			}
		}
		control = append(control, sint)
	}

	var sum int

	for _, s := range control {
		sum += s
	}

	if sum%10 != 0 {
		return fmt.Errorf("not valid by luhn")
	}

	return nil
}

func (s *Service) UploadOrder(ctx context.Context, params *request.UploadOrder) (*model.Order, error) {
	if err := s.checkByLuhn(params.OrderNumber); err != nil {
		return nil, application.ErrUnprocessable
	}

	order, err := s.storage.GetOrderByNumber(ctx, params.OrderNumber)
	if err != nil && !errors.Is(err, application.ErrNotFound) {
		return nil, fmt.Errorf("failed to check order exists: %w", err)
	}

	if err == nil {
		if order.UserID == params.UserID {
			return nil, application.ErrAlreadyExist
		} else {
			return nil, application.ErrConflict
		}
	}

	order = model.NewOrder(params.UserID, params.OrderNumber)
	order, err = s.storage.SaveOrder(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	s.pool.Submit(context.Background(), 1*time.Hour, s.checkOrderJob(order.ID, order.Number, 1*time.Second), false)

	return order, nil
}

func (s *Service) ListUserOrders(ctx context.Context, id uuid.UUID) ([]*response.UserOrder, error) {
	orders, err := s.storage.GetUserOrdersNewestFirst(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}

	if len(orders) == 0 {
		return nil, application.ErrNoData
	}

	resp := make([]*response.UserOrder, len(orders))

	for i, order := range orders {
		resp[i] = &response.UserOrder{
			Number:     order.Number,
			Status:     string(order.Status),
			Accrual:    float32(order.Accrual) / 100.0, // Changed
			UploadedAt: order.CreatedAt.Format(time.RFC3339),
		}
	}

	return resp, nil
}

func (s *Service) GetUserBalance(ctx context.Context, id uuid.UUID) (*response.UserBalance, error) {
	user, err := s.storage.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, application.ErrNotFound) {
			return nil, application.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	withdrawalSum, err := s.storage.GetUserWithdrawalSum(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user withdrawal sum: %w", err)
	}

	resp := &response.UserBalance{
		Current:   float32(user.Balance) / 100.0,   // Changed
		Withdrawn: float32(withdrawalSum) / 100.0, // Changed
	}

	return resp, nil
}

func (s *Service) WithdrawUserBonuses(ctx context.Context, params *request.WithdrawBonuses) error {
	if err := s.checkByLuhn(params.OrderNumber); err != nil {
		return application.ErrUnprocessable
	}

	_, err := s.storage.WithdrawUserBonuses(ctx, &storage.WithdrawUserBonuses{
		UserID:   params.UserID,
		OrderNum: params.OrderNumber,
		Sum:      params.Sum,
	})
	if err != nil {
		if errors.Is(err, application.ErrNotFound) {
			return err
		}
		if errors.Is(err, application.ErrNotEnoughBonuses) {
			return err
		}
		return fmt.Errorf("failed to withdraw user bonuses: %w", err)
	}

	return nil
}

func (s *Service) ListUserWithdrawals(ctx context.Context, id uuid.UUID) ([]*response.UserWithdrawal, error) {
	withdrawals, err := s.storage.GetUserWithdrawals(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user withdrawals: %w", err)
	}

	if len(withdrawals) == 0 {
		return nil, application.ErrNoData
	}

	resp := make([]*response.UserWithdrawal, len(withdrawals))

	for i, withdrawal := range withdrawals {
		resp[i] = &response.UserWithdrawal{
			OrderNumber: withdrawal.OrderNumber,
			Sum:         float32(withdrawal.Amount) / 100.0, // Changed
			ProcessedAt: withdrawal.CreatedAt.Format(time.RFC3339),
		}
	}

	return resp, nil
}
