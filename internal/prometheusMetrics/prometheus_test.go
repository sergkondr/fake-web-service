package prometheusMetrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestPrometheusMiddlewareGroupsRequestsByPath(t *testing.T) {
	prom := New("fakesvc")

	router := chi.NewRouter()
	router.Handle("/metrics", prom.MetricsHandler())
	router.Group(func(r chi.Router) {
		r.Use(prom.MiddlewareHandler)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Test"))
		})
	})

	for _, target := range []string{
		"/test?request_id=123",
		"/test?request_id=456",
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, target, nil)

		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("request %q returned status %d, want %d", target, recorder.Code, http.StatusOK)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(recorder, request)

	body := recorder.Body.String()
	wantCounter := `fakesvc_request_count{method="GET",status_code="200",uri="/test"} 2`
	if !strings.Contains(body, wantCounter) {
		t.Fatalf("metrics body does not contain %q:\n%s", wantCounter, body)
	}
	if strings.Contains(body, "request_id") {
		t.Fatalf("metrics body contains query parameter and can create high-cardinality series:\n%s", body)
	}
}
