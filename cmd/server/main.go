package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/h0ugetsu/realworld-api/internal/config"
	"github.com/h0ugetsu/realworld-api/internal/handler"
	"github.com/h0ugetsu/realworld-api/internal/repository"
	"github.com/h0ugetsu/realworld-api/internal/server"
	"github.com/h0ugetsu/realworld-api/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return err
	}

	dbpool, err := pgxpool.New(context.Background(), cfg.DB.URL)
	if err != nil {
		slog.Error("failed to connect database", "error", err)
		return err
	}
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := dbpool.Ping(pingCtx); err != nil {
		slog.Error("failed to ping database", "error", err)
		return err
	}
	defer dbpool.Close()

	queries := repository.New(dbpool)

	userService := service.NewUserService(queries)
	authService := service.NewAuthService(cfg.JWT.Secret)
	userHandler := handler.NewUserHandler(userService, authService)

	articleService := service.NewArticleService(queries, dbpool)
	articleHandler := handler.NewArticleHandler(articleService)

	router := server.NewRouter(userHandler, articleHandler, authService, cfg)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("🚀 Starting server...", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("😭 Failed to start server", "error", err)
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		logger.Info("🌇 Shutting down server...")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("😭 Failed to shutdown server", "error", err)
			return err
		}
		logger.Info("🚪 Server shutdown complete")
	case err := <-errCh:
		return err
	}

	return nil
}
