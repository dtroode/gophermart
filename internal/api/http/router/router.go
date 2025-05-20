package router

import (
	"github.com/dtroode/gophermart/internal/api/http/handler"
	"github.com/dtroode/gophermart/internal/api/http/middleware"
	"github.com/dtroode/gophermart/internal/application/service"
	"github.com/dtroode/gophermart/internal/logger"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Router struct {
	chi.Router
}

func NewRouter() *Router {
	return &Router{
		Router: chi.NewRouter(),
	}
}

func (r *Router) RegisterRoutes(s *service.Service, token middleware.TokenManager, l *logger.Logger) {
	loggerMiddleware := middleware.NewRequestLog(l).Handle
	authenticate := middleware.NewAuthenticate(token, l).Handle
	degzipper := middleware.Decompress
	compressor := chiMiddleware.Compress(5)

	h := handler.New(s, l)

	// Swagger UI endpoint
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The url pointing to API definition
	))

	r.Route("/api/user", func(r chi.Router) {
		r.Use(loggerMiddleware)
		r.Use(degzipper)
		r.Use(compressor)

		r.Post("/register", h.RegisterUser)
		r.Post("/login", h.Login)

		r.Group(func(r chi.Router) {
			r.Use(authenticate)
			r.Post("/orders", h.UploadOrder)
			r.Get("/orders", h.ListUserOrders)
			r.Get("/balance", h.GetUserBalance)
			r.Post("/balance/withdraw", h.WithdrawUserBonuses)
			r.Get("/withdrawals", h.ListUserWithdrawals)
		})
	})
}
