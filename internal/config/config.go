package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr    string         `yaml:"listen"`
	HTTPEndpoints []HTTPEndpoint `yaml:"http_endpoints"`
	WSEndpoints   []WSEndpoint   `yaml:"ws_endpoints"`
}

type HTTPEndpoint struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`

	Path      string   `yaml:"path"`
	ErrorRate float64  `yaml:"error_rate"`
	Slowness  Slowness `yaml:"slowness"`

	Hidden   bool `yaml:"hidden,omitempty"`
	DoNotLog bool `yaml:"do_not_log,omitempty"`
}

type Slowness struct {
	Min time.Duration `yaml:"min"`
	Max time.Duration `yaml:"max"`
	P95 time.Duration `yaml:"p95"`
}

type WSEndpoint struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`

	Path string `yaml:"path"`
	Type string `yaml:"type"`
}

const (
	defaultListenAddr = "0.0.0.0:8080"
)

func Get(path string) (Config, error) {
	var config Config
	file, err := os.Open(path)
	if err != nil {
		return config, fmt.Errorf("could not open config file: %w", err)
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		return config, fmt.Errorf("could not read config file: %w", err)
	}

	err = yaml.Unmarshal(fileData, &config)
	if err != nil {
		return config, fmt.Errorf("could not parse config file: %w", err)
	}

	if config.ListenAddr == "" {
		config.ListenAddr = defaultListenAddr
		slog.Info("using default listen address: " + config.ListenAddr)
	}

	if err = validateConfig(config); err != nil {
		return config, fmt.Errorf("can't validate config: %w", err)
	}

	slog.Debug("HTTPEndpoints:" + fmt.Sprintf("%+v", config.HTTPEndpoints))

	return config, nil
}

func validateConfig(cfg Config) error {
	if len(cfg.HTTPEndpoints) == 0 {
		return fmt.Errorf("no endpoints defined in the config")
	}

	paths := make(map[string]struct{}, len(cfg.HTTPEndpoints))
	for _, ep := range cfg.HTTPEndpoints {
		if ep.ErrorRate < 0 || ep.ErrorRate > 1 {
			return fmt.Errorf("endpoint error rate must be between 0.0 and 1.0 inclusive")
		}

		if _, ok := paths[ep.Path]; ok {
			return fmt.Errorf("duplicate endpoint path: %s", ep.Path)
		}
		paths[ep.Path] = struct{}{}

		if ep.Slowness.Min > ep.Slowness.Max || ep.Slowness.Min > ep.Slowness.P95 {
			return fmt.Errorf("slowness min cannot be greater than max or p95")
		}

		if ep.Slowness.P95 > ep.Slowness.Max {
			return fmt.Errorf("slowness p95 cannot be greater than max")
		}
	}

	if len(cfg.WSEndpoints) > 1 {
		return fmt.Errorf("only one websocket endpoint is supported now")
	}

	for _, ep := range cfg.WSEndpoints {
		if ep.Type != "echo" {
			return fmt.Errorf("only echo websocket endpoints are supported now")
		}
	}

	return nil
}
