package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/angoo/agentfoundry-ui/internal/auth"
	"github.com/angoo/agentfoundry-ui/internal/config"
	"github.com/angoo/agentfoundry-ui/internal/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	slog.Info("loaded config", "listen", cfg.Listen, "backend_url", cfg.BackendURL, "auth_enabled", cfg.AuthEnabled())

	ctx := context.Background()

	authMgr, err := auth.NewManager(ctx, cfg)
	if err != nil {
		slog.Error("failed to create auth manager", "error", err)
		os.Exit(1)
	}

	handler, err := web.NewHandler(cfg.BackendURL, authMgr)
	if err != nil {
		slog.Error("failed to create web handler", "error", err)
		os.Exit(1)
	}

	if authMgr.Enabled() {
		handler.Client().SetTokenProvider(&contextTokenProvider{})
	}

	mux := http.NewServeMux()

	authHandler := auth.NewHandler(authMgr, cfg)
	authHandler.RegisterRoutes(mux)

	handler.RegisterRoutes(mux)

	var rootHandler http.Handler = mux
	if authMgr.Enabled() {
		rootHandler = authMgr.Middleware(mux)
	}

	server := &http.Server{
		Addr:    cfg.Listen,
		Handler: rootHandler,
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("agentfoundry-ui starting", "addr", cfg.Listen)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-sigCtx.Done()
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	fmt.Println("agentfoundry-ui stopped")
}

type contextTokenProvider struct{}

func (p *contextTokenProvider) GetAccessToken(ctx context.Context) string {
	return auth.AccessTokenFromContext(ctx)
}
