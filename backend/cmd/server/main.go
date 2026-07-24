package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/database"
	"gravitylink/backend/internal/router"
	"gravitylink/backend/internal/worker"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	db, err := database.ConnectMySQL(cfg.MySQLDSN)
	if err != nil {
		logger.Error("connect mysql failed", "error", err)
		os.Exit(1)
	}

	redisClient, err := database.ConnectRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		logger.Error("connect redis failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Warn("close redis failed", "error", err)
		}
	}()

	engine := router.New(router.Dependencies{
		Config: cfg,
		DB:     db,
		Redis:  redisClient,
		Logger: logger,
	})
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()
	go worker.NewAccessLogConsumer(db, redisClient, logger).Start(workerCtx)
	go worker.NewLogArchiver(db, logger).Start(workerCtx)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}
	adminServer := &http.Server{
		Addr:              cfg.AdminHTTPAddr,
		Handler:           router.NewAdminFrontendHandler(engine),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("gravitylink server starting", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()
	if cfg.AdminHTTPAddr != "" {
		go func() {
			logger.Info("gravitylink admin frontend starting", "addr", cfg.AdminHTTPAddr)
			if err := adminServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("admin frontend server failed", "error", err)
				os.Exit(1)
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("http server shutdown failed", "error", err)
		os.Exit(1)
	}
	if cfg.AdminHTTPAddr != "" {
		if err := adminServer.Shutdown(ctx); err != nil {
			logger.Error("admin frontend server shutdown failed", "error", err)
			os.Exit(1)
		}
	}
	logger.Info("gravitylink server stopped")
}
