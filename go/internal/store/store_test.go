package store_test

import (
	"context"
	"sync"
	"testing"

	"github.com/VishRayk/brandshake/internal/store"
	"github.com/VishRayk/brandshake/internal/testdb"
)

func TestMigrateIsIdempotent(t *testing.T) {
	pool := testdb.New(t) // already migrated once
	ctx := context.Background()

	var before int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	var after int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before == 0 || before != after {
		t.Fatalf("migrations before=%d after=%d", before, after)
	}
}

func TestConcurrentMigrateIsSafe(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make(chan error, 5)
	for range 5 {
		wg.Go(func() { errs <- store.Migrate(ctx, pool) })
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent migrate: %v", err)
		}
	}
}

func TestIsUniqueViolation(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()

	insert := `INSERT INTO users (email, password_hash, role, display_name) VALUES ('a@x.test', 'h', 'brand', 'A')`
	if _, err := pool.Exec(ctx, insert); err != nil {
		t.Fatal(err)
	}
	_, err := pool.Exec(ctx, insert)
	if !store.IsUniqueViolation(err) || !store.IsUniqueViolation(err, "users_email_key") {
		t.Fatalf("expected unique violation, got %v", err)
	}
	if store.IsUniqueViolation(err, "some_other_key") {
		t.Fatal("matched the wrong constraint")
	}
}
