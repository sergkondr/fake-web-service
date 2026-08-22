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
	Registry *prometheus.Registry

	requestDuration *prometheus.HistogramVec
	requestsTotal   *prometheus.CounterVec

	webSocketConnectionsActive *prometheus.GaugeVec
	webSocketConnectionsTotal  *prometheus.CounterVec
	webSocketMessagesTotal     *prometheus.CounterVec
	webSocketErrorsTotal       *prometheus.CounterVec
	proxyUpstreamDuration      *prometheus.HistogramVec
	proxyUpstreamErrorsTotal   *prometheus.CounterVec
}

func New(metricsNamespace string) MetricsServer {
	m := MetricsServer{Registry: prometheus.NewRegistry()}

	m.requestDuration = promauto.With(m.Registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration by configured endpoint, endpoint type, method and status code.",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"endpoint", "type", "method", "status_code"},
	)

	m.requestsTotal = promauto.With(m.Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "http_requests_total",
			Help:      "HTTP requests by configured endpoint, endpoint type, method and status code.",
		},
		[]string{"endpoint", "type", "method", "status_code"},
	)

	m.webSocketConnectionsActive = promauto.With(m.Registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "websocket_connections_active",
			Help:      "Currently active WebSocket connections by configured endpoint.",
		},
		[]string{"endpoint"},
	)
	m.webSocketConnectionsTotal = promauto.With(m.Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "websocket_connections_total",
			Help:      "Opened and closed WebSocket connections by configured endpoint.",
		},
		[]string{"endpoint", "state"},
	)
	m.webSocketMessagesTotal = promauto.With(m.Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "websocket_messages_total",
			Help:      "Received and sent WebSocket messages by configured endpoint.",
		},
		[]string{"endpoint", "direction"},
	)
	m.webSocketErrorsTotal = promauto.With(m.Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "websocket_errors_total",
			Help:      "WebSocket read and write errors by configured endpoint.",
		},
		[]string{"endpoint", "operation"},
	)
	m.proxyUpstreamDuration = promauto.With(m.Registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "proxy_upstream_duration_seconds",
			Help:      "Proxy upstream round-trip duration by configured endpoint.",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"endpoint"},
	)
	m.proxyUpstreamErrorsTotal = promauto.With(m.Registry).NewCounterVec(
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

// WebSocketObserver returns an observer for WebSocket endpoint activity.
func (m *MetricsServer) WebSocketObserver() ws.Observer {
	return webSocketObserver{metrics: m}
}

// ProxyObserver returns an observer for proxy upstream requests.
func (m *MetricsServer) ProxyObserver() httpendpoint.ProxyObserver {
	return proxyObserver{metrics: m}
}

// MetricsHandler returns an HTTP handler that exposes this instance's registry.
func (m *MetricsServer) MetricsHandler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{Registry: m.Registry})
}

type webSocketObserver struct {
	metrics *MetricsServer
}

type proxyObserver struct {
	metrics *MetricsServer
}

func (observer proxyObserver) UpstreamRequest(endpoint string, duration time.Duration) {
	observer.metrics.proxyUpstreamDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
}

func (observer proxyObserver) UpstreamError(endpoint string) {
	observer.metrics.proxyUpstreamErrorsTotal.WithLabelValues(endpoint).Inc()
}

func (observer webSocketObserver) ConnectionOpened(endpoint string) {
	observer.metrics.webSocketConnectionsActive.WithLabelValues(endpoint).Inc()
	observer.metrics.webSocketConnectionsTotal.WithLabelValues(endpoint, "opened").Inc()
}

func (observer webSocketObserver) ConnectionClosed(endpoint string) {
	observer.metrics.webSocketConnectionsActive.WithLabelValues(endpoint).Dec()
	observer.metrics.webSocketConnectionsTotal.WithLabelValues(endpoint, "closed").Inc()
}

func (observer webSocketObserver) MessageReceived(endpoint string) {
	observer.metrics.webSocketMessagesTotal.WithLabelValues(endpoint, "received").Inc()
}

func (observer webSocketObserver) MessageSent(endpoint string) {
	observer.metrics.webSocketMessagesTotal.WithLabelValues(endpoint, "sent").Inc()
}

func (observer webSocketObserver) ReadError(endpoint string) {
	observer.metrics.webSocketErrorsTotal.WithLabelValues(endpoint, "read").Inc()
}

func (observer webSocketObserver) WriteError(endpoint string) {
	observer.metrics.webSocketErrorsTotal.WithLabelValues(endpoint, "write").Inc()
}
