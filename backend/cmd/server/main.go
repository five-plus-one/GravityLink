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

	"gravitylink/backend/internal/app"
	"gravitylink/backend/internal/config"
	"gravitylink/backend/internal/router"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	manager := app.NewManager(cfg, logger)
	if err := manager.Start(); err != nil {
		logger.Warn("normal runtime unavailable; admin setup mode enabled", "error", err)
	}
	defer manager.Shutdown()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           manager,
		ReadHeaderTimeout: 5 * time.Second,
	}
	adminServer := &http.Server{
		Addr:              cfg.AdminHTTPAddr,
		Handler:           router.NewAdminFrontendHandler(manager, manager.SetupHandler()),
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
