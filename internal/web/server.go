package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sergkondr/fake-web-service/internal/config"
)

func New(cfg config.Config) chi.Router {
	r := chi.NewRouter()

	hostname, err := os.Hostname()
	if err != nil {
		slog.Error("could not get hostname:" + err.Error())
		hostname = "unknown"
	}

	var endpoints strings.Builder
	for _, endpoint := range cfg.HTTPEndpoints {
		if !endpoint.Hidden {
			endpoints.WriteString(fmt.Sprintf("- %s - %s: %s\n", endpoint.Path, endpoint.Name, endpoint.Description))
		}

		r.Route(endpoint.Path, func(r chi.Router) {
			if !endpoint.DoNotLog {
				r.Use(logger())
			}
			r.Use(decelerator(endpoint))
			r.Use(errorInjector(endpoint))
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(fmt.Sprintf("success: %s%s\n", hostname, endpoint.Path)))
			})
		})
	}

	if len(cfg.WSEndpoints) != 0 {
		endpoints.WriteString("\n\n- /ws - WebSocket endpoint is available")
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Available endpoints:\n%s\nHostname: %s\n", endpoints.String(), hostname)))
	})

	r.Route(wsURLPreffix, func(r chi.Router) {
		for _, endpoint := range cfg.WSEndpoints {
			if endpoint.Type == "echo" {
				r.HandleFunc(endpoint.Path, wsHandlerEcho(hostname))
				r.HandleFunc("/", newWSRoot(endpoint.Path))
			}
		}
	})

	return r
}
