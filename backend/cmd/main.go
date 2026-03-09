package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gabrielrnascimento/nightfall/backend/internal/lobby"
	"github.com/gabrielrnascimento/nightfall/backend/internal/telemetry"
)

func main() {
	err := run()
	if err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	if os.Getenv("ENABLE_OTEL") == "true" {
		shutdown, err := telemetry.Setup(ctx, "nightfall-backend", "0.1.0")
		if err != nil {
			return err
		}
		defer func() {
			if err := shutdown(ctx); err != nil {
				slog.Error("failed to shutdown telemetry", "error", err)
			}
		}()
	}

	l, err := net.Listen("tcp", "127.0.0.1:3001")
	if err != nil {
		return err
	}
	slog.Info("listening", "addr", "ws://"+l.Addr().String())

	s := &http.Server{
		Handler: lobby.Server{
			Logger: slog.Default(),
		},
	}
	errC := make(chan error, 1)
	go func() {
		errC <- s.Serve(l)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errC:
		slog.Error("failed to serve", "error", err)
	case sig := <-sigs:
		slog.Info("terminating", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}
