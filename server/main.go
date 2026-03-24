package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/handler"
	"github.com/0x63616c/screenspace/server/middleware"
	"github.com/0x63616c/screenspace/server/service"
	"github.com/0x63616c/screenspace/server/storage"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Database pool
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database config", "error", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = int32(cfg.DBMaxConns)
	poolCfg.MinConns = int32(cfg.DBMinConns)
	poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolCfg.HealthCheckPeriod = cfg.DBHealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		slog.Error("database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping", "error", err)
		os.Exit(1)
	}

	if err := RunMigrations(pool); err != nil {
		slog.Error("migrations", "error", err)
		os.Exit(1)
	}

	// Querier
	q := db.New(pool)

	// S3 Storage
	store, err := storage.NewS3Store(cfg.S3Endpoint, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey)
	if err != nil {
		slog.Error("storage", "error", err)
		os.Exit(1)
	}
	if err := store.EnsureBucket(ctx); err != nil {
		slog.Warn("could not ensure bucket", "error", err)
	}

	// Services
	authService := service.NewAuthService(cfg.JWTSecret)
	videoService := service.NewVideoService()

	// Handlers
	authHandler := handler.NewAuthHandler(q, authService, cfg.AdminEmail)
	wallpaperHandler := handler.NewWallpaperHandler(q, store, videoService, authService)
	favoriteHandler := handler.NewFavoriteHandler(q, pool)
	reportHandler := handler.NewReportHandler(q)
	adminHandler := handler.NewAdminHandler(q)

	// Middleware
	authMw := middleware.Auth(authService)
	uploadLimiter := middleware.NewRateLimiter(cfg.UploadRateLimit)

	// Router (still net/http.ServeMux - chi migration is a later plan)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("GET /api/v1/categories", handler.ListCategories)
	mux.HandleFunc("GET /api/v1/wallpapers", wallpaperHandler.List)
	mux.HandleFunc("GET /api/v1/wallpapers/popular", wallpaperHandler.Popular)
	mux.HandleFunc("GET /api/v1/wallpapers/recent", wallpaperHandler.Recent)
	mux.HandleFunc("GET /api/v1/wallpapers/{id}", wallpaperHandler.Get)

	mux.Handle("GET /api/v1/auth/me", authMw(http.HandlerFunc(authHandler.Me)))
	mux.Handle("POST /api/v1/wallpapers", authMw(uploadLimiter.Middleware(http.HandlerFunc(wallpaperHandler.Create))))
	mux.Handle("POST /api/v1/wallpapers/{id}/download", authMw(http.HandlerFunc(wallpaperHandler.Download)))
	mux.Handle("POST /api/v1/wallpapers/{id}/finalize", authMw(http.HandlerFunc(wallpaperHandler.Finalize)))
	mux.Handle("DELETE /api/v1/wallpapers/{id}", authMw(http.HandlerFunc(wallpaperHandler.Delete)))
	mux.Handle("POST /api/v1/wallpapers/{id}/favorite", authMw(http.HandlerFunc(favoriteHandler.Toggle)))
	mux.Handle("GET /api/v1/me/favorites", authMw(http.HandlerFunc(favoriteHandler.List)))
	mux.Handle("POST /api/v1/wallpapers/{id}/report", authMw(http.HandlerFunc(reportHandler.Create)))

	mux.Handle("GET /api/v1/admin/queue", authMw(middleware.Admin(http.HandlerFunc(adminHandler.Queue))))
	mux.Handle("POST /api/v1/admin/queue/{id}/approve", authMw(middleware.Admin(http.HandlerFunc(adminHandler.Approve))))
	mux.Handle("POST /api/v1/admin/queue/{id}/reject", authMw(middleware.Admin(http.HandlerFunc(adminHandler.Reject))))
	mux.Handle("GET /api/v1/admin/wallpapers", authMw(middleware.Admin(http.HandlerFunc(adminHandler.ListWallpapers))))
	mux.Handle("PATCH /api/v1/admin/wallpapers/{id}", authMw(middleware.Admin(http.HandlerFunc(adminHandler.EditWallpaper))))
	mux.Handle("GET /api/v1/admin/users", authMw(middleware.Admin(http.HandlerFunc(adminHandler.ListUsers))))
	mux.Handle("POST /api/v1/admin/users/{id}/ban", authMw(middleware.Admin(http.HandlerFunc(adminHandler.BanUser))))
	mux.Handle("POST /api/v1/admin/users/{id}/unban", authMw(middleware.Admin(http.HandlerFunc(adminHandler.UnbanUser))))
	mux.Handle("POST /api/v1/admin/users/{id}/promote", authMw(middleware.Admin(http.HandlerFunc(adminHandler.PromoteUser))))
	mux.Handle("GET /api/v1/admin/reports", authMw(middleware.Admin(http.HandlerFunc(adminHandler.ListReports))))
	mux.Handle("POST /api/v1/admin/reports/{id}/dismiss", authMw(middleware.Admin(http.HandlerFunc(adminHandler.DismissReport))))

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down", "signal", ctx.Err())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "error", err)
	}
}
