package main

import (
	"net/http"
	"testing"
)

func TestNewServerConfiguresTimeouts(t *testing.T) {
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	server := newServer("127.0.0.1:8080", handler)

	if server.Addr != "127.0.0.1:8080" {
		t.Errorf("Addr = %q, want %q", server.Addr, "127.0.0.1:8080")
	}
	if server.Handler == nil {
		t.Error("Handler must be configured")
	}
	if server.ReadHeaderTimeout != readHeaderTimeout {
		t.Errorf("ReadHeaderTimeout = %s, want %s", server.ReadHeaderTimeout, readHeaderTimeout)
	}
	if server.ReadTimeout != readTimeout {
		t.Errorf("ReadTimeout = %s, want %s", server.ReadTimeout, readTimeout)
	}
	if server.WriteTimeout != writeTimeout {
		t.Errorf("WriteTimeout = %s, want %s", server.WriteTimeout, writeTimeout)
	}
	if server.IdleTimeout != idleTimeout {
		t.Errorf("IdleTimeout = %s, want %s", server.IdleTimeout, idleTimeout)
	}
}

func TestRunReturnsConfigError(t *testing.T) {
	err := run(t.Context(), []string{"--config", "does-not-exist.yaml"})
	if err == nil {
		t.Fatal("run() error = nil, want config error")
	}
}
