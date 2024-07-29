package config

import (
	"testing"
	"time"
)

func Test_validateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateConfig(tt.cfg); (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}