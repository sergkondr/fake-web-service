package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sergkondr/fake-web-service/internal/config"
)

func TestHTTPRouteUsesConfiguredPathWithoutTrailingSlash(t *testing.T) {
	router := newTestRouter(t, config.Config{
		Hostname: "test-host",
		Endpoints: []config.Endpoint{
			{Type: config.EndpointTypeHTTP, Path: "/good"},
		},
	})

	t.Run("configured path", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/good", nil)

		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
		}
		if body := recorder.Body.String(); !strings.Contains(body, "success: test-host/good") {
			t.Fatalf("body = %q, want successful response", body)
		}
	})

	t.Run("trailing slash is a different path", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/good/", nil)

		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNotFound)
		}
	})
}

func TestHTTPEndpointUsesConfiguredResponse(t *testing.T) {
	router := newTestRouter(t, config.Config{
		Endpoints: []config.Endpoint{
			{
				Type: config.EndpointTypeHTTP,
				Path: "/custom",
				Response: &config.ResponseConfig{
					Status:  http.StatusCreated,
					Headers: map[string]string{"X-Test": "configured"},
					Body:    "created",
				},
			},
		},
	})

	t.Run("GET", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/custom", nil)

		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusCreated)
		}
		if got := recorder.Header().Get("X-Test"); got != "configured" {
			t.Errorf("X-Test header = %q, want %q", got, "configured")
		}
		if body := recorder.Body.String(); body != "created" {
			t.Errorf("body = %q, want %q", body, "created")
		}
	})

	t.Run("HEAD", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodHead, "/custom", nil)

		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusCreated)
		}
		if body := recorder.Body.String(); body != "" {
			t.Errorf("body = %q, want empty response body", body)
		}
	})
}

func TestRouterRejectsEndpointTypesNotImplementedYet(t *testing.T) {
	_, err := New(config.Config{Endpoints: []config.Endpoint{{
		Type: config.EndpointTypeProxy,
		Path: "/proxy",
	}}})
	if err == nil || !strings.Contains(err.Error(), "not implemented yet") {
		t.Fatalf("New() error = %v, want not implemented error", err)
	}
}

func TestEndpointChaosUsesConfiguredErrorStatus(t *testing.T) {
	router := newTestRouter(t, config.Config{Endpoints: []config.Endpoint{{
		Type:  config.EndpointTypeHTTP,
		Path:  "/unavailable",
		Chaos: config.Chaos{ErrorRate: 1, ErrorStatus: http.StatusServiceUnavailable},
	}}})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/unavailable", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestIndexListsVisibleEndpointsOfAllImplementedTypes(t *testing.T) {
	router := newTestRouter(t, config.Config{Endpoints: []config.Endpoint{
		{Type: config.EndpointTypeHTTP, Path: "/good", Name: "Good"},
		{Type: config.EndpointTypeWSEcho, Path: "/echo", Name: "Echo"},
		{Type: config.EndpointTypeHTTP, Path: "/hidden", Hidden: true},
	}})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, request)

	body := recorder.Body.String()
	if !strings.Contains(body, "/good") || !strings.Contains(body, "/echo") {
		t.Fatalf("index body = %q, want visible endpoint paths", body)
	}
	if strings.Contains(body, "/hidden") {
		t.Fatalf("index body = %q, must not contain hidden endpoint", body)
	}
}

func TestExampleConfigCreatesRouter(t *testing.T) {
	cfg, err := config.Get("../../examples/config.yaml")
	if err != nil {
		t.Fatalf("Get(example config) error = %v", err)
	}
	router := newTestRouter(t, cfg)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/good", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func newTestRouter(t *testing.T, cfg config.Config) http.Handler {
	t.Helper()

	router, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return router
}
