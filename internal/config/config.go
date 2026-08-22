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
	Hostname string `yaml:"-"`

	ListenAddr string     `yaml:"listen"`
	Endpoints  []Endpoint `yaml:"endpoints"`
	Metrics    Metrics    `yaml:"metrics,omitempty"`
}

type EndpointType string

const (
	EndpointTypeHTTP     EndpointType = "http"
	EndpointTypeProxy    EndpointType = "proxy"
	EndpointTypeWSEcho   EndpointType = "ws/echo"
	EndpointTypeWSStream EndpointType = "ws/stream"
)

type Endpoint struct {
	Name        string       `yaml:"name,omitempty"`
	Description string       `yaml:"description,omitempty"`
	Type        EndpointType `yaml:"type"`
	Path        string       `yaml:"path"`
	Hidden      bool         `yaml:"hidden,omitempty"`
	DoNotLog    bool         `yaml:"do_not_log,omitempty"`

	Chaos    Chaos           `yaml:"chaos,omitempty"`
	Response *ResponseConfig `yaml:"response,omitempty"`
	Backend  *BackendConfig  `yaml:"backend,omitempty"`
	Stream   *StreamConfig   `yaml:"stream,omitempty"`
}

type Chaos struct {
	ErrorRate   float64  `yaml:"error_rate,omitempty"`
	ErrorStatus int      `yaml:"error_status,omitempty"`
	Latency     *Latency `yaml:"latency,omitempty"`
}

type Latency struct {
	Min time.Duration `yaml:"min"`
	Max time.Duration `yaml:"max"`
	P95 time.Duration `yaml:"p95"`
}

type ResponseConfig struct {
	Status  int               `yaml:"status,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    string            `yaml:"body,omitempty"`
}

type BackendConfig struct {
	URL          string        `yaml:"url"`
	Timeout      time.Duration `yaml:"timeout,omitempty"`
	PreserveHost bool          `yaml:"preserve_host,omitempty"`
}

type StreamConfig struct {
	Interval time.Duration `yaml:"interval"`
}

type Metrics struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	Path    string `yaml:"path,omitempty"`
}

const (
	defaultListenAddr = "0.0.0.0:8080"

	defaultMetricsPath     = "/metrics"
	defaultHealthcheckPath = "/healthz"
	defaultErrorStatus     = 500
	defaultBackendTimeout  = 10 * time.Second
	defaultResponseStatus  = 200
)

func Get(path string) (Config, error) {
	config, err := parseFileConfig(path)
	if err != nil {
		return config, fmt.Errorf("could not parse config: %w", err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		slog.Error("could not get hostname:" + err.Error())
		hostname = "unknown"
	}
	config.Hostname = hostname

	return config, nil
}

func parseFileConfig(path string) (Config, error) {
	var config Config

	file, err := os.Open(path)
	if err != nil {
		return config, fmt.Errorf("could not open config file: %w", err)
	}
	defer file.Close()

	return parseConfig(file)
}

func parseConfig(reader io.Reader) (Config, error) {
	var config Config
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)

	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("could not parse config file: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return config, fmt.Errorf("multiple YAML documents are not supported")
		}
		return config, fmt.Errorf("could not parse config file: %w", err)
	}

	usedDefaultListenAddress := config.ListenAddr == ""
	applyDefaults(&config)

	if usedDefaultListenAddress {
		slog.Info("using default listen address: " + config.ListenAddr)
	}

	if config.Metrics.Enabled {
		slog.Info("prometheus metrics enabled on " + config.Metrics.Path)
	}

	if err := validateConfig(config); err != nil {
		return config, fmt.Errorf("can't validate config: %w", err)
	}

	return config, nil
}

func applyDefaults(config *Config) {
	if config.ListenAddr == "" {
		config.ListenAddr = defaultListenAddr
	}
	if config.Metrics.Enabled && config.Metrics.Path == "" {
		config.Metrics.Path = defaultMetricsPath
	}

	for i := range config.Endpoints {
		endpoint := &config.Endpoints[i]
		if endpoint.Chaos.ErrorStatus == 0 {
			endpoint.Chaos.ErrorStatus = defaultErrorStatus
		}
		if endpoint.Response != nil && endpoint.Response.Status == 0 {
			endpoint.Response.Status = defaultResponseStatus
		}
		if endpoint.Backend != nil && endpoint.Backend.Timeout == 0 {
			endpoint.Backend.Timeout = defaultBackendTimeout
		}
	}
}
