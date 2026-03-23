package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/0x63616c/screenspace/server/handler"
	"github.com/0x63616c/screenspace/server/middleware"
	"github.com/0x63616c/screenspace/server/repository"
	"github.com/0x63616c/screenspace/server/service"
	"github.com/0x63616c/screenspace/server/storage"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(db); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	// S3 Storage
	store, err := storage.NewS3Store(cfg.S3Endpoint, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	if err := store.EnsureBucket(context.Background()); err != nil {
		log.Printf("warning: could not ensure bucket: %v", err)
	}

	// Repositories
	userRepo := repository.NewUserRepo(db)
	wallpaperRepo := repository.NewWallpaperRepo(db)
	favoriteRepo := repository.NewFavoriteRepo(db)
	reportRepo := repository.NewReportRepo(db)

	// Services
	authService := service.NewAuthService(cfg.JWTSecret)
	videoService := service.NewVideoService()

	// Handlers
	authHandler := handler.NewAuthHandler(userRepo, authService, cfg.AdminEmail)
	wallpaperHandler := handler.NewWallpaperHandler(wallpaperRepo, store, videoService, authService)
	favoriteHandler := handler.NewFavoriteHandler(favoriteRepo)
	reportHandler := handler.NewReportHandler(reportRepo)
	adminHandler := handler.NewAdminHandler(wallpaperRepo, userRepo, reportRepo)

	// Middleware
	authMw := middleware.Auth(authService)
	uploadLimiter := middleware.NewRateLimiter(5) // 5 uploads per day

	// Router
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("GET /api/v1/wallpapers", wallpaperHandler.List)
	mux.HandleFunc("GET /api/v1/wallpapers/popular", wallpaperHandler.Popular)
	mux.HandleFunc("GET /api/v1/wallpapers/recent", wallpaperHandler.Recent)
	mux.HandleFunc("GET /api/v1/wallpapers/{id}", wallpaperHandler.Get)

	// Authenticated routes
	mux.Handle("GET /api/v1/auth/me", authMw(http.HandlerFunc(authHandler.Me)))
	mux.Handle("POST /api/v1/wallpapers", authMw(uploadLimiter.Middleware(http.HandlerFunc(wallpaperHandler.Create))))
	mux.Handle("POST /api/v1/wallpapers/{id}/download", authMw(http.HandlerFunc(wallpaperHandler.Download)))
	mux.Handle("POST /api/v1/wallpapers/{id}/finalize", authMw(http.HandlerFunc(wallpaperHandler.Finalize)))
	mux.Handle("DELETE /api/v1/wallpapers/{id}", authMw(http.HandlerFunc(wallpaperHandler.Delete)))
	mux.Handle("POST /api/v1/wallpapers/{id}/favorite", authMw(http.HandlerFunc(favoriteHandler.Toggle)))
	mux.Handle("GET /api/v1/me/favorites", authMw(http.HandlerFunc(favoriteHandler.List)))
	mux.Handle("POST /api/v1/wallpapers/{id}/report", authMw(http.HandlerFunc(reportHandler.Create)))

	// Admin routes (auth + admin middleware)
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

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
