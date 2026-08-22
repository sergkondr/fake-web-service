package prometheusMetrics

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpendpoint "github.com/sergkondr/fake-web-service/internal/web/http"
	"github.com/sergkondr/fake-web-service/internal/web/ws"
)

type MetricsServer struct {
	registry *prometheus.Registry

	requestDuration *prometheus.HistogramVec
	requestsTotal   *prometheus.CounterVec

	webSocketConnectionsActive *prometheus.GaugeVec
	webSocketConnectionsTotal  *prometheus.CounterVec
	webSocketMessagesTotal     *prometheus.CounterVec
	webSocketErrorsTotal       *prometheus.CounterVec
	proxyUpstreamDuration      *prometheus.HistogramVec
	proxyUpstreamErrorsTotal   *prometheus.CounterVec
}

func New(metricsNamespace string) *MetricsServer {
	m := &MetricsServer{registry: prometheus.NewRegistry()}

	m.requestDuration = promauto.With(m.registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration by configured endpoint, endpoint type, method and status code.",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"endpoint", "type", "method", "status_code"},
	)

	m.requestsTotal = promauto.With(m.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "http_requests_total",
			Help:      "HTTP requests by configured endpoint, endpoint type, method and status code.",
		},
		[]string{"endpoint", "type", "method", "status_code"},
	)

	m.webSocketConnectionsActive = promauto.With(m.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "websocket_connections_active",
			Help:      "Currently active WebSocket connections by configured endpoint.",
		},
		[]string{"endpoint"},
	)
	m.webSocketConnectionsTotal = promauto.With(m.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "websocket_connections_total",
			Help:      "Opened and closed WebSocket connections by configured endpoint.",
		},
		[]string{"endpoint", "state"},
	)
	m.webSocketMessagesTotal = promauto.With(m.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "websocket_messages_total",
			Help:      "Received and sent WebSocket messages by configured endpoint.",
		},
		[]string{"endpoint", "direction"},
	)
	m.webSocketErrorsTotal = promauto.With(m.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "websocket_errors_total",
			Help:      "WebSocket read and write errors by configured endpoint.",
		},
		[]string{"endpoint", "operation"},
	)
	m.proxyUpstreamDuration = promauto.With(m.registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "proxy_upstream_duration_seconds",
			Help:      "Proxy upstream round-trip duration by configured endpoint.",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"endpoint"},
	)
	m.proxyUpstreamErrorsTotal = promauto.With(m.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "proxy_upstream_errors_total",
			Help:      "Proxy upstream errors by configured endpoint.",
		},
		[]string{"endpoint"},
	)

	slog.Debug("prometheus metrics server initialized")
	return m
}

// EndpointMiddleware records HTTP requests using a configured endpoint identity.
func (m *MetricsServer) EndpointMiddleware(endpoint, endpointType string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			responseWriter := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(responseWriter, r)

			statusCode := strconv.Itoa(responseWriter.Status())
			m.requestsTotal.WithLabelValues(endpoint, endpointType, r.Method, statusCode).Inc()
			m.requestDuration.WithLabelValues(endpoint, endpointType, r.Method, statusCode).Observe(time.Since(start).Seconds())
		})
	}
}

// WebSocketObserver returns an observer scoped to one configured endpoint.
func (m *MetricsServer) WebSocketObserver(endpoint string) ws.Observer {
	return func(event ws.Event) {
		switch event {
		case ws.ConnectionOpened:
			m.webSocketConnectionsActive.WithLabelValues(endpoint).Inc()
			m.webSocketConnectionsTotal.WithLabelValues(endpoint, "opened").Inc()
		case ws.ConnectionClosed:
			m.webSocketConnectionsActive.WithLabelValues(endpoint).Dec()
			m.webSocketConnectionsTotal.WithLabelValues(endpoint, "closed").Inc()
		case ws.MessageReceived:
			m.webSocketMessagesTotal.WithLabelValues(endpoint, "received").Inc()
		case ws.MessageSent:
			m.webSocketMessagesTotal.WithLabelValues(endpoint, "sent").Inc()
		case ws.ReadError:
			m.webSocketErrorsTotal.WithLabelValues(endpoint, "read").Inc()
		case ws.WriteError:
			m.webSocketErrorsTotal.WithLabelValues(endpoint, "write").Inc()
		}
	}
}

// ProxyObserver returns an observer scoped to one configured endpoint.
func (m *MetricsServer) ProxyObserver(endpoint string) httpendpoint.ProxyObserver {
	return func(duration time.Duration, err error) {
		m.proxyUpstreamDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
		if err != nil {
			m.proxyUpstreamErrorsTotal.WithLabelValues(endpoint).Inc()
		}
	}
}

// MetricsHandler returns an HTTP handler that exposes this instance's registry.
func (m *MetricsServer) MetricsHandler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{Registry: m.registry})
}
