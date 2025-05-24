package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/dtroode/gophermart/config"
	_ "github.com/dtroode/gophermart/docs" // swagger docs
	"github.com/dtroode/gophermart/internal/accrual"
	"github.com/dtroode/gophermart/internal/api/http/router"
	"github.com/dtroode/gophermart/internal/application/service"
	"github.com/dtroode/gophermart/internal/auth"
	"github.com/dtroode/gophermart/internal/logger"
	"github.com/dtroode/gophermart/internal/postgres"
	"github.com/dtroode/gophermart/internal/workerpool"
)

// @title           GopherMart API
// @version         1.0
// @description     A loyalty points service for an online marketplace where users can register orders and receive bonuses.

// @contact.name   API Support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	cfg, err := config.Initialize()
	if err != nil {
		log.Fatal(err)
	}

	log := logger.NewLog(cfg.LogLevel)

	log.Info("config: ", "config", cfg)

	store, err := postgres.NewStorage(cfg.DatabaseDSN)
	if err != nil {
		log.Error("failed to start db conn", "error", err)
		os.Exit(1)
	}

	argon := auth.NewArgon2Id(
		[]byte(cfg.ArgonSalt),
		uint32(cfg.ArgonTime),
		uint32(cfg.ArgonMemory),
		uint8(cfg.ArgonThreads),
		uint32(cfg.ArgonKeyLen),
	)

	jwt := auth.NewJWT(cfg.JWTSecretKey)

	accrualAdapter := accrual.NewAdapter(cfg.AccrualAddr)

	pool := workerpool.NewPool(cfg.ConcurrencyLimit, cfg.QueueSize)
	pool.Start()

	srv := service.NewService(store, argon, jwt, accrualAdapter, pool)

	r := router.NewRouter()

	r.RegisterRoutes(srv, jwt, log)

	go func() {
		log.Info("server started", "address", cfg.RunAddr)
		err = http.ListenAndServe(cfg.RunAddr, r)
		if err != nil {
			log.Error("error running server", "error", err)
			os.Exit(1)
		}
	}()

	<-sigChan
	log.Info("received interruption signal, exitting")
	pool.Stop()
}
