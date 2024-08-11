package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sergkondr/fake-web-service/internal/config"
)

func New(cfg config.Config) chi.Router {
	r := chi.NewRouter()

	prometheusMWHandler := dummyMWHandler
	if cfg.Metrics.Enabled {
		promRegistry := prometheus.NewRegistry()
		r.Handle(cfg.Metrics.Path, promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{
			Registry: promRegistry,
		}))
		prometheusMWHandler = newPrometheusMetrics(promRegistry)
	}

	var endpoints strings.Builder
	for _, endpoint := range cfg.HTTPEndpoints {
		r.Route(endpoint.Path, func(r chi.Router) {
			if !endpoint.DoNotLog {
				r.Use(logger())
			}
			r.Use(prometheusMWHandler) // if metrics are disabled, dummyMWHandler will be used
			r.Use(decelerator(endpoint))
			r.Use(errorInjector(endpoint))
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(fmt.Sprintf("success: %s%s\n", cfg.Hostname, endpoint.Path)))
			})
		})
		if !endpoint.Hidden {
			endpoints.WriteString(fmt.Sprintf("- %s - %s: %s\n", endpoint.Path, endpoint.Name, endpoint.Description))
		}
	}

	if len(cfg.WSEndpoints) != 0 {
		endpoints.WriteString("\n\n- /ws - WebSocket endpoint is available")
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Available endpoints:\n%s\nHostname: %s\n", endpoints.String(), cfg.Hostname)))
	})

	r.Route(wsURLPreffix, func(r chi.Router) {
		for _, endpoint := range cfg.WSEndpoints {
			if endpoint.Type == "echo" {
				r.HandleFunc(endpoint.Path, wsHandlerEcho(cfg.Hostname))
				r.HandleFunc("/", newWSRoot(endpoint.Path))
			}
		}
	})

	return r
}

func dummyMWHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
	}

	return http.HandlerFunc(fn)
}
