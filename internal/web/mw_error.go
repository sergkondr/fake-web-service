package web

import (
	"math/rand"
	"net/http"

	"github.com/sergkondr/fake-web-service/internal/config"
)

func errorInjector(chaos config.Chaos) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			errThreshold := 100 * chaos.ErrorRate
			if float64(rand.Intn(100)) < errThreshold {
				status := chaos.ErrorStatus
				if status == 0 {
					status = http.StatusInternalServerError
				}
				http.Error(w, http.StatusText(status), status)

				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
