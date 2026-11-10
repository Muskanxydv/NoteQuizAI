// Command api starts the NoteQuizAI HTTP server.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notequizai/internal/config"
	"notequizai/internal/generator"
	"notequizai/internal/handler"
	"notequizai/internal/llm"
	"notequizai/internal/router"
	"notequizai/internal/service"
	"notequizai/internal/storage"
	"notequizai/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New()

	pool, err := llm.NewPool(llm.PoolConfig{
		APIKey:         cfg.GeminiAPIKey,
		Models:         cfg.GeminiModels,
		RPM:            cfg.RPMPerModel,
		RPD:            cfg.RPDPerModel,
		RequestTimeout: cfg.RequestTimeout,
		MaxRetries:     cfg.MaxRetries,
	})
	if err != nil {
		return err
	}

	gen := generator.New(pool, cfg.MaxWorkers, 2)
	svc := service.New(gen, storage.NewMemory(200))
	h := handler.NewAssessmentHandler(svc, log, cfg.GenerateTimeout)

	srv := &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:           router.New(h, log),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      cfg.GenerateTimeout + 15*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("server listening", "addr", srv.Addr, "models", cfg.GeminiModels, "workers", cfg.MaxWorkers)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.GenerateTimeout+5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
