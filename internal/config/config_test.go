package config

import (
	"strings"
	"testing"
	"time"
)

func TestParseConfig(t *testing.T) {
	configYAML := `
metrics:
  enabled: true

endpoints:
  - name: Static response
    type: http
    path: /good
    response:
      headers:
        Content-Type: application/json
      body: '{"status":"ok"}'
    chaos:
      latency:
        min: 10ms
        p95: 50ms
        max: 100ms

  - name: Backend
    type: proxy
    path: /backend
    backend:
      url: https://service.example.com/api/v1

  - name: Echo
    type: ws/echo
    path: /ws/echo

  - name: Clock
    type: ws/stream
    path: /ws/time
    stream:
      interval: 1s
`

	cfg, err := parseConfig(strings.NewReader(configYAML))
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if cfg.ListenAddr != defaultListenAddr {
		t.Errorf("listen address = %q, want %q", cfg.ListenAddr, defaultListenAddr)
	}
	if cfg.Metrics.Path != defaultMetricsPath {
		t.Errorf("metrics path = %q, want %q", cfg.Metrics.Path, defaultMetricsPath)
	}
	if len(cfg.Endpoints) != 4 {
		t.Fatalf("endpoint count = %d, want 4", len(cfg.Endpoints))
	}

	httpEndpoint := cfg.Endpoints[0]
	if httpEndpoint.Chaos.ErrorStatus != defaultErrorStatus {
		t.Errorf("HTTP error status = %d, want %d", httpEndpoint.Chaos.ErrorStatus, defaultErrorStatus)
	}
	if httpEndpoint.Response == nil || httpEndpoint.Response.Status != defaultResponseStatus {
		t.Fatalf("HTTP response = %#v, want default status %d", httpEndpoint.Response, defaultResponseStatus)
	}
	if httpEndpoint.Chaos.Latency == nil || httpEndpoint.Chaos.Latency.P95 != 50*time.Millisecond {
		t.Fatalf("HTTP latency = %#v, want p95 50ms", httpEndpoint.Chaos.Latency)
	}

	proxyEndpoint := cfg.Endpoints[1]
	if proxyEndpoint.Backend == nil || proxyEndpoint.Backend.Timeout != defaultBackendTimeout {
		t.Fatalf("proxy backend = %#v, want default timeout %s", proxyEndpoint.Backend, defaultBackendTimeout)
	}

	streamEndpoint := cfg.Endpoints[3]
	if streamEndpoint.Stream == nil || streamEndpoint.Stream.Format != defaultStreamFormat {
		t.Fatalf("stream config = %#v, want default format %q", streamEndpoint.Stream, defaultStreamFormat)
	}
}

func TestParseConfigRejectsUnknownAndLegacyFields(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
		wantError  string
	}{
		{
			name: "unknown endpoint field",
			configYAML: `
endpoints:
  - type: http
    path: /test
    error_rtae: 0.5
`,
			wantError: "field error_rtae not found",
		},
		{
			name: "legacy HTTP endpoints",
			configYAML: `
http_endpoints:
  - path: /test
`,
			wantError: "field http_endpoints not found",
		},
		{
			name: "legacy WebSocket endpoints",
			configYAML: `
ws_endpoints:
  - type: echo
    path: /echo
`,
			wantError: "field ws_endpoints not found",
		},
		{
			name: "legacy slowness field",
			configYAML: `
endpoints:
  - type: http
    path: /test
    slowness:
      min: 1ms
      p95: 2ms
      max: 3ms
`,
			wantError: "field slowness not found",
		},
		{
			name: "multiple YAML documents",
			configYAML: `
endpoints:
  - type: http
    path: /first
---
endpoints:
  - type: http
    path: /second
`,
			wantError: "multiple YAML documents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseConfig(strings.NewReader(tt.configYAML))
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("parseConfig() error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantError string
	}{
		{
			name: "valid config with all endpoint types",
			cfg: Config{Endpoints: []Endpoint{
				{Type: EndpointTypeHTTP, Path: "/http"},
				{Type: EndpointTypeProxy, Path: "/proxy", Backend: &BackendConfig{URL: "https://example.com"}},
				{Type: EndpointTypeWSEcho, Path: "/ws/echo"},
				{Type: EndpointTypeWSStream, Path: "/ws/time", Stream: &StreamConfig{Interval: time.Second}},
			}},
		},
		{
			name:      "no endpoints",
			cfg:       Config{},
			wantError: "no endpoints defined",
		},
		{
			name: "invalid listen address",
			cfg: Config{
				ListenAddr: "localhost",
				Endpoints:  []Endpoint{{Type: EndpointTypeHTTP, Path: "/test"}},
			},
			wantError: "host:port format",
		},
		{
			name: "invalid listen port",
			cfg: Config{
				ListenAddr: "localhost:70000",
				Endpoints:  []Endpoint{{Type: EndpointTypeHTTP, Path: "/test"}},
			},
			wantError: "between 1 and 65535",
		},
		{
			name:      "unknown type",
			cfg:       Config{Endpoints: []Endpoint{{Type: "unknown", Path: "/test"}}},
			wantError: "unsupported type",
		},
		{
			name:      "empty path",
			cfg:       Config{Endpoints: []Endpoint{{Type: EndpointTypeHTTP}}},
			wantError: "path cannot be empty",
		},
		{
			name:      "path without leading slash",
			cfg:       Config{Endpoints: []Endpoint{{Type: EndpointTypeHTTP, Path: "test"}}},
			wantError: "path must start with /",
		},
		{
			name:      "root path",
			cfg:       Config{Endpoints: []Endpoint{{Type: EndpointTypeHTTP, Path: "/"}}},
			wantError: "path is reserved",
		},
		{
			name:      "trailing slash",
			cfg:       Config{Endpoints: []Endpoint{{Type: EndpointTypeHTTP, Path: "/test/"}}},
			wantError: "trailing slash",
		},
		{
			name:      "non-normalized path",
			cfg:       Config{Endpoints: []Endpoint{{Type: EndpointTypeHTTP, Path: "/api//test"}}},
			wantError: "must be normalized",
		},
		{
			name:      "path with query",
			cfg:       Config{Endpoints: []Endpoint{{Type: EndpointTypeHTTP, Path: "/test?debug=true"}}},
			wantError: "query or fragment",
		},
		{
			name:      "healthcheck collision",
			cfg:       Config{Endpoints: []Endpoint{{Type: EndpointTypeHTTP, Path: "/healthz"}}},
			wantError: "reserved for healthcheck",
		},
		{
			name: "metrics collision",
			cfg: Config{
				Metrics:   Metrics{Enabled: true, Path: "/metrics"},
				Endpoints: []Endpoint{{Type: EndpointTypeHTTP, Path: "/metrics"}},
			},
			wantError: "overlaps with metrics path",
		},
		{
			name: "duplicate paths across types",
			cfg: Config{Endpoints: []Endpoint{
				{Type: EndpointTypeHTTP, Path: "/same"},
				{Type: EndpointTypeWSEcho, Path: "/same"},
			}},
			wantError: "duplicates endpoint[0]",
		},
		{
			name: "proxy owns nested path",
			cfg: Config{Endpoints: []Endpoint{
				{Type: EndpointTypeProxy, Path: "/api", Backend: &BackendConfig{URL: "https://example.com"}},
				{Type: EndpointTypeHTTP, Path: "/api/debug"},
			}},
			wantError: "is owned by proxy",
		},
		{
			name: "invalid error rate",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeHTTP, Path: "/test", Chaos: Chaos{ErrorRate: 1.1},
			}}},
			wantError: "error_rate",
		},
		{
			name: "invalid error status",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeHTTP, Path: "/test", Chaos: Chaos{ErrorStatus: 600},
			}}},
			wantError: "error_status",
		},
		{
			name: "negative latency",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeHTTP, Path: "/test",
				Chaos: Chaos{Latency: &Latency{Min: -time.Millisecond, P95: time.Millisecond, Max: 2 * time.Millisecond}},
			}}},
			wantError: "cannot be negative",
		},
		{
			name: "unordered latency",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeHTTP, Path: "/test",
				Chaos: Chaos{Latency: &Latency{Min: time.Second, P95: 500 * time.Millisecond, Max: 2 * time.Second}},
			}}},
			wantError: "min <= p95 <= max",
		},
		{
			name: "HTTP endpoint with backend",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeHTTP, Path: "/test", Backend: &BackendConfig{URL: "https://example.com"},
			}}},
			wantError: "does not allow backend",
		},
		{
			name:      "proxy without backend",
			cfg:       Config{Endpoints: []Endpoint{{Type: EndpointTypeProxy, Path: "/proxy"}}},
			wantError: "requires backend",
		},
		{
			name: "proxy with unsupported scheme",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeProxy, Path: "/proxy", Backend: &BackendConfig{URL: "ftp://example.com"},
			}}},
			wantError: "scheme must be http or https",
		},
		{
			name: "proxy URL without host",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeProxy, Path: "/proxy", Backend: &BackendConfig{URL: "https:///api"},
			}}},
			wantError: "contain a host",
		},
		{
			name: "proxy URL with userinfo",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeProxy, Path: "/proxy", Backend: &BackendConfig{URL: "https://user:pass@example.com"},
			}}},
			wantError: "must not contain userinfo",
		},
		{
			name: "proxy with invalid timeout",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeProxy, Path: "/proxy",
				Backend: &BackendConfig{URL: "https://example.com", Timeout: -time.Second},
			}}},
			wantError: "timeout must be greater than zero",
		},
		{
			name: "invalid response status",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeHTTP, Path: "/test", Response: &ResponseConfig{Status: 99},
			}}},
			wantError: "status must be between",
		},
		{
			name: "invalid response header name",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeHTTP, Path: "/test",
				Response: &ResponseConfig{Headers: map[string]string{"Bad Header": "value"}},
			}}},
			wantError: "invalid header name",
		},
		{
			name: "response header injection",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeHTTP, Path: "/test",
				Response: &ResponseConfig{Headers: map[string]string{"X-Test": "value\r\ninjected"}},
			}}},
			wantError: "contains a newline",
		},
		{
			name:      "stream without config",
			cfg:       Config{Endpoints: []Endpoint{{Type: EndpointTypeWSStream, Path: "/ws/time"}}},
			wantError: "requires stream config",
		},
		{
			name: "stream without interval",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeWSStream, Path: "/ws/time", Stream: &StreamConfig{},
			}}},
			wantError: "interval must be greater than zero",
		},
		{
			name: "stream with unknown format",
			cfg: Config{Endpoints: []Endpoint{{
				Type: EndpointTypeWSStream, Path: "/ws/time",
				Stream: &StreamConfig{Interval: time.Second, Format: "xml"},
			}}},
			wantError: "unsupported format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			applyDefaults(&cfg)
			err := validateConfig(cfg)

			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("validateConfig() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("validateConfig() error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}

func TestExampleConfigIsRunnable(t *testing.T) {
	if _, err := Get("../../examples/config.yaml"); err != nil {
		t.Fatalf("Get(example config) error = %v", err)
	}
}
