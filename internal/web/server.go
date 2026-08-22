package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sergkondr/fake-web-service/internal/config"
	"github.com/sergkondr/fake-web-service/internal/prometheusMetrics"
	httpendpoint "github.com/sergkondr/fake-web-service/internal/web/http"
	"github.com/sergkondr/fake-web-service/internal/web/middleware"
	"github.com/sergkondr/fake-web-service/internal/web/ws"
)

func New(cfg config.Config) (chi.Router, error) {
	r := chi.NewRouter()

	var httpMetricsMiddleware func(endpoint, endpointType string) func(next http.Handler) http.Handler
	var proxyObserver func(endpoint string) httpendpoint.ProxyObserver
	webSocketObserver := func(string) ws.Observer { return nil }
	if cfg.Metrics.Enabled {
		metrics := prometheusMetrics.New("fakesvc")
		r.Handle(cfg.Metrics.Path, metrics.MetricsHandler())
		httpMetricsMiddleware = metrics.EndpointMiddleware
		proxyObserver = metrics.ProxyObserver
		webSocketObserver = metrics.WebSocketObserver
	}

	var endpoints strings.Builder
	for _, endpoint := range cfg.Endpoints {
		if err := registerEndpoint(r, cfg.Hostname, endpoint, httpMetricsMiddleware, webSocketObserver, proxyObserver); err != nil {
			return nil, err
		}
		if !endpoint.Hidden {
			_, _ = fmt.Fprintf(&endpoints, "- %s - %s: %s\n", endpoint.Path, endpoint.Name, endpoint.Description)
		}
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "Available endpoints:\n%s\nHostname: %s\n", endpoints.String(), cfg.Hostname)
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return r, nil
}

func registerEndpoint(
	r chi.Router,
	hostname string,
	endpoint config.Endpoint,
	httpMetricsMiddleware func(endpoint, endpointType string) func(next http.Handler) http.Handler,
	webSocketObserver func(endpoint string) ws.Observer,
	proxyObserver func(endpoint string) httpendpoint.ProxyObserver,
) error {
	switch endpoint.Type {
	case config.EndpointTypeHTTP:
		route := endpointRouter(r, endpoint, httpMetricsMiddleware)
		handler := httpendpoint.Static(hostname, endpoint)
		route.MethodFunc(http.MethodGet, endpoint.Path, handler)
		route.MethodFunc(http.MethodHead, endpoint.Path, handler)
	case config.EndpointTypeWSEcho:
		route := endpointRouter(r, endpoint, nil)
		route.HandleFunc(endpoint.Path, ws.Echo(hostname, endpoint.Path, webSocketObserver(endpoint.Path)))
	case config.EndpointTypeWSStream:
		if endpoint.Stream == nil {
			return fmt.Errorf("endpoint %q of type %q has no stream config", endpoint.Path, endpoint.Type)
		}
		route := endpointRouter(r, endpoint, nil)
		route.HandleFunc(endpoint.Path, ws.Stream(hostname, endpoint.Path, endpoint.Stream.Interval, webSocketObserver(endpoint.Path)))
	case config.EndpointTypeProxy:
		if endpoint.Backend == nil {
			return fmt.Errorf("endpoint %q of type %q has no backend config", endpoint.Path, endpoint.Type)
		}
		var observer httpendpoint.ProxyObserver
		if proxyObserver != nil {
			observer = proxyObserver(endpoint.Path)
		}
		handler, err := httpendpoint.Proxy(endpoint.Path, *endpoint.Backend, observer)
		if err != nil {
			return fmt.Errorf("create proxy for endpoint %q: %w", endpoint.Path, err)
		}
		route := endpointRouter(r, endpoint, httpMetricsMiddleware)
		route.Handle(endpoint.Path, handler)
		route.Handle(endpoint.Path+"/*", handler)
	default:
		return fmt.Errorf("endpoint %q has unsupported type %q", endpoint.Path, endpoint.Type)
	}

	return nil
}

func endpointRouter(
	r chi.Router,
	endpoint config.Endpoint,
	httpMetricsMiddleware func(endpoint, endpointType string) func(next http.Handler) http.Handler,
) chi.Router {
	route := r.With()
	useEndpointMiddleware(route, endpoint, httpMetricsMiddleware)
	return route
}

func useEndpointMiddleware(
	r chi.Router,
	endpoint config.Endpoint,
	httpMetricsMiddleware func(endpoint, endpointType string) func(next http.Handler) http.Handler,
) {
	if !endpoint.DoNotLog {
		r.Use(middleware.Logger(endpoint.Path, string(endpoint.Type)))
	}
	if httpMetricsMiddleware != nil {
		r.Use(httpMetricsMiddleware(endpoint.Path, string(endpoint.Type)))
	}
	r.Use(middleware.Chaos(endpoint.Chaos))
}
