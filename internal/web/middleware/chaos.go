// Package middleware provides common HTTP middleware for configured endpoints.
package middleware

import (
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/sergkondr/fake-web-service/internal/config"
)

// Chaos applies the configured delay and error injection to a request.
func Chaos(chaos config.Chaos) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if chaos.Latency != nil {
				delay := getDelay(chaos.Latency.Min, chaos.Latency.Max, chaos.Latency.P95)
				if !waitForDelay(r, delay) {
					return
				}
			}

			if rand.Float64() < chaos.ErrorRate {
				status := chaos.ErrorStatus
				if status == 0 {
					status = http.StatusInternalServerError
				}
				http.Error(w, http.StatusText(status), status)
				return
			}

			next.ServeHTTP(w, r)
		})
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

func getDelay(minimum, maximum, p95 time.Duration) time.Duration {
	if minimum == maximum {
		return minimum
	}

	if rand.N(100) >= 95 {
		if maximum > p95 {
			return p95 + rand.N(maximum-p95)
		}
		return p95
	}

	if p95 > minimum {
		return minimum + rand.N(p95-minimum)
	}

	return minimum
}
