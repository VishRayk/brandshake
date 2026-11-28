// Package jobs is a small, durable background job queue on PostgreSQL.
//
// Producers call Enqueue (optionally inside their own transaction, so a job
// exists only if the business change committed). Workers claim one job at a
// time with FOR UPDATE SKIP LOCKED, run its handler, and either mark it done
// or schedule a retry with exponential backoff. Jobs whose worker died are
// reclaimed after a lock timeout.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Execer is satisfied by *pgxpool.Pool and pgx.Tx.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Job struct {
	ID       int64
	Kind     string
	Payload  json.RawMessage
	Attempts int
}

// Handler processes one job. Returning an error schedules a retry; wrap it
// with Permanent to fail the job immediately.
type Handler func(ctx context.Context, job Job) error

type permanentError struct{ err error }

func (p permanentError) Error() string { return p.err.Error() }
func (p permanentError) Unwrap() error { return p.err }

// Permanent marks an error as not worth retrying.
func Permanent(err error) error { return permanentError{err} }

type EnqueueOptions struct {
	RunAt       time.Time // zero means now
	MaxAttempts int       // zero means 5
	DedupeKey   string    // empty means no deduplication
}

// Enqueue adds a job. It returns false if a job with the same dedupe key
// already exists.
func Enqueue(ctx context.Context, db Execer, kind string, payload any, opt EnqueueOptions) (bool, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("marshal payload: %w", err)
	}
	if opt.MaxAttempts == 0 {
		opt.MaxAttempts = 5
	}
	var runAt any = nil
	if !opt.RunAt.IsZero() {
		runAt = opt.RunAt
	}
	var dedupe any = nil
	if opt.DedupeKey != "" {
		dedupe = opt.DedupeKey
	}
	tag, err := db.Exec(ctx, `
		INSERT INTO jobs (kind, payload, run_at, max_attempts, dedupe_key)
		VALUES ($1, $2, coalesce($3::timestamptz, now()), $4, $5)
		ON CONFLICT (dedupe_key) DO NOTHING`,
		kind, body, runAt, opt.MaxAttempts, dedupe)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

type Runner struct {
	db          *pgxpool.Pool
	handlers    map[string]Handler
	Workers     int
	PollEvery   time.Duration
	LockTimeout time.Duration
	BaseBackoff time.Duration
}

func NewRunner(db *pgxpool.Pool) *Runner {
	return &Runner{
		db:          db,
		handlers:    map[string]Handler{},
		Workers:     2,
		PollEvery:   time.Second,
		LockTimeout: 5 * time.Minute,
		BaseBackoff: 30 * time.Second,
	}
}

func (r *Runner) Handle(kind string, h Handler) { r.handlers[kind] = h }

// Run starts the workers and blocks until ctx is cancelled and in-flight
// jobs finish.
func (r *Runner) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for range r.Workers {
		wg.Go(func() { r.loop(ctx) })
	}
	wg.Wait()
}

func (r *Runner) loop(ctx context.Context) {
	for {
		worked, err := r.RunOnce(ctx)
		if err != nil && ctx.Err() == nil {
			slog.Error("jobs: claim failed", "err", err)
		}
		if worked {
			continue // drain the queue before sleeping
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(r.PollEvery + time.Duration(rand.Int64N(int64(r.PollEvery/2)+1))):
		}
	}
}

// RunOnce claims and processes at most one job. It reports whether a job ran.
func (r *Runner) RunOnce(ctx context.Context) (bool, error) {
	var job Job
	err := r.db.QueryRow(ctx, `
		UPDATE jobs SET status = 'running', locked_at = now(), attempts = attempts + 1, updated_at = now()
		WHERE id = (
			SELECT id FROM jobs
			WHERE (status = 'pending' AND run_at <= now())
			   OR (status = 'running' AND locked_at < now() - $1 * interval '1 second')
			ORDER BY run_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, kind, payload, attempts`, r.LockTimeout.Seconds()).
		Scan(&job.ID, &job.Kind, &job.Payload, &job.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Finish the job even if shutdown starts mid-handler.
	finishCtx := context.WithoutCancel(ctx)
	h, ok := r.handlers[job.Kind]
	var runErr error
	if !ok {
		runErr = Permanent(fmt.Errorf("no handler for job kind %q", job.Kind))
	} else {
		runErr = safeRun(ctx, h, job)
	}

	if runErr == nil {
		_, err = r.db.Exec(finishCtx, `UPDATE jobs SET status = 'done', locked_at = NULL, last_error = '', updated_at = now() WHERE id = $1`, job.ID)
		return true, err
	}

	var perm permanentError
	backoff := r.BaseBackoff * time.Duration(1<<min(job.Attempts-1, 10))
	_, err = r.db.Exec(finishCtx, `
		UPDATE jobs SET
			status = CASE WHEN $2 OR attempts >= max_attempts THEN 'failed' ELSE 'pending' END,
			run_at = now() + $3 * interval '1 second',
			locked_at = NULL, last_error = $4, updated_at = now()
		WHERE id = $1`, job.ID, errors.As(runErr, &perm), backoff.Seconds(), truncate(runErr.Error(), 1000))
	slog.Warn("jobs: job failed", "id", job.ID, "kind", job.Kind, "attempt", job.Attempts, "err", runErr)
	return true, err
}

func safeRun(ctx context.Context, h Handler, job Job) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = Permanent(fmt.Errorf("panic: %v", v))
		}
	}()
	return h(ctx, job)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
