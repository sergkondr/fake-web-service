package web

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/sergkondr/fake-web-service/internal/config"
)

func decelerator(chaos config.Chaos) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if chaos.Latency == nil {
			return next
		}

		fn := func(w http.ResponseWriter, r *http.Request) {
			delay := getDelay(chaos.Latency.Min, chaos.Latency.Max, chaos.Latency.P95)
			if !waitForDelay(r, delay) {
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func waitForDelay(r *http.Request, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-r.Context().Done():
		return false
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
