package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sergkondr/fake-web-service/internal/config"
	"github.com/sergkondr/fake-web-service/internal/web"
)

var (
	version               = "dev"
	defaultConfigFilename = "config.yaml"
)

const (
	readHeaderTimeout       = 5 * time.Second
	readTimeout             = 30 * time.Second
	writeTimeout            = 30 * time.Second
	idleTimeout             = 2 * time.Minute
	gracefulShutdownTimeout = 10 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:]); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("fakesvc", flag.ContinueOnError)
	debugMode := flags.Bool("debug", false, "debug mode")
	configPath := flags.String("config", defaultConfigFilename, "path to config file")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *debugMode {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	slog.Debug("debug mode is on")
	slog.Debug("version", "version", version)

	cfg, err := config.Get(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	slog.Debug("config is loaded")

	handler, err := web.New(cfg)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	return serve(ctx, newServer(cfg.ListenAddr, handler))
}

func newServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func serve(ctx context.Context, server *http.Server) error {
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("starting server", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen on %s: %w", server.Addr, err)
	case <-ctx.Done():
		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen on %s: %w", server.Addr, err)
		}
		return nil
	}
}
