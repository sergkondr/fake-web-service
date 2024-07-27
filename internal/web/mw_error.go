package web

import (
	"math/rand"
	"net/http"

	"github.com/sergkondr/fake-web-service/internal/config"
)

func errorInjector(cfg config.Endpoint) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			errThreshold := 100 * cfg.ErrorRate

			if float64(rand.Intn(100)) < errThreshold {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("sorry, something went wrong"))
				return
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
