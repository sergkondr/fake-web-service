// Package httpendpoint contains handlers for HTTP endpoint types.
package httpendpoint

import (
	"fmt"
	"net/http"

	"github.com/sergkondr/fake-web-service/internal/config"
)

// Static returns a configurable static HTTP response handler.
func Static(hostname string, endpoint config.Endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if endpoint.Response == nil {
			w.WriteHeader(http.StatusOK)
			if r.Method != http.MethodHead {
				_, _ = w.Write([]byte(fmt.Sprintf("success: %s%s\n", hostname, endpoint.Path)))
			}
			return
		}

		for name, value := range endpoint.Response.Headers {
			w.Header().Set(name, value)
		}
		status := endpoint.Response.Status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		if r.Method != http.MethodHead {
			_, _ = w.Write([]byte(endpoint.Response.Body))
		}
	}
}
