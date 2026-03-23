package handler

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"github.com/0x63616c/screenspace/server/repository"
	"github.com/0x63616c/screenspace/server/service"
)

func newTestAuthHandler(t *testing.T) *AuthHandler {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://screenspace:devpassword@localhost:5432/screenspace?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("skipping, no database: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("skipping, database unreachable: %v", err)
	}
	// Clean test data
	db.Exec("DELETE FROM favorites")
	db.Exec("DELETE FROM reports")
	db.Exec("DELETE FROM wallpapers")
	db.Exec("DELETE FROM users WHERE email LIKE '%example.com'")

	users := repository.NewUserRepo(db)
	auth := service.NewAuthService("test-secret")
	return NewAuthHandler(users, auth, "admin@test.com")
}
