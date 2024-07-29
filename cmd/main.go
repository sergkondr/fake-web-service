package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/sergkondr/fake-web-service/internal/config"
	"github.com/sergkondr/fake-web-service/internal/web"
)

var (
	version               = "dev"
	defaultConfigFilename = "config.yaml"
)

func main() {
	debugMode := flag.Bool("debug", false, "debug mode")
	configPath := flag.String("config", defaultConfigFilename, "path to config file")
	flag.Parse()

	if *debugMode {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	slog.Debug("debug mode is on")
	slog.Debug("version: " + version)

	cfg, err := config.Get(*configPath)
	if err != nil {
		slog.Error("error loading config: " + err.Error())
		os.Exit(1)
	}
	slog.Debug("config is loaded")

	srv := web.New(cfg)
	slog.Info("starting server")
	if err = http.ListenAndServe(cfg.ListenAddr, srv); err != nil {
		slog.Error("error starting server: " + err.Error())
	}
}
