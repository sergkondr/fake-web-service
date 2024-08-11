package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sergkondr/fake-web-service/internal/config"
	"github.com/sergkondr/fake-web-service/internal/prometheusMetrics"
)

func New(cfg config.Config) chi.Router {
	r := chi.NewRouter()

	var prometheusMWHandler func(next http.Handler) http.Handler
	if cfg.Metrics.Enabled {
		metrics := prometheusMetrics.New("fakesvc")
		r.Handle(cfg.Metrics.Path, metrics.MetricsHandler())
		prometheusMWHandler = metrics.MiddlewareHandler
	}

	var endpoints strings.Builder
	for _, endpoint := range cfg.HTTPEndpoints {
		r.Route(endpoint.Path, func(r chi.Router) {
			if !endpoint.DoNotLog {
				r.Use(logger())
			}
			if cfg.Metrics.Enabled {
				r.Use(prometheusMWHandler)
			}
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
