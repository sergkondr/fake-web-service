package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sergkondr/fake-web-service/internal/config"
)

func TestGetDelayReturnsValueWithinConfiguredRange(t *testing.T) {
	tests := []struct {
		name string
		min  time.Duration
		p95  time.Duration
		max  time.Duration
	}{
		{
			name: "regular range",
			min:  10 * time.Millisecond,
			p95:  50 * time.Millisecond,
			max:  100 * time.Millisecond,
		},
		{
			name: "min equals p95",
			min:  10 * time.Millisecond,
			p95:  10 * time.Millisecond,
			max:  100 * time.Millisecond,
		},
		{
			name: "p95 equals max",
			min:  10 * time.Millisecond,
			p95:  100 * time.Millisecond,
			max:  100 * time.Millisecond,
		},
		{
			name: "all values equal",
			min:  10 * time.Millisecond,
			p95:  10 * time.Millisecond,
			max:  10 * time.Millisecond,
		},
		{
			name: "sub-millisecond range",
			min:  time.Nanosecond,
			p95:  2 * time.Nanosecond,
			max:  3 * time.Nanosecond,
		},
		{
			name: "zero delay",
			min:  0,
			p95:  0,
			max:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				delay := getDelay(tt.min, tt.max, tt.p95)
				if delay < tt.min || delay > tt.max {
					t.Fatalf("getDelay() = %v, want value in [%v, %v]", delay, tt.min, tt.max)
				}
			}
		})
	}
}

func TestDeceleratorCallsNextHandler(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := decelerator(structuredEndpoint(0))(next)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler.ServeHTTP(recorder, request)

	if !called {
		t.Fatal("decelerator did not call the next handler")
	}
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func structuredEndpoint(delay time.Duration) config.HTTPEndpoint {
	return config.HTTPEndpoint{
		Slowness: config.Slowness{
			Min: delay,
			P95: delay,
			Max: delay,
		},
	}
}
