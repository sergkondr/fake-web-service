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
	ListenAddr string     `yaml:"listen"`
	Endpoints  []Endpoint `yaml:"endpoints"`
}

type Endpoint struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`

	Path      string  `yaml:"path"`
	ErrorRate float64 `yaml:"error_rate"`
	Slowness  struct {
		Min time.Duration `yaml:"min"`
		Max time.Duration `yaml:"max"`
		P95 time.Duration `yaml:"p95"`
	} `yaml:"slowness"`
}

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

	slog.Debug("Endpoints:" + fmt.Sprintf("%+v", config.Endpoints))

	return config, nil
}
