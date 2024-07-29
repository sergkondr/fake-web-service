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
	minMsec := minT.Milliseconds()
	maxMsec := maxT.Milliseconds()
	p95Msec := p95T.Milliseconds()

	r := minMsec + rand.Int63n(p95Msec-minMsec)
	if rand.Intn(100) > 95 {
		// if this request did not get into the 95th percentile
		r = r + rand.Int63n(maxMsec-p95Msec)
	}

	return time.Duration(r) * time.Millisecond
}
