package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	apphandler "github.com/0x63616c/screenspace/server/internal/handler"
	appmw "github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/video"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	// Structured logger.
	level := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Database pool.
	pool, err := openDB(ctx, cfg)
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := RunMigrations(pool); err != nil {
		slog.Error("migrations", "error", err)
		os.Exit(1)
	}

	// S3 storage.
	store, err := storage.NewS3Store(cfg.S3Endpoint, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey)
	if err != nil {
		slog.Error("storage", "error", err)
		os.Exit(1)
	}
	if err := store.EnsureBucket(ctx); err != nil {
		slog.Warn("could not ensure bucket", "error", err)
	}

	// sqlc querier.
	q := db.New(pool)

	// Services.
	authSvc := service.NewAuthService(cfg)
	videoProber := video.NewFFProber()
	wallpaperSvc := service.NewWallpaperService(q, store, videoProber, cfg)
	favoriteSvc := service.NewFavoriteService(q)
	reportSvc := service.NewReportService(q, cfg)

	// Middleware.
	bannedCache := appmw.NewBannedCache()
	authMw := appmw.Auth(authSvc)
	bannedMw := appmw.BannedCheck(q, bannedCache)

	publicLimiter := appmw.NewRateLimiter(cfg.PublicRateLimit, time.Minute)
	authLimiter := appmw.NewRateLimiter(cfg.AuthRateLimit, time.Minute)
	userLimiter := appmw.NewRateLimiter(cfg.UserRateLimit, time.Minute)
	uploadLimiter := appmw.NewRateLimiter(cfg.UploadRateLimit, 24*time.Hour)
	downloadLimiter := appmw.NewRateLimiter(cfg.DownloadRateLimit, time.Hour)

	// Handlers.
	wallpaperH := apphandler.NewWallpaperHandler(q, store, wallpaperSvc, authSvc, cfg)
	authH := apphandler.NewAuthHandler(q, authSvc, bannedCache, cfg)
	favoriteH := apphandler.NewFavoriteHandler(favoriteSvc)
	reportH := apphandler.NewReportHandler(reportSvc)
	adminH := apphandler.NewAdminHandler(q, store, wallpaperSvc, bannedCache, cfg)

	// Router.
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.CleanPath)
	r.Use(appmw.SecurityHeaders)
	r.Use(appmw.MaxBodySize)

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes.
		r.Group(func(r chi.Router) {
			r.Use(publicLimiter.PerIP())
			r.Get("/health", apphandler.Wrap(wallpaperH.Health))
			r.Get("/categories", apphandler.Wrap(apphandler.ListCategories))
			r.Get("/wallpapers", apphandler.Wrap(wallpaperH.List))
			r.Get("/wallpapers/popular", apphandler.Wrap(wallpaperH.Popular))
			r.Get("/wallpapers/recent", apphandler.Wrap(wallpaperH.Recent))
			r.Get("/wallpapers/{id}", apphandler.Wrap(wallpaperH.Get))
		})

		// Auth endpoints (no JWT required, IP-limited).
		r.Group(func(r chi.Router) {
			r.Use(authLimiter.PerIP())
			r.Post("/auth/register", apphandler.Wrap(authH.Register))
			r.Post("/auth/login", apphandler.Wrap(authH.Login))
		})

		// Authenticated routes.
		r.Group(func(r chi.Router) {
			r.Use(authMw)
			r.Use(bannedMw)
			r.Use(userLimiter.PerUser())
			r.Get("/auth/me", apphandler.Wrap(authH.Me))
			r.Post("/wallpapers", apphandler.Wrap(func(w http.ResponseWriter, req *http.Request) error {
				claims := appmw.ClaimsFromContext(req.Context())
				if claims != nil && !uploadLimiter.Allow(claims.UserID) {
					return &apphandler.AppError{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: "upload limit reached"}
				}
				return wallpaperH.Create(w, req)
			}))
			r.Post("/wallpapers/{id}/finalize", apphandler.Wrap(wallpaperH.Finalize))
			r.Post("/wallpapers/{id}/download", apphandler.Wrap(func(w http.ResponseWriter, req *http.Request) error {
				claims := appmw.ClaimsFromContext(req.Context())
				if claims != nil && !downloadLimiter.Allow(claims.UserID) {
					return &apphandler.AppError{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: "download limit reached"}
				}
				return wallpaperH.Download(w, req)
			}))
			r.Post("/wallpapers/{id}/favorite", apphandler.Wrap(favoriteH.Toggle))
			r.Post("/wallpapers/{id}/report", apphandler.Wrap(reportH.Create))
		})

		// Admin routes.
		r.Group(func(r chi.Router) {
			r.Use(authMw)
			r.Use(bannedMw)
			r.Use(appmw.Admin)
			r.Get("/admin/queue", apphandler.Wrap(adminH.Queue))
			r.Post("/admin/queue/{id}/approve", apphandler.Wrap(adminH.Approve))
			r.Post("/admin/queue/{id}/reject", apphandler.Wrap(adminH.Reject))
			r.Get("/admin/wallpapers", apphandler.Wrap(adminH.ListWallpapers))
			r.Patch("/admin/wallpapers/{id}", apphandler.Wrap(adminH.EditWallpaper))
			r.Get("/admin/users", apphandler.Wrap(adminH.ListUsers))
			r.Post("/admin/users/{id}/ban", apphandler.Wrap(adminH.BanUser))
			r.Post("/admin/users/{id}/unban", apphandler.Wrap(adminH.UnbanUser))
			r.Post("/admin/users/{id}/promote", apphandler.Wrap(adminH.PromoteUser))
			r.Get("/admin/reports", apphandler.Wrap(adminH.ListReports))
			r.Post("/admin/reports/{id}/dismiss", apphandler.Wrap(adminH.DismissReport))
		})
	})

	// HTTP server with hardened timeouts.
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", srv.Addr)
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
	slog.Info("shutdown complete")
}

func openDB(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = int32(cfg.DBMaxConns)
	poolCfg.MinConns = int32(cfg.DBMinConns)
	poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolCfg.HealthCheckPeriod = cfg.DBHealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
