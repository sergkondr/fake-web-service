package prometheusMetrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func Test_prometheusMW_handler(t *testing.T) {
	tests := []struct {
		name        string
		requestPath string
		wantBody    string
		wantErr     bool
	}{
		{
			name:        "test request #1",
			requestPath: "/test",
			wantBody:    "Test",
			wantErr:     false,
		},
		{
			name:        "request to metrics #1",
			requestPath: "/metrics",
			wantBody:    `fakesvc_request_count{method="GET",status_code="200",uri=""} 1`,
			wantErr:     false,
		},
		{
			name:        "test request #2",
			requestPath: "/test",
			wantBody:    "Test",
			wantErr:     false,
		},
		{
			name:        "request to metrics #2",
			requestPath: "/metrics",
			wantBody:    `fakesvc_request_count{method="GET",status_code="200",uri=""} 2`,
			wantErr:     false,
		},
	}

	prom := New("fakesvc")

	r := chi.NewRouter()
	r.Handle("/metrics", prom.MetricsHandler())

	r.Route("/", func(r chi.Router) {
		r.Use(prom.MiddlewareHandler)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Test"))
		})
	})

	recorder := httptest.NewRecorder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, tt.requestPath, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("request error = %v, wantErr %v", err, tt.wantErr)
			}

			r.ServeHTTP(recorder, req)
			body := recorder.Body.String()
			if !strings.Contains(body, tt.wantBody) {
				t.Errorf("Expected response body to contain: %s\n, but got: %s", tt.wantBody, body)
			}
		})
	}
}
