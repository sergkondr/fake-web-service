package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sergkondr/fake-web-service/internal/config"
)

func New(cfg config.Config) chi.Router {
	r := chi.NewRouter()

	var endpoints strings.Builder
	for _, endpoint := range cfg.Endpoints {
		endpoints.WriteString(fmt.Sprintf("- %s - %s: %s\n", endpoint.Path, endpoint.Name, endpoint.Description))

		r.Route(endpoint.Path, func(r chi.Router) {
			r.Use(logger())
			r.Use(decelerator(endpoint))
			r.Use(errorInjector(endpoint))
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("you successfully requested " + endpoint.Path))
			})
		})
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Available endpoints:\n" + endpoints.String()))
	})

	return r
}
