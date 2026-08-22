package web

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/sergkondr/fake-web-service/internal/config"
)

func decelerator(cfg config.HTTPEndpoint) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			delay := getDelay(cfg.Slowness.Min, cfg.Slowness.Max, cfg.Slowness.P95)
			time.Sleep(delay)

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func getDelay(minT, maxT, p95T time.Duration) time.Duration {
	if minT == maxT {
		return minT
	}

	if rand.Intn(100) >= 95 {
		// If this request did not get into the 95th percentile, use the tail
		// of the configured distribution.
		if maxT > p95T {
			return p95T + time.Duration(rand.Int63n(int64(maxT-p95T)))
		}
		return p95T
	}

	if p95T > minT {
		return minT + time.Duration(rand.Int63n(int64(p95T-minT)))
	}

	return minT
}
