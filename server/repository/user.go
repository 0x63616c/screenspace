package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
	Banned       bool
	CreatedAt    time.Time
}

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, email, passwordHash, role string) (*User, error) {
	u := &User{}
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3)
		 RETURNING id, email, password_hash, role, banned, created_at`,
		email, passwordHash, role,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Banned, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, role, banned, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Banned, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, role, banned, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Banned, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *UserRepo) SetBanned(ctx context.Context, id string, banned bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET banned = $1 WHERE id = $2`,
		banned, id,
	)
	if err != nil {
		return fmt.Errorf("set banned: %w", err)
	}
	return nil
}

func (r *UserRepo) SetRole(ctx context.Context, id, role string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET role = $1 WHERE id = $2`,
		role, id,
	)
	if err != nil {
		return fmt.Errorf("set role: %w", err)
	}
	return nil
}

func (r *UserRepo) List(ctx context.Context, limit, offset int) ([]*User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, password_hash, role, banned, created_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Banned, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepo) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}
