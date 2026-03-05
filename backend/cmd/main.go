package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gabrielrnascimento/nightfall/backend/internal/lobby"
	"github.com/gabrielrnascimento/nightfall/backend/internal/telemetry"
)

func main() {
	log.SetFlags(0)

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	shutdown, err := telemetry.Setup(ctx, "nightfall-backend", "0.1.0")
	if err != nil {
		return err
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Printf("failed to shutdown telemetry: %v", err)
		}
	}()

	l, err := net.Listen("tcp", "127.0.0.1:3001")
	if err != nil {
		return err
	}
	log.Printf("listening on ws://%v", l.Addr())

	s := &http.Server{
		Handler: lobby.Server{
			Logf: log.Printf,
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
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}
