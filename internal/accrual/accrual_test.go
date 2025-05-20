package accrual_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dtroode/gophermart/internal/accrual"
	"github.com/dtroode/gophermart/internal/application"
	"github.com/dtroode/gophermart/internal/application/model"
	"github.com/stretchr/testify/assert"
)

func TestAdapter_GetOrder(t *testing.T) {
	tests := map[string]struct {
		accrualHandler http.Handler
		expectedResp   *model.AccrualOrder
		expectedErr    error
	}{
		"accrual OK processing": {
			accrualHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("content-type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"order": "1234", "status": "PROCESSING"}`))
			}),
			expectedResp: &model.AccrualOrder{
				Number: "1234",
				Status: model.AccrualOrderStatusProcessing,
			},
		},
		"accrual OK processed": {
			accrualHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("content-type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"order": "1234", "status": "PROCESSED", "accrual": 50}`))
			}),
			expectedResp: &model.AccrualOrder{
				Number:  "1234",
				Status:  model.AccrualOrderStatusProcessed,
				Accrual: 50,
			},
		},
		"accrual error order not registered": {
			accrualHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}),
			expectedErr: application.ErrAccrualOrderNotRegistered,
		},
		"accrual error too many requests": {
			accrualHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			}),
			expectedErr: application.ErrAccrualTooManyRequests,
		},
		"accrual error internal": {
			accrualHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			}),
			expectedErr: application.ErrAccrualInternal,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			httpserver := httptest.NewServer(tt.accrualHandler)

			a := accrual.NewAdapter(httpserver.URL)

			res, err := a.GetOrder(context.Background(), "1234")

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedResp, res)
		})
	}
}
