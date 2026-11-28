package jobs_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/VishRayk/brandshake/internal/jobs"
	"github.com/VishRayk/brandshake/internal/testdb"
)

func TestEnqueueRunAndDedupe(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := context.Background()

	ok, err := jobs.Enqueue(ctx, db, "greet", map[string]string{"name": "maya"}, jobs.EnqueueOptions{DedupeKey: "greet:maya"})
	if err != nil || !ok {
		t.Fatalf("enqueue: %v %v", ok, err)
	}
	if ok, _ := jobs.Enqueue(ctx, db, "greet", nil, jobs.EnqueueOptions{DedupeKey: "greet:maya"}); ok {
		t.Fatal("duplicate dedupe key enqueued twice")
	}

	r := jobs.NewRunner(db)
	var got string
	r.Handle("greet", func(_ context.Context, j jobs.Job) error {
		var p map[string]string
		_ = json.Unmarshal(j.Payload, &p)
		got = p["name"]
		return nil
	})
	if worked, err := r.RunOnce(ctx); !worked || err != nil {
		t.Fatalf("RunOnce: %v %v", worked, err)
	}
	if got != "maya" {
		t.Errorf("payload = %q", got)
	}
	if worked, _ := r.RunOnce(ctx); worked {
		t.Error("a done job ran again")
	}
	var st string
	_ = db.QueryRow(ctx, `SELECT status FROM jobs`).Scan(&st)
	if st != "done" {
		t.Errorf("status = %s", st)
	}
}

func TestRetryWithBackoffThenFail(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := context.Background()
	if _, err := jobs.Enqueue(ctx, db, "flaky", nil, jobs.EnqueueOptions{MaxAttempts: 2}); err != nil {
		t.Fatal(err)
	}
	r := jobs.NewRunner(db)
	r.BaseBackoff = time.Hour
	r.Handle("flaky", func(context.Context, jobs.Job) error { return errors.New("upstream down") })

	r.RunOnce(ctx)
	var st, lastErr string
	var attempts int
	var wait float64
	_ = db.QueryRow(ctx, `SELECT status, attempts, last_error, extract(epoch FROM run_at - now()) FROM jobs`).Scan(&st, &attempts, &lastErr, &wait)
	if st != "pending" || attempts != 1 || lastErr != "upstream down" || wait < 3500 {
		t.Fatalf("after 1st failure: %s attempts=%d err=%q wait=%.0fs", st, attempts, lastErr, wait)
	}
	if worked, _ := r.RunOnce(ctx); worked {
		t.Fatal("job retried before its backoff elapsed")
	}

	_, _ = db.Exec(ctx, `UPDATE jobs SET run_at = now()`)
	r.RunOnce(ctx)
	_ = db.QueryRow(ctx, `SELECT status, attempts FROM jobs`).Scan(&st, &attempts)
	if st != "failed" || attempts != 2 {
		t.Fatalf("after max attempts: %s attempts=%d", st, attempts)
	}
}

func TestPermanentErrorsAndPanicsFailImmediately(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := context.Background()
	for _, k := range []string{"bad", "boom", "unknown-kind"} {
		if _, err := jobs.Enqueue(ctx, db, k, nil, jobs.EnqueueOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	r := jobs.NewRunner(db)
	r.Handle("bad", func(context.Context, jobs.Job) error { return jobs.Permanent(errors.New("invalid")) })
	r.Handle("boom", func(context.Context, jobs.Job) error { panic("nil map") })
	for range 3 {
		r.RunOnce(ctx)
	}
	var failed int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE status = 'failed' AND attempts = 1`).Scan(&failed)
	if failed != 3 {
		t.Fatalf("failed = %d, want 3", failed)
	}
}

func TestStaleRunningJobIsReclaimed(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := context.Background()
	_, _ = jobs.Enqueue(ctx, db, "work", nil, jobs.EnqueueOptions{})
	// Simulate a worker that crashed mid-job ten minutes ago.
	_, _ = db.Exec(ctx, `UPDATE jobs SET status = 'running', attempts = 1, locked_at = now() - interval '10 minutes'`)

	r := jobs.NewRunner(db)
	ran := false
	r.Handle("work", func(context.Context, jobs.Job) error { ran = true; return nil })
	if worked, err := r.RunOnce(ctx); !worked || err != nil || !ran {
		t.Fatalf("stale job not reclaimed: worked=%v ran=%v err=%v", worked, ran, err)
	}
}

func TestConcurrentWorkersProcessEachJobOnce(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := context.Background()
	const n = 60
	for i := range n {
		if _, err := jobs.Enqueue(ctx, db, "count", map[string]int{"i": i}, jobs.EnqueueOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	var mu sync.Mutex
	seen := map[int64]int{}
	var total atomic.Int64
	r := jobs.NewRunner(db)
	r.Handle("count", func(_ context.Context, j jobs.Job) error {
		mu.Lock()
		seen[j.ID]++
		mu.Unlock()
		total.Add(1)
		return nil
	})

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for {
				worked, err := r.RunOnce(ctx)
				if err != nil {
					t.Error(err)
					return
				}
				if !worked {
					return
				}
			}
		})
	}
	wg.Wait()
	if total.Load() != n || len(seen) != n {
		t.Fatalf("processed %d jobs (%d distinct), want %d", total.Load(), len(seen), n)
	}
	for id, c := range seen {
		if c != 1 {
			t.Errorf("job %d ran %d times", id, c)
		}
	}
}
