package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Logger records completed requests using their configured endpoint identity.
func Logger(endpoint, endpointType string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			responseWriter := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				slog.Info("request handled",
					"endpoint", endpoint,
					"type", endpointType,
					"method", r.Method,
					"duration", time.Since(startTime).String(),
					"status", responseWriter.Status(),
					"size", responseWriter.BytesWritten(),
				)
			}()
			next.ServeHTTP(responseWriter, r)
		})
	}
}
