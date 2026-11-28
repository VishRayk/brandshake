// Command core-api runs the Brandshake REST API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/VishRayk/brandshake/internal/aiclient"
	"github.com/VishRayk/brandshake/internal/auth"
	"github.com/VishRayk/brandshake/internal/config"
	"github.com/VishRayk/brandshake/internal/deal"
	"github.com/VishRayk/brandshake/internal/email"
	"github.com/VishRayk/brandshake/internal/httpapi"
	"github.com/VishRayk/brandshake/internal/invoice"
	"github.com/VishRayk/brandshake/internal/jobs"
	"github.com/VishRayk/brandshake/internal/profile"
	"github.com/VishRayk/brandshake/internal/store"
	"github.com/VishRayk/brandshake/internal/user"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, nil)
	if cfg.IsDev() {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	slog.SetDefault(slog.New(handler))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := store.Migrate(ctx, db); err != nil {
		return err
	}

	// Background jobs run in this process until there is a reason to split them out.
	runner := jobs.NewRunner(db)
	runner.Workers = cfg.Workers
	runner.Handle(email.JobKind, email.Handler(db, email.LogSender{}))
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	var workersDone sync.WaitGroup
	if cfg.Workers > 0 {
		workersDone.Go(func() { runner.Run(workerCtx) })
		slog.Info("job workers started", "workers", cfg.Workers, "email", cfg.EmailProvider)
	}
	defer func() {
		stopWorkers()
		workersDone.Wait()
	}()

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: httpapi.NewHandler(httpapi.Deps{
			DB:             db,
			Users:          user.NewRepo(db),
			Profiles:       profile.NewRepo(db),
			Deals:          deal.NewRepo(db),
			Invoices:       invoice.NewRepo(db, cfg.PublicURL),
			AI:             aiclient.New(cfg.AIServiceURL, 45*time.Second),
			Tokens:         auth.NewTokens(cfg.JWTSecret, cfg.AccessTokenTTL),
			AllowedOrigins: cfg.AllowedOrigins,
			DevMode:        cfg.IsDev(),
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second, // above the AI client timeout (45s)
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("core-api listening", "addr", cfg.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
