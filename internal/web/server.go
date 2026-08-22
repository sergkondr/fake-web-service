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
	var proxyObserver httpendpoint.ProxyObserver
	var webSocketObserver ws.Observer
	if cfg.Metrics.Enabled {
		metrics := prometheusMetrics.New("fakesvc")
		r.Handle(cfg.Metrics.Path, metrics.MetricsHandler())
		httpMetricsMiddleware = metrics.EndpointMiddleware
		proxyObserver = metrics.ProxyObserver()
		webSocketObserver = metrics.WebSocketObserver()
	}

	var endpoints strings.Builder
	for _, endpoint := range cfg.Endpoints {
		if err := registerEndpoint(r, cfg, endpoint, httpMetricsMiddleware, webSocketObserver, proxyObserver); err != nil {
			return nil, err
		}
		if !endpoint.Hidden {
			endpoints.WriteString(fmt.Sprintf("- %s - %s: %s\n", endpoint.Path, endpoint.Name, endpoint.Description))
		}
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Available endpoints:\n%s\nHostname: %s\n", endpoints.String(), cfg.Hostname)))
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return r, nil
}

func registerEndpoint(
	r chi.Router,
	cfg config.Config,
	endpoint config.Endpoint,
	httpMetricsMiddleware func(endpoint, endpointType string) func(next http.Handler) http.Handler,
	webSocketObserver ws.Observer,
	proxyObserver httpendpoint.ProxyObserver,
) error {
	switch endpoint.Type {
	case config.EndpointTypeHTTP:
		r.Group(func(r chi.Router) {
			useEndpointMiddleware(r, endpoint, httpMetricsMiddleware)
			handler := httpendpoint.Static(cfg.Hostname, endpoint)
			r.MethodFunc(http.MethodGet, endpoint.Path, handler)
			r.MethodFunc(http.MethodHead, endpoint.Path, handler)
		})
	case config.EndpointTypeWSEcho:
		r.Group(func(r chi.Router) {
			useEndpointMiddleware(r, endpoint, nil)
			r.HandleFunc(endpoint.Path, ws.Echo(cfg.Hostname, endpoint.Path, webSocketObserver))
		})
	case config.EndpointTypeWSStream:
		if endpoint.Stream == nil {
			return fmt.Errorf("endpoint %q of type %q has no stream config", endpoint.Path, endpoint.Type)
		}
		r.Group(func(r chi.Router) {
			useEndpointMiddleware(r, endpoint, nil)
			r.HandleFunc(endpoint.Path, ws.Stream(cfg.Hostname, endpoint.Path, endpoint.Stream.Interval, webSocketObserver))
		})
	case config.EndpointTypeProxy:
		if endpoint.Backend == nil {
			return fmt.Errorf("endpoint %q of type %q has no backend config", endpoint.Path, endpoint.Type)
		}
		handler, err := httpendpoint.Proxy(endpoint.Path, *endpoint.Backend, proxyObserver)
		if err != nil {
			return fmt.Errorf("create proxy for endpoint %q: %w", endpoint.Path, err)
		}
		r.Group(func(r chi.Router) {
			useEndpointMiddleware(r, endpoint, httpMetricsMiddleware)
			r.Handle(endpoint.Path, handler)
			r.Handle(endpoint.Path+"/*", handler)
		})
	default:
		return fmt.Errorf("endpoint %q has unsupported type %q", endpoint.Path, endpoint.Type)
	}

	return nil
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
	r.Use(middleware.Decelerator(endpoint.Chaos))
	r.Use(middleware.ErrorInjector(endpoint.Chaos))
}
