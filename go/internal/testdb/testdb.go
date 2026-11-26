// Package testdb gives each test its own freshly migrated PostgreSQL schema,
// so tests are isolated and can run in parallel against one database.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/VishRayk/brandshake/internal/store"
)

const defaultURL = "postgres://brandshake:brandshake@localhost:5432/brandshake_test?sslmode=disable"

// New returns a pool whose search_path points at a new, migrated schema.
// The schema is dropped when the test finishes.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	url := os.Getenv("BRANDSHAKE_TEST_DATABASE_URL")
	if url == "" {
		url = defaultURL
	}

	admin, err := store.Open(ctx, url)
	if err != nil {
		t.Fatalf("test database unavailable (is PostgreSQL running? try `make db-start`): %v", err)
	}
	t.Cleanup(admin.Close)

	b := make([]byte, 6)
	_, _ = rand.Read(b)
	schema := "test_" + hex.EncodeToString(b)
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 4

	pool, err := store.OpenConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}
