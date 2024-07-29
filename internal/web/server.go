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

	var endpoints strings.Builder
	for _, endpoint := range cfg.HTTPEndpoints {
		endpoints.WriteString(fmt.Sprintf("- %s - %s: %s\n", endpoint.Path, endpoint.Name, endpoint.Description))

		r.Route(endpoint.Path, func(r chi.Router) {
			r.Use(logger())
			r.Use(decelerator(endpoint))
			r.Use(errorInjector(endpoint))
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(fmt.Sprintf("success: %s\n", endpoint.Path)))
			})
		})
	}

	hostname, err := os.Hostname()
	if err != nil {
		slog.Error("could not get hostname:" + err.Error())
		hostname = "unknown"
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Available endpoints:\n%s\nHostname: %s\n", endpoints.String(), hostname)))
	})

	return r
}
