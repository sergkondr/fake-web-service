package web

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func logger() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				// we need to wrap this in a function call because the value of ww.Status()
				// will be captured at the moment of defer, not after the execution of next.ServeHTTP(ww, r),
				// and it will always be 0 there.
				slog.Info("request handled",
					"uri", r.RequestURI,
					"method", r.Method,
					"duration", time.Since(startTime).String(),
					"status", ww.Status(),
					"size", ww.BytesWritten(),
				)
			}()
			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}
