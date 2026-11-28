// Package user manages user accounts.
package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

// NormalizeEmail lowercases and trims an email for storage and lookup.
func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

const columns = `id::text, email, password_hash, role, display_name, created_at`

func (r *Repo) Create(ctx context.Context, email, passwordHash string, role Role, displayName string) (User, error) {
	u, err := scan(r.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role, display_name)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+columns,
		NormalizeEmail(email), passwordHash, role, strings.TrimSpace(displayName)))
	if store.IsUniqueViolation(err, "users_email_key") {
		return User{}, ErrEmailTaken
	}
	return u, err
}

func (r *Repo) ByEmail(ctx context.Context, email string) (User, error) {
	return scan(r.db.QueryRow(ctx, `SELECT `+columns+` FROM users WHERE email = $1`, NormalizeEmail(email)))
}

func (r *Repo) ByID(ctx context.Context, id string) (User, error) {
	// Compare as text so a malformed id is simply "not found" rather than a cast error.
	return scan(r.db.QueryRow(ctx, `SELECT `+columns+` FROM users WHERE id::text = $1`, id))
}

// Delete removes the user and everything tied to them: profiles, deals and
// invoices cascade; emails sent to their address and unsent queued emails
// are removed explicitly because they are keyed by address, not user.
func (r *Repo) Delete(ctx context.Context, id string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var email string
		err := tx.QueryRow(ctx, `DELETE FROM users WHERE id::text = $1 RETURNING email`, id).Scan(&email)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM emails WHERE to_address = $1`, email); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `DELETE FROM jobs WHERE status IN ('pending', 'failed') AND payload->>'to' = $1`, email)
		return err
	})
}

func scan(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.DisplayName, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}
