package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/dtroode/gophermart/internal/application"
	"github.com/dtroode/gophermart/internal/application/model"
)

type Adapter struct {
	endpoint string
}

func NewAdapter(endpoint string) *Adapter {
	return &Adapter{
		endpoint: endpoint,
	}
}

func (a *Adapter) GetOrder(ctx context.Context, orderNumber string) (*model.AccrualOrder, error) {
	u, err := url.JoinPath(a.endpoint, "/api/orders/", orderNumber)
	fmt.Printf("[ACCRUAL] path: %s\n", u)
	if err != nil {
		return nil, fmt.Errorf("failed to join path: %w", err)
	}

	resp, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to request order: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		order := &model.AccrualOrder{}
		if err := json.NewDecoder(resp.Body).Decode(order); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return order, nil
	case http.StatusNoContent:
		return nil, application.ErrAccrualOrderNotRegistered
	case http.StatusTooManyRequests:
		return nil, application.ErrAccrualTooManyRequests
	default:
		return nil, application.ErrAccrualInternal
	}
}
