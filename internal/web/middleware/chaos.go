// Package middleware provides common HTTP middleware for configured endpoints.
package middleware

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/sergkondr/fake-web-service/internal/config"
)

// Decelerator delays a request according to the endpoint chaos configuration.
func Decelerator(chaos config.Chaos) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if chaos.Latency == nil {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			delay := getDelay(chaos.Latency.Min, chaos.Latency.Max, chaos.Latency.P95)
			if !waitForDelay(r, delay) {
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ErrorInjector returns a configured error for a share of requests.
func ErrorInjector(chaos config.Chaos) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if float64(rand.Intn(100)) < 100*chaos.ErrorRate {
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

	if rand.Intn(100) >= 95 {
		if maximum > p95 {
			return p95 + time.Duration(rand.Int63n(int64(maximum-p95)))
		}
		return p95
	}

	if p95 > minimum {
		return minimum + time.Duration(rand.Int63n(int64(p95-minimum)))
	}

	return minimum
}
