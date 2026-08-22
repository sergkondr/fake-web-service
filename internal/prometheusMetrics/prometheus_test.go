package prometheusMetrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sergkondr/fake-web-service/internal/web/ws"
)

func TestPrometheusMiddlewareGroupsRequestsByPath(t *testing.T) {
	prom := New("fakesvc")

	router := chi.NewRouter()
	router.Handle("/metrics", prom.MetricsHandler())
	router.Group(func(r chi.Router) {
		r.Use(prom.EndpointMiddleware("/test", "http"))
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
	wantCounter := `fakesvc_http_requests_total{endpoint="/test",method="GET",status_code="200",type="http"} 2`
	if !strings.Contains(body, wantCounter) {
		t.Fatalf("metrics body does not contain %q:\n%s", wantCounter, body)
	}
	if strings.Contains(body, "request_id") {
		t.Fatalf("metrics body contains query parameter and can create high-cardinality series:\n%s", body)
	}
}

func TestWebSocketMetricsUseConfiguredEndpoint(t *testing.T) {
	prom := New("fakesvc")
	observer := prom.WebSocketObserver("/time")

	observer(ws.ConnectionOpened)
	observer(ws.MessageSent)
	observer(ws.MessageReceived)
	observer(ws.ReadError)
	observer(ws.ConnectionClosed)

	recorder := httptest.NewRecorder()
	prom.MetricsHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := recorder.Body.String()

	for _, want := range []string{
		`fakesvc_websocket_connections_active{endpoint="/time"} 0`,
		`fakesvc_websocket_connections_total{endpoint="/time",state="closed"} 1`,
		`fakesvc_websocket_connections_total{endpoint="/time",state="opened"} 1`,
		`fakesvc_websocket_messages_total{direction="received",endpoint="/time"} 1`,
		`fakesvc_websocket_messages_total{direction="sent",endpoint="/time"} 1`,
		`fakesvc_websocket_errors_total{endpoint="/time",operation="read"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics body does not contain %q:\n%s", want, body)
		}
	}
}
