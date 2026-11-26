// Package user manages user accounts.
package user

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/VishRayk/brandshake/internal/store"
)

type Role string

const (
	RoleCreator Role = "creator"
	RoleBrand   Role = "brand"
)

func (r Role) Valid() bool { return r == RoleCreator || r == RoleBrand }

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	DisplayName  string    `json:"display_name"`
	CreatedAt    time.Time `json:"created_at"`
}

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already registered")
)

type Repo struct{ db *sql.DB }

func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// NormalizeEmail lowercases and trims an email for storage and lookup.
func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func (r *Repo) Create(ctx context.Context, email, passwordHash string, role Role, displayName string) (User, error) {
	u := User{
		ID:           uuid.NewString(),
		Email:        NormalizeEmail(email),
		PasswordHash: passwordHash,
		Role:         role,
		DisplayName:  strings.TrimSpace(displayName),
		CreatedAt:    time.Now().UTC().Truncate(time.Second),
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, role, display_name, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.PasswordHash, u.Role, u.DisplayName, u.CreatedAt.Format(time.RFC3339))
	if store.IsUniqueViolation(err) {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (r *Repo) ByEmail(ctx context.Context, email string) (User, error) {
	return r.scanOne(ctx, `WHERE email = ?`, NormalizeEmail(email))
}

func (r *Repo) ByID(ctx context.Context, id string) (User, error) {
	return r.scanOne(ctx, `WHERE id = ?`, id)
}

func (r *Repo) scanOne(ctx context.Context, where string, arg any) (User, error) {
	var (
		u       User
		created string
	)
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, role, display_name, created_at FROM users `+where, arg).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.DisplayName, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	u.CreatedAt, err = time.Parse(time.RFC3339, created)
	return u, err
}
