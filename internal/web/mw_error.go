package web

import (
	"math/rand"
	"net/http"

	"github.com/sergkondr/fake-web-service/internal/config"
)

func errorInjector(cfg config.HTTPEndpoint) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			errThreshold := 100 * cfg.ErrorRate
			if float64(rand.Intn(100)) < errThreshold {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
