package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func newTestUserRepo(t *testing.T) *UserRepo {
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

	return NewUserRepo(db)
}

func TestUserRepo_CreateAndGetByEmail(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	u, err := repo.Create(ctx, "create-test@example.com", "hashedpw", "user")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.Email != "create-test@example.com" {
		t.Fatalf("expected email create-test@example.com, got %s", u.Email)
	}
	if u.Role != "user" {
		t.Fatalf("expected role user, got %s", u.Role)
	}

	got, err := repo.GetByEmail(ctx, "create-test@example.com")
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if got.ID != u.ID {
		t.Fatalf("expected id %s, got %s", u.ID, got.ID)
	}
}

func TestUserRepo_GetByID(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	u, err := repo.Create(ctx, "getbyid-test@example.com", "hashedpw", "user")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Email != u.Email {
		t.Fatalf("expected email %s, got %s", u.Email, got.Email)
	}
}

func TestUserRepo_DuplicateEmail(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, "dup-test@example.com", "hashedpw", "user")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = repo.Create(ctx, "dup-test@example.com", "hashedpw2", "user")
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestUserRepo_SetBanned(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	u, err := repo.Create(ctx, "ban-test@example.com", "hashedpw", "user")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.SetBanned(ctx, u.ID, true); err != nil {
		t.Fatalf("set banned: %v", err)
	}

	got, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.Banned {
		t.Fatal("expected user to be banned")
	}
}

func TestUserRepo_SetRole(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	u, err := repo.Create(ctx, "role-test@example.com", "hashedpw", "user")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.SetRole(ctx, u.ID, "admin"); err != nil {
		t.Fatalf("set role: %v", err)
	}

	got, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Role != "admin" {
		t.Fatalf("expected role admin, got %s", got.Role)
	}
}

func TestUserRepo_List(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	for i := range 3 {
		email := "list-test-" + string(rune('a'+i)) + "@example.com"
		if _, err := repo.Create(ctx, email, "hashedpw", "user"); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	users, err := repo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(users) < 3 {
		t.Fatalf("expected at least 3 users, got %d", len(users))
	}

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count < 3 {
		t.Fatalf("expected count >= 3, got %d", count)
	}
}
