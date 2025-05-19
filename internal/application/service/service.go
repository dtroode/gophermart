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

type Hasher interface {
	Hash(ctx context.Context, password []byte) (string, error)
}

type TokenManager interface {
	CreateToken(ctx context.Context, userID uuid.UUID) (string, error)
}

type AccrualAdapter interface {
	GetOrder(ctx context.Context, orderNumber string) (*model.AccrualOrder, error)
}

type WorkerPool interface {
	Submit(ctx context.Context, timeout time.Duration, fn func(ctx context.Context) (any, error), expectResult bool) chan *workerpool.Result
}

type Service struct {
	storage        storage.Storage
	hasher         Hasher
	tokenManager   TokenManager
	accrualAdapter AccrualAdapter
	pool           WorkerPool
	sync.Mutex
}

func NewService(
	storage storage.Storage,
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
	_, err := s.storage.GetUserByLogin(ctx, params.Login)
	if err != nil {
		if !errors.Is(err, application.ErrNotFound) {
			return "", fmt.Errorf("failed to check user with login: %w", err)
		}
	} else {
		return "", application.ErrConflict
	}

	hash, err := s.hasher.Hash(ctx, []byte(params.Password))
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Login:    params.Login,
		Password: hash,
	}

	user, err = s.storage.SaveUser(ctx, user)
	if err != nil {
		return "", fmt.Errorf("failed to save user: %w", err)
	}

	token, err := s.tokenManager.CreateToken(ctx, user.ID)
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

	token, err := s.tokenManager.CreateToken(ctx, user.ID)
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	return token, nil
}

func (s *Service) checkByLuna(number string) error {
	start := len(number) % 2
	control := make([]int, 0)

	for i, s := range number {
		sint, err := strconv.Atoi(string(s))
		if err != nil {
			return fmt.Errorf("failed to convert string character to number: %w", err)
		}
		if (start+i)%2 == 0 {
			sint = sint * 2
			if sint > 9 {
				sint = sint % 9
			}
		}
		control = append(control, sint)
	}

	var sum int

	for _, s := range control {
		sum += s
	}

	if sum%10 != 0 {
		return fmt.Errorf("not valid by luna")
	}

	return nil
}

func (s *Service) UploadOrder(ctx context.Context, params *request.UploadOrder) (*model.Order, error) {
	if err := s.checkByLuna(params.OrderNumber); err != nil {
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

	s.pool.Submit(context.Background(), 1*time.Hour, s.checkOrderJob(order.ID, order.Number), false)

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
			Accrual:    float32(order.Accrual),
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
		Current:   user.Balance,
		Withdrawn: withdrawalSum,
	}

	return resp, nil
}

func (s *Service) WithdrawUserBonuses(ctx context.Context, params *request.WithdrawBonuses) error {
	if err := s.checkByLuna(params.OrderNumber); err != nil {
		return application.ErrUnprocessable
	}

	// INSERT_YOUR_CODE
	// Round the sum to the nearest integer (up or down as needed)
	// If you want to always round down: use math.Floor
	// If you want to always round up: use math.Ceil
	// If you want to round to the nearest integer: use math.Round

	// Example: round to nearest integer
	// params.Sum = float32(int(params.Sum + 0.5))

	// If you want to always round down, use:
	// params.Sum = float32(int(params.Sum))

	// If you want to always round up, use:
	// if float32(int(params.Sum)) < params.Sum {
	//     params.Sum = float32(int(params.Sum) + 1)
	// }

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
			Sum:         withdrawal.Amount,
			ProcessedAt: withdrawal.CreatedAt.Format(time.RFC3339),
		}
	}

	return resp, nil
}
