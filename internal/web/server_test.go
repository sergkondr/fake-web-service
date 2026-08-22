package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sergkondr/fake-web-service/internal/config"
)

func TestHTTPRouteUsesConfiguredPathWithoutTrailingSlash(t *testing.T) {
	router := New(config.Config{
		Hostname: "test-host",
		HTTPEndpoints: []config.HTTPEndpoint{
			{Path: "/good"},
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
