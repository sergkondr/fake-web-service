package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/sergkondr/fake-web-service/internal/config"
	"github.com/sergkondr/fake-web-service/internal/web"
)

func main() {
	debugMode := flag.Bool("debug", false, "debug mode")
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	if *debugMode {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("debug mode is on")
	}

	cfg, err := config.Get(*configPath)
	if err != nil {
		slog.Error("error loading config: " + err.Error())
		os.Exit(1)
	}
	slog.Debug("loaded config")

	srv := web.New(cfg)

	slog.Info("starting server")
	err = http.ListenAndServe(cfg.ListenAddr, srv)
	if err != nil {
		slog.Error("error starting server", err)
	}
}
