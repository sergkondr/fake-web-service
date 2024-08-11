package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	defaultNamespace = "fakesvc"
)

type prometheusMW struct {
	registry prometheus.Registerer

	requestDuration *prometheus.HistogramVec
	requestsTotal   *prometheus.CounterVec
}

func newPrometheusMetrics(reg prometheus.Registerer) func(next http.Handler) http.Handler {
	m := prometheusMW{
		registry: reg,
	}

	m.requestDuration = promauto.With(m.registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: defaultNamespace,
			Name:      "response_time_seconds",
			Help:      "Histogram of response time in seconds aggregated by uri, method and status code",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"uri", "method", "status_code"})

	m.requestsTotal = promauto.With(m.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: defaultNamespace,
			Name:      "request_count",
			Help:      "How many HTTP requests processed, aggregated by uri, method and status code",
		},
		[]string{"uri", "method", "status_code"},
	)

	return m.handler
}

func (m *prometheusMW) handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		m.requestsTotal.WithLabelValues(r.RequestURI, r.Method, strconv.Itoa(ww.Status())).Inc()
		m.requestDuration.WithLabelValues(r.RequestURI, r.Method, strconv.Itoa(ww.Status())).Observe(time.Since(start).Seconds())
	}

	return http.HandlerFunc(fn)
}
