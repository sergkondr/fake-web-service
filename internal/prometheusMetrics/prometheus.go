package prometheusMetrics

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsServer struct {
	Registry *prometheus.Registry

	requestDuration *prometheus.HistogramVec
	requestsTotal   *prometheus.CounterVec
}

func New(metricsNamespace string) MetricsServer {
	m := MetricsServer{
		Registry: prometheus.NewRegistry(),
	}

	m.requestDuration = promauto.With(m.Registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "response_time_seconds",
			Help:      "Histogram of response time in seconds aggregated by uri, method and status code",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"uri", "method", "status_code"})

	m.requestsTotal = promauto.With(m.Registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "request_count",
			Help:      "How many HTTP requests processed, aggregated by uri, method and status code",
		},
		[]string{"uri", "method", "status_code"},
	)

	slog.Debug("prometheus metrics server initialized")

	return m
}

// MiddlewareHandler takes an http.Handler as input and returns an http.Handler.
// It wraps the provided handler with additional functionality to measure request
// duration and increment request counters for Prometheus metrics.
func (m *MetricsServer) MiddlewareHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		m.requestsTotal.WithLabelValues(r.RequestURI, r.Method, strconv.Itoa(ww.Status())).Inc()
		m.requestDuration.WithLabelValues(r.RequestURI, r.Method, strconv.Itoa(ww.Status())).Observe(time.Since(start).Seconds())
	}

	return http.HandlerFunc(fn)
}

// MetricsHandler returns an http.Handler that handles requests to metrics gathering endpoint.
func (m *MetricsServer) MetricsHandler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{
		Registry: m.Registry,
	})
}
