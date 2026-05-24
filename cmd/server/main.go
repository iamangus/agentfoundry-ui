package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/angoo/agentfoundry-ui/frontend"
	"github.com/angoo/agentfoundry-ui/internal/auth"
	"github.com/angoo/agentfoundry-ui/internal/config"
	"github.com/angoo/agentfoundry-ui/internal/cors"
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

	handler, err := web.NewHandler(cfg.BackendURL)
	if err != nil {
		slog.Error("failed to create web handler", "error", err)
		os.Exit(1)
	}

	handler.Client().SetTokenProvider(&contextTokenProvider{})

	distFS, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		slog.Error("failed to load embedded frontend", "error", err)
		os.Exit(1)
	}
	fileServer := http.FileServer(http.FS(distFS))

	mux := http.NewServeMux()

	authHandler := auth.NewHandler(authMgr, cfg)
	authHandler.RegisterRoutes(mux)

	handler.RegisterRoutes(mux)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/auth/") || strings.HasPrefix(r.URL.Path, "/api/") {
			return
		}

		assetPath := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/"), "./")
		if assetPath == "" {
			assetPath = "index.html"
		}

		info, err := fs.Stat(distFS, assetPath)
		if err != nil || info.IsDir() {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeFileFS(w, r, distFS, "index.html")
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	var rootHandler http.Handler = mux
	if authMgr.Enabled() {
		rootHandler = authMgr.Middleware(rootHandler)
	}
	rootHandler = cors.Middleware(cfg.CORSOrigin)(rootHandler)

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
