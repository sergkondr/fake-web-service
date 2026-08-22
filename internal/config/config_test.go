package config

import (
	"strings"
	"testing"
	"time"
)

func Test_validateConfig(t *testing.T) {
	tests := []struct {
		name            string
		cfg             Config
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "valid config",
			cfg: Config{ListenAddr: ":8080", HTTPEndpoints: []HTTPEndpoint{
				{
					Path: "/test", ErrorRate: 0.0, Slowness: Slowness{1 * time.Second, 3 * time.Second, 2 * time.Second},
				},
			}},
			wantErr: false,
		},
		{
			name:    "invalid config, no endpoints",
			cfg:     Config{ListenAddr: ":8080", HTTPEndpoints: []HTTPEndpoint{}},
			wantErr: true,
		},
		{
			name: "invalid config, duplicate paths",
			cfg: Config{ListenAddr: ":8080", HTTPEndpoints: []HTTPEndpoint{
				{
					Path: "/test", ErrorRate: 0.0, Slowness: Slowness{1 * time.Second, 3 * time.Second, 2 * time.Second},
				},
				{
					Path: "/test", ErrorRate: 0.0, Slowness: Slowness{1 * time.Second, 3 * time.Second, 2 * time.Second},
				},
			}},
			wantErr: true,
		},
		{
			name: "invalid config, min > max",
			cfg: Config{ListenAddr: ":8080", HTTPEndpoints: []HTTPEndpoint{
				{
					Path: "/test", ErrorRate: 0.0, Slowness: Slowness{5 * time.Second, 3 * time.Second, 2 * time.Second},
				},
			}},
			wantErr: true,
		},
		{
			name: "invalid config, wrong error rate",
			cfg: Config{ListenAddr: ":8080", HTTPEndpoints: []HTTPEndpoint{
				{
					Path: "/test", ErrorRate: 10.0, Slowness: Slowness{1 * time.Second, 3 * time.Second, 2 * time.Second},
				},
			}},
			wantErr: true,
		},
		{
			name: "invalid config, multiple ws endpoints",
			cfg: Config{
				ListenAddr:    ":8080",
				HTTPEndpoints: []HTTPEndpoint{{Path: "/test"}},
				WSEndpoints: []WSEndpoint{
					{
						Path: "/echo", Type: "echo",
					},
					{
						Path: "/random", Type: "random",
					},
				},
			},
			wantErr:         true,
			wantErrContains: "only one websocket endpoint",
		},
		{
			name: "invalid config, unknown endpoint type",
			cfg: Config{
				ListenAddr:    ":8080",
				HTTPEndpoints: []HTTPEndpoint{{Path: "/test"}},
				WSEndpoints: []WSEndpoint{
					{
						Path: "/echo", Type: "qwerty",
					},
				},
			},
			wantErr:         true,
			wantErrContains: "only echo websocket endpoints",
		},
		{
			name: "invalid config, endpoint path overlaps metric path",
			cfg: Config{
				ListenAddr: ":8080", Metrics: Metrics{Enabled: true, Path: "/metrics"},
				HTTPEndpoints: []HTTPEndpoint{
					{
						Path: "/metrics", ErrorRate: 0.0, Slowness: Slowness{1 * time.Second, 3 * time.Second, 2 * time.Second},
					},
				},
			},
			wantErr:         true,
			wantErrContains: "prometheus metrics path",
		},
		{
			name: "valid config, monitoring enabled",
			cfg: Config{
				ListenAddr: ":8080", Metrics: Metrics{Enabled: true, Path: "/metrics"},
				HTTPEndpoints: []HTTPEndpoint{
					{
						Path: "/path", ErrorRate: 0.0, Slowness: Slowness{1 * time.Second, 3 * time.Second, 2 * time.Second},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config, overlap with healthcheck path",
			cfg: Config{
				ListenAddr: ":8080", HTTPEndpoints: []HTTPEndpoint{
					{
						Path: "/healthz", ErrorRate: 0.0, Slowness: Slowness{1 * time.Second, 3 * time.Second, 2 * time.Second},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "invalid config, empty endpoint path",
			cfg:     Config{HTTPEndpoints: []HTTPEndpoint{{Path: ""}}},
			wantErr: true,
		},
		{
			name:    "invalid config, endpoint path without slash",
			cfg:     Config{HTTPEndpoints: []HTTPEndpoint{{Path: "test"}}},
			wantErr: true,
		},
		{
			name:    "invalid config, root endpoint path",
			cfg:     Config{HTTPEndpoints: []HTTPEndpoint{{Path: "/"}}},
			wantErr: true,
		},
		{
			name:    "invalid config, endpoint path overlaps websocket root",
			cfg:     Config{HTTPEndpoints: []HTTPEndpoint{{Path: "/ws"}}},
			wantErr: true,
		},
		{
			name:    "invalid config, endpoint path overlaps websocket endpoint",
			cfg:     Config{HTTPEndpoints: []HTTPEndpoint{{Path: "/ws/echo"}}},
			wantErr: true,
		},
		{
			name: "invalid config, empty websocket path",
			cfg: Config{
				HTTPEndpoints: []HTTPEndpoint{{Path: "/test"}},
				WSEndpoints:   []WSEndpoint{{Path: "", Type: "echo"}},
			},
			wantErr: true,
		},
		{
			name: "invalid config, websocket path without slash",
			cfg: Config{
				HTTPEndpoints: []HTTPEndpoint{{Path: "/test"}},
				WSEndpoints:   []WSEndpoint{{Path: "echo", Type: "echo"}},
			},
			wantErr: true,
		},
		{
			name: "invalid config, websocket root path",
			cfg: Config{
				HTTPEndpoints: []HTTPEndpoint{{Path: "/test"}},
				WSEndpoints:   []WSEndpoint{{Path: "/", Type: "echo"}},
			},
			wantErr: true,
		},
		{
			name: "invalid config, metrics path is reserved",
			cfg: Config{
				Metrics:       Metrics{Enabled: true, Path: "/ws"},
				HTTPEndpoints: []HTTPEndpoint{{Path: "/test"}},
			},
			wantErr:         true,
			wantErrContains: "reserved for WebSocket endpoints",
		},
		{
			name: "invalid config, negative minimum slowness",
			cfg: Config{HTTPEndpoints: []HTTPEndpoint{{
				Path: "/test", Slowness: Slowness{Min: -time.Second, P95: time.Second, Max: 2 * time.Second},
			}}},
			wantErr:         true,
			wantErrContains: "cannot be negative",
		},
		{
			name: "invalid config, negative p95 slowness",
			cfg: Config{HTTPEndpoints: []HTTPEndpoint{{
				Path: "/test", Slowness: Slowness{Min: -2 * time.Second, P95: -time.Second, Max: 0},
			}}},
			wantErr:         true,
			wantErrContains: "cannot be negative",
		},
		{
			name: "invalid config, negative maximum slowness",
			cfg: Config{HTTPEndpoints: []HTTPEndpoint{{
				Path: "/test", Slowness: Slowness{Min: -3 * time.Second, P95: -2 * time.Second, Max: -time.Second},
			}}},
			wantErr:         true,
			wantErrContains: "cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrContains)) {
				t.Errorf("validateConfig() error = %v, want error containing %q", err, tt.wantErrContains)
			}
		})
	}
}
