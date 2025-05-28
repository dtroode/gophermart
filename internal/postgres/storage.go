package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/dtroode/gophermart/database"
	"github.com/dtroode/gophermart/internal/application"
	"github.com/dtroode/gophermart/internal/application/model"
	"github.com/dtroode/gophermart/internal/application/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db      *pgxpool.Pool
	queries *Queries
}

func NewStorage(dsn string) (*Storage, error) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize postgres connection: %w", err)
	}

	if err := database.Migrate(ctx, dsn); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &Storage{
		db:      pool,
		queries: New(pool),
	}, nil
}

func (s *Storage) Close() error {
	s.db.Close()

	return nil
}

func (s *Storage) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

func (s *Storage) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	dbUser, err := s.queries.GetUser(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}

	user := &model.User{
		ID:        dbUser.ID.Bytes,
		Login:     dbUser.Login,
		Password:  dbUser.Password,
		CreatedAt: dbUser.CreatedAt.Time,
		Balance:   dbUser.Balance.Int32, // Changed
	}

	return user, nil
}

func (s *Storage) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	dbUser, err := s.queries.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}

	user := &model.User{
		ID:        dbUser.ID.Bytes,
		Login:     dbUser.Login,
		Password:  dbUser.Password,
		CreatedAt: dbUser.CreatedAt.Time,
		Balance:   dbUser.Balance.Int32, // Changed
	}

	return user, nil
}

func (s *Storage) SaveUser(ctx context.Context, user *model.User) (*model.User, error) {
	params := SaveUserParams{
		Login:    user.Login,
		Password: user.Password,
	}

	dbUser, err := s.queries.SaveUser(ctx, params)
	if err != nil {
		return nil, err
	}

	user = &model.User{
		ID:        dbUser.ID.Bytes,
		Login:     dbUser.Login,
		Password:  dbUser.Password,
		CreatedAt: dbUser.CreatedAt.Time,
		Balance:   dbUser.Balance.Int32, // Changed
	}

	return user, nil
}

func (s *Storage) WithdrawUserBonuses(ctx context.Context, dto *storage.WithdrawUserBonuses) (*model.User, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	dbUser, err := qtx.SubstractUserBalance(ctx, SubstractUserBalanceParams{
		Balance: pgtype.Int4{Int32: int32(dto.Sum * 100), Valid: true}, // Changed
		ID:      pgtype.UUID{Bytes: dto.UserID, Valid: true},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.ConstraintName == "users_balance_nonnegative" {
				return nil, application.ErrNotEnoughBonuses
			}
		}
		return nil, err
	}

	_, err = qtx.CreateWithdrawal(ctx, CreateWithdrawalParams{
		UserID:   dbUser.ID,
		OrderNum: dto.OrderNum,
		Amount:   int32(dto.Sum * 100), // Changed
	})
	if err != nil {
		return nil, err
	}
	tx.Commit(ctx)

	user := &model.User{
		ID:        dbUser.ID.Bytes,
		Login:     dbUser.Login,
		CreatedAt: dbUser.CreatedAt.Time,
		Balance:   dbUser.Balance.Int32, // Changed
	}

	return user, nil
}

func (s *Storage) SetUserBalance(ctx context.Context, dto *storage.SetUserBalance) (*model.User, error) {
	params := SetUserBalanceParams{
		Balance: pgtype.Int4{Int32: int32(dto.Balance * 100), Valid: true}, // Changed
		ID:      pgtype.UUID{Bytes: dto.ID, Valid: true},
	}
	dbUser, err := s.queries.SetUserBalance(ctx, params)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:        dbUser.ID.Bytes,
		Login:     dbUser.Login,
		CreatedAt: dbUser.CreatedAt.Time,
		Balance:   dbUser.Balance.Int32, // Changed
	}

	return user, nil
}

func (s *Storage) IncrementUserBalance(ctx context.Context, dto *storage.IncrementUserBalance) (*model.User, error) {
	params := IncrementUserBalanceParams{
		Balance: pgtype.Int4{Int32: int32(dto.Sum * 100), Valid: true}, // Changed
		ID:      pgtype.UUID{Bytes: dto.ID, Valid: true},
	}
	dbUser, err := s.queries.IncrementUserBalance(ctx, params)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:        dbUser.ID.Bytes,
		Login:     dbUser.Login,
		CreatedAt: dbUser.CreatedAt.Time,
		Balance:   dbUser.Balance.Int32, // Changed
	}

	return user, nil
}

func (s *Storage) GetOrderByNumber(ctx context.Context, number string) (*model.Order, error) {
	dbOrder, err := s.queries.GetOrderByNumber(ctx, number)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}

	order := &model.Order{
		ID:        dbOrder.ID.Bytes,
		UserID:    dbOrder.UserID.Bytes,
		CreatedAt: dbOrder.CreatedAt.Time,
		Number:    dbOrder.Num,
		Accrual:   dbOrder.Accrual.Int32, // Changed
		Status:    model.OrderStatus(dbOrder.Status),
	}

	return order, nil
}

func (s *Storage) SaveOrder(ctx context.Context, order *model.Order) (*model.Order, error) {
	params := SaveOrderParams{
		UserID:  pgtype.UUID{Bytes: order.UserID, Valid: true},
		Num:     order.Number,
		Accrual: pgtype.Int4{Int32: order.Accrual, Valid: true}, // Changed
		Status:  OrderStatus(order.Status),
	}
	dbOrder, err := s.queries.SaveOrder(ctx, params)
	if err != nil {
		return nil, err
	}

	order = &model.Order{
		ID:        dbOrder.ID.Bytes,
		UserID:    dbOrder.UserID.Bytes,
		CreatedAt: dbOrder.CreatedAt.Time,
		Number:    dbOrder.Num,
		Accrual:   dbOrder.Accrual.Int32, // Changed
		Status:    model.OrderStatus(dbOrder.Status),
	}

	return order, nil
}

func (s *Storage) SetOrderAccrual(ctx context.Context, dto *storage.SetOrderAccrual) (*model.Order, error) {
	params := SetOrderAccrualParams{
		Accrual: pgtype.Int4{Int32: int32(dto.Accrual * 100), Valid: true}, // Changed
		ID:      pgtype.UUID{Bytes: dto.ID, Valid: true},
	}
	dbOrder, err := s.queries.SetOrderAccrual(ctx, params)
	if err != nil {
		return nil, err
	}

	order := &model.Order{
		ID:        dbOrder.ID.Bytes,
		UserID:    dbOrder.UserID.Bytes,
		CreatedAt: dbOrder.CreatedAt.Time,
		Number:    dbOrder.Num,
		Accrual:   dbOrder.Accrual.Int32, // Changed
		Status:    model.OrderStatus(dbOrder.Status),
	}

	return order, nil
}

func (s *Storage) SetOrderStatus(ctx context.Context, dto *storage.SetOrderStatus) (*model.Order, error) {
	params := SetOrderStatusParams{
		Status: OrderStatus(dto.Status),
		ID:     pgtype.UUID{Bytes: dto.ID, Valid: true},
	}
	dbOrder, err := s.queries.SetOrderStatus(ctx, params)
	if err != nil {
		return nil, err
	}

	order := &model.Order{
		ID:        dbOrder.ID.Bytes,
		UserID:    dbOrder.UserID.Bytes,
		CreatedAt: dbOrder.CreatedAt.Time,
		Number:    dbOrder.Num,
		Accrual:   dbOrder.Accrual.Int32, // Changed
		Status:    model.OrderStatus(dbOrder.Status),
	}

	return order, nil
}

func (s *Storage) SetOrderStatusAndAccrual(ctx context.Context, dto *storage.SetOrderStatusAndAccrual) (*model.Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	statusParams := SetOrderStatusParams{
		Status: OrderStatus(dto.Status),
		ID:     pgtype.UUID{Bytes: dto.ID, Valid: true},
	}
	_, err = qtx.SetOrderStatus(ctx, statusParams)
	if err != nil {
		return nil, err
	}

	accrualParams := SetOrderAccrualParams{
		Accrual: pgtype.Int4{Int32: int32(dto.Accrual * 100), Valid: true}, // Changed
		ID:      pgtype.UUID{Bytes: dto.ID, Valid: true},
	}
	dbOrder, err := qtx.SetOrderAccrual(ctx, accrualParams)
	if err != nil {
		return nil, err
	}

	userParams := IncrementUserBalanceParams{
		Balance: pgtype.Int4{Int32: int32(dto.Accrual * 100), Valid: true}, // Changed
		ID:      dbOrder.UserID,
	}
	_, err = qtx.IncrementUserBalance(ctx, userParams)
	if err != nil {
		return nil, err
	}

	tx.Commit(ctx)

	order := &model.Order{
		ID:        dbOrder.ID.Bytes,
		UserID:    dbOrder.UserID.Bytes,
		CreatedAt: dbOrder.CreatedAt.Time,
		Number:    dbOrder.Num,
		Accrual:   dbOrder.Accrual.Int32, // Changed
		Status:    model.OrderStatus(dbOrder.Status),
	}

	return order, nil
}

func (s *Storage) GetUserOrdersNewestFirst(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	dbOrders, err := s.queries.GetUserOrdersNewestFirst(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	orders := make([]*model.Order, len(dbOrders))

	for i, dbOrder := range dbOrders {
		order := &model.Order{
			ID:        dbOrder.ID.Bytes,
			UserID:    dbOrder.UserID.Bytes,
			CreatedAt: dbOrder.CreatedAt.Time,
			Number:    dbOrder.Num,
			Accrual:   dbOrder.Accrual.Int32, // Changed
			Status:    model.OrderStatus(dbOrder.Status),
		}
		orders[i] = order
	}

	return orders, nil
}

func (s *Storage) GetUserWithdrawalSum(ctx context.Context, userID uuid.UUID) (int32, error) { // Changed signature
	dbSum, err := s.queries.GetUserWithdrawalSum(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return 0, err
	}
	// dbSum is int64 as SUM() on INTEGER results in BIGINT.
	// Perform type assertion to int64 first, then convert to int32.
	if sum, ok := dbSum.(int64); ok {
		return int32(sum), nil
	}
	// If the type assertion fails, it might be zero or another numeric type.
	// Handle cases where dbSum might be other types if necessary, or return an error.
	// For now, assuming it's int64 or we return 0 if not.
	// This part might need more robust error handling or type checking depending on actual DB returns.
	if sum, ok := dbSum.(int32); ok { // Fallback for direct int32, though sum is usually int64
	    return sum, nil
    }
	return 0, fmt.Errorf("unexpected type for sum: %T", dbSum)
}

func (s *Storage) GetUserWithdrawals(ctx context.Context, userID uuid.UUID) ([]*model.WithdrawalOrder, error) {
	dbWithdrawals, err := s.queries.GetUserWithdrawals(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	withdrawals := make([]*model.WithdrawalOrder, len(dbWithdrawals))

	for i, withdrawal := range dbWithdrawals {
		withdrawals[i] = &model.WithdrawalOrder{
			ID:          withdrawal.ID.Bytes,
			UserID:      withdrawal.UserID.Bytes,
			OrderNumber: withdrawal.OrderNum,
			CreatedAt:   withdrawal.CreatedAt.Time,
			Amount:      withdrawal.Amount, // Changed (assuming withdrawal.Amount is int32 from sqlc)
		}
	}

	return withdrawals, nil
}
